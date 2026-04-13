package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awscfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
)

var (
	ddb        *dynamodb.Client
	s3c        *s3.Client
	uploader   *manager.Uploader
	bucket     string
	region     string
	albumTable string
	photoTable string
)

// ---------- helpers ----------

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func ddbKey(name, val string) map[string]types.AttributeValue {
	return map[string]types.AttributeValue{name: &types.AttributeValueMemberS{Value: val}}
}

func str(m map[string]types.AttributeValue, k string) string {
	if v, ok := m[k].(*types.AttributeValueMemberS); ok {
		return v.Value
	}
	return ""
}

func num(m map[string]types.AttributeValue, k string) int {
	if v, ok := m[k].(*types.AttributeValueMemberN); ok {
		n, _ := strconv.Atoi(v.Value)
		return n
	}
	return 0
}

// ---------- handlers ----------

func health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, 200, map[string]string{"status": "ok"})
}

func putAlbum(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("album_id")
	var body struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Owner       string `json:"owner"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, 400, map[string]string{"error": "bad request"})
		return
	}

	out, err := ddb.UpdateItem(r.Context(), &dynamodb.UpdateItemInput{
		TableName:        &albumTable,
		Key:              ddbKey("album_id", id),
		UpdateExpression: aws.String("SET title = :t, description = :d, #o = :o"),
		ExpressionAttributeNames: map[string]string{
			"#o": "owner",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":t": &types.AttributeValueMemberS{Value: body.Title},
			":d": &types.AttributeValueMemberS{Value: body.Description},
			":o": &types.AttributeValueMemberS{Value: body.Owner},
		},
		ReturnValues: types.ReturnValueAllOld,
	})
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}

	code := 201
	if len(out.Attributes) > 0 {
		code = 200
	}
	writeJSON(w, code, map[string]string{
		"album_id":    id,
		"title":       body.Title,
		"description": body.Description,
		"owner":       body.Owner,
	})
}

func getAlbum(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("album_id")
	out, err := ddb.GetItem(r.Context(), &dynamodb.GetItemInput{
		TableName: &albumTable,
		Key:       ddbKey("album_id", id),
	})
	if err != nil || out.Item == nil {
		writeJSON(w, 404, map[string]string{"error": "not found"})
		return
	}
	writeJSON(w, 200, map[string]string{
		"album_id":    str(out.Item, "album_id"),
		"title":       str(out.Item, "title"),
		"description": str(out.Item, "description"),
		"owner":       str(out.Item, "owner"),
	})
}

func listAlbums(w http.ResponseWriter, r *http.Request) {
	var albums []map[string]string
	p := dynamodb.NewScanPaginator(ddb, &dynamodb.ScanInput{
		TableName:            &albumTable,
		ProjectionExpression: aws.String("album_id, title, description, #o"),
		ExpressionAttributeNames: map[string]string{
			"#o": "owner",
		},
	})
	for p.HasMorePages() {
		page, err := p.NextPage(r.Context())
		if err != nil {
			break
		}
		for _, item := range page.Items {
			albums = append(albums, map[string]string{
				"album_id":    str(item, "album_id"),
				"title":       str(item, "title"),
				"description": str(item, "description"),
				"owner":       str(item, "owner"),
			})
		}
	}
	if albums == nil {
		albums = []map[string]string{}
	}
	writeJSON(w, 200, albums)
}

func uploadPhoto(w http.ResponseWriter, r *http.Request) {
	albumID := r.PathValue("album_id")

	reader, err := r.MultipartReader()
	if err != nil {
		writeJSON(w, 400, map[string]string{"error": "missing photo"})
		return
	}
	var part *multipart.Part
	for {
		p, err := reader.NextPart()
		if err != nil {
			writeJSON(w, 400, map[string]string{"error": "missing photo"})
			return
		}
		if p.FormName() == "photo" {
			part = p
			break
		}
	}
	defer part.Close()

	photoID := uuid.New().String()
	s3Key := albumID + "/" + photoID

	// Atomic per-album seq increment
	seqOut, err := ddb.UpdateItem(r.Context(), &dynamodb.UpdateItemInput{
		TableName:        &albumTable,
		Key:              ddbKey("album_id", albumID),
		UpdateExpression: aws.String("SET photo_seq = if_not_exists(photo_seq, :z) + :one"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":z":   &types.AttributeValueMemberN{Value: "0"},
			":one": &types.AttributeValueMemberN{Value: "1"},
		},
		ReturnValues: types.ReturnValueUpdatedNew,
	})
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	seq := num(seqOut.Attributes, "photo_seq")

	// Save photo record as "processing"
	if _, err = ddb.PutItem(r.Context(), &dynamodb.PutItemInput{
		TableName: &photoTable,
		Item: map[string]types.AttributeValue{
			"photo_id": &types.AttributeValueMemberS{Value: photoID},
			"album_id": &types.AttributeValueMemberS{Value: albumID},
			"seq":      &types.AttributeValueMemberN{Value: strconv.Itoa(seq)},
			"status":   &types.AttributeValueMemberS{Value: "processing"},
			"s3_key":   &types.AttributeValueMemberS{Value: s3Key},
		},
	}); err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}

	// Read up to 5MB to determine small vs large
	const threshold = 5 * 1024 * 1024
	buf := make([]byte, threshold)
	n, readErr := io.ReadAtLeast(part, buf, 1)
	buf = buf[:n]

	if readErr == io.EOF || readErr == io.ErrUnexpectedEOF || n < threshold {
		// Small file — fully in buffer, return 202 instantly
		writeJSON(w, 202, map[string]any{
			"photo_id": photoID,
			"seq":      seq,
			"status":   "processing",
		})
		go func() {
			ctx := context.Background()
			_, s3Err := uploader.Upload(ctx, &s3.PutObjectInput{
				Bucket: &bucket,
				Key:    &s3Key,
				Body:   bytes.NewReader(buf),
			})
			if s3Err != nil {
				ddb.UpdateItem(ctx, &dynamodb.UpdateItemInput{
					TableName:                &photoTable,
					Key:                      ddbKey("photo_id", photoID),
					UpdateExpression:         aws.String("SET #s = :s"),
					ConditionExpression:      aws.String("attribute_exists(photo_id)"),
					ExpressionAttributeNames: map[string]string{"#s": "status"},
					ExpressionAttributeValues: map[string]types.AttributeValue{
						":s": &types.AttributeValueMemberS{Value: "failed"},
					},
				})
				return
			}
			url := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", bucket, region, s3Key)
			_, err := ddb.UpdateItem(ctx, &dynamodb.UpdateItemInput{
				TableName:                &photoTable,
				Key:                      ddbKey("photo_id", photoID),
				UpdateExpression:         aws.String("SET #s = :s, #u = :u"),
				ConditionExpression:      aws.String("attribute_exists(photo_id)"),
				ExpressionAttributeNames: map[string]string{"#s": "status", "#u": "url"},
				ExpressionAttributeValues: map[string]types.AttributeValue{
					":s": &types.AttributeValueMemberS{Value: "completed"},
					":u": &types.AttributeValueMemberS{Value: url},
				},
			})
			if err != nil {
				s3c.DeleteObject(ctx, &s3.DeleteObjectInput{
					Bucket: &bucket,
					Key:    &s3Key,
				})
			}
		}()
	} else {
		// Large file — stream remainder directly to S3
		combined := io.MultiReader(bytes.NewReader(buf), part)
		_, s3Err := uploader.Upload(r.Context(), &s3.PutObjectInput{
			Bucket: &bucket,
			Key:    &s3Key,
			Body:   combined,
		})
		go func() {
			ctx := context.Background()
			if s3Err != nil {
				ddb.UpdateItem(ctx, &dynamodb.UpdateItemInput{
					TableName:                &photoTable,
					Key:                      ddbKey("photo_id", photoID),
					UpdateExpression:         aws.String("SET #s = :s"),
					ConditionExpression:      aws.String("attribute_exists(photo_id)"),
					ExpressionAttributeNames: map[string]string{"#s": "status"},
					ExpressionAttributeValues: map[string]types.AttributeValue{
						":s": &types.AttributeValueMemberS{Value: "failed"},
					},
				})
				return
			}
			url := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", bucket, region, s3Key)
			_, err := ddb.UpdateItem(ctx, &dynamodb.UpdateItemInput{
				TableName:                &photoTable,
				Key:                      ddbKey("photo_id", photoID),
				UpdateExpression:         aws.String("SET #s = :s, #u = :u"),
				ConditionExpression:      aws.String("attribute_exists(photo_id)"),
				ExpressionAttributeNames: map[string]string{"#s": "status", "#u": "url"},
				ExpressionAttributeValues: map[string]types.AttributeValue{
					":s": &types.AttributeValueMemberS{Value: "completed"},
					":u": &types.AttributeValueMemberS{Value: url},
				},
			})
			if err != nil {
				s3c.DeleteObject(ctx, &s3.DeleteObjectInput{
					Bucket: &bucket,
					Key:    &s3Key,
				})
			}
		}()
		writeJSON(w, 202, map[string]any{
			"photo_id": photoID,
			"seq":      seq,
			"status":   "processing",
		})
	}
}

func getPhoto(w http.ResponseWriter, r *http.Request) {
	photoID := r.PathValue("photo_id")
	out, err := ddb.GetItem(r.Context(), &dynamodb.GetItemInput{
		TableName: &photoTable,
		Key:       ddbKey("photo_id", photoID),
	})
	if err != nil || out.Item == nil {
		writeJSON(w, 404, map[string]string{"error": "not found"})
		return
	}
	resp := map[string]any{
		"photo_id": str(out.Item, "photo_id"),
		"album_id": str(out.Item, "album_id"),
		"seq":      num(out.Item, "seq"),
		"status":   str(out.Item, "status"),
	}
	if u := str(out.Item, "url"); u != "" {
		resp["url"] = u
	}
	writeJSON(w, 200, resp)
}

func deletePhoto(w http.ResponseWriter, r *http.Request) {
	photoID := r.PathValue("photo_id")

	out, err := ddb.DeleteItem(r.Context(), &dynamodb.DeleteItemInput{
		TableName:    &photoTable,
		Key:          ddbKey("photo_id", photoID),
		ReturnValues: types.ReturnValueAllOld,
	})
	if err != nil || len(out.Attributes) == 0 {
		writeJSON(w, 404, map[string]string{"error": "not found"})
		return
	}

	if key := str(out.Attributes, "s3_key"); key != "" {
		s3c.DeleteObject(r.Context(), &s3.DeleteObjectInput{
			Bucket: &bucket,
			Key:    &key,
		})
	}

	w.WriteHeader(200)
}

// ---------- logging middleware ----------

type statusWriter struct {
	http.ResponseWriter
	code int
}

func (w *statusWriter) WriteHeader(code int) {
	w.code = code
	w.ResponseWriter.WriteHeader(code)
}

func logWrap(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sw := &statusWriter{ResponseWriter: w, code: 200}
		start := time.Now()
		h.ServeHTTP(sw, r)
		log.Printf("%s %s → %d (%v)", r.Method, r.URL.Path, sw.code, time.Since(start).Round(time.Millisecond))
	})
}

// ---------- main ----------

func main() {
	region = env("AWS_REGION", "us-west-2")
	bucket = env("S3_BUCKET", "album-store-photos")
	albumTable = env("ALBUM_TABLE", "albums")
	photoTable = env("PHOTO_TABLE", "photos")

	cfg, err := awscfg.LoadDefaultConfig(context.TODO(),
		awscfg.WithRegion(region),
		awscfg.WithHTTPClient(&http.Client{
			Transport: &http.Transport{
				MaxIdleConns:        300,
				MaxIdleConnsPerHost: 100,
				IdleConnTimeout:     90 * time.Second,
			},
		}),
	)
	if err != nil {
		log.Fatal(err)
	}
	ddb = dynamodb.NewFromConfig(cfg)
	s3c = s3.NewFromConfig(cfg)
	uploader = manager.NewUploader(s3c)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", health)
	mux.HandleFunc("PUT /albums/{album_id}", putAlbum)
	mux.HandleFunc("GET /albums/{album_id}", getAlbum)
	mux.HandleFunc("GET /albums", listAlbums)
	mux.HandleFunc("POST /albums/{album_id}/photos", uploadPhoto)
	mux.HandleFunc("GET /albums/{album_id}/photos/{photo_id}", getPhoto)
	mux.HandleFunc("DELETE /albums/{album_id}/photos/{photo_id}", deletePhoto)

	port := env("PORT", "80")
	log.Printf("listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, logWrap(mux)))
}
