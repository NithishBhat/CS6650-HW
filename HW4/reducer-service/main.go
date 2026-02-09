package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	r.GET("/reduce", func(c *gin.Context) {
		bucket := c.Query("bucket")
		// This captures all "keys" parameters from the URL
		keys := c.QueryArray("keys") 

		if len(keys) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "No keys provided"})
			return
		}

		cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("us-west-2"))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load AWS config"})
			return
		}
		s3Client := s3.NewFromConfig(cfg)

		finalCounts := make(map[string]int)

		// 1. Process each map result file
		for _, key := range keys {
			obj, err := s3Client.GetObject(context.TODO(), &s3.GetObjectInput{
				Bucket: &bucket,
				Key:    &key,
			})
			if err != nil {
				fmt.Printf("Error downloading key %s: %v\n", key, err)
				continue
			}

			content, _ := io.ReadAll(obj.Body)
			obj.Body.Close()

			var counts map[string]int
			if err := json.Unmarshal(content, &counts); err != nil {
				fmt.Printf("Error unmarshaling JSON for key %s: %v\n", key, err)
				continue
			}

			// 2. Aggregate the counts
			for word, count := range counts {
				finalCounts[word] += count
			}
		}

		// 3. Convert final aggregation to JSON
		finalData, _ := json.Marshal(finalCounts)
		resultKey := "final-word-count.json"

		// 4. Upload the final result to S3
		_, err = s3Client.PutObject(context.TODO(), &s3.PutObjectInput{
			Bucket: &bucket,
			Key:    &resultKey,
			Body:   bytes.NewReader(finalData),
		})

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload final result"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":          "success",
			"final_result_s3": fmt.Sprintf("s3://%s/%s", bucket, resultKey),
		})
	})

	r.Run(":8080")
}