package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"unicode"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	r.GET("/map", func(c *gin.Context) {
		bucket := c.Query("bucket")
		key := c.Query("key") // e.g., chunk1.txt

		cfg, _ := config.LoadDefaultConfig(context.TODO(), config.WithRegion("us-west-2"))
		s3Client := s3.NewFromConfig(cfg)

		// 1. Download the chunk
		obj, _ := s3Client.GetObject(context.TODO(), &s3.GetObjectInput{
			Bucket: &bucket,
			Key:    &key,
		})
		content, _ := io.ReadAll(obj.Body)

		// 2. Count words
		wordCounts := make(map[string]int)
		words := strings.FieldsFunc(string(content), func(r rune) bool {
			return !unicode.IsLetter(r) && !unicode.IsNumber(r)
		})

		for _, word := range words {
			word = strings.ToLower(word)
			wordCounts[word]++
		}

		// 3. Convert counts to JSON
		jsonData, _ := json.Marshal(wordCounts)

		// 4. Upload result to S3
		resultKey := fmt.Sprintf("map-%s.json", key)
		_, _ = s3Client.PutObject(context.TODO(), &s3.PutObjectInput{
			Bucket: &bucket,
			Key:    &resultKey,
			Body:   strings.NewReader(string(jsonData)),
		})

		c.JSON(http.StatusOK, gin.H{
			"result_s3": fmt.Sprintf("s3://%s/%s", bucket, resultKey),
		})
	})
	r.Run(":8080")
}