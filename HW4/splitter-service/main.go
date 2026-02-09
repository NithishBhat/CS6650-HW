package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	r.GET("/split", func(c *gin.Context) {
		bucket := c.Query("bucket")
		key := c.Query("key")

		cfg, _ := config.LoadDefaultConfig(context.TODO(), config.WithRegion("us-west-2"))
		s3Client := s3.NewFromConfig(cfg)

		obj, err := s3Client.GetObject(context.TODO(), &s3.GetObjectInput{
			Bucket: &bucket,
			Key:    &key,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		content, _ := io.ReadAll(obj.Body)

		chunkSize := len(content) / 3
		var chunkUrls []string

		for i := 0; i < 3; i++ {
			start := i * chunkSize
			end := (i + 1) * chunkSize
			if i == 2 {
				end = len(content)
			}

			chunkKey := fmt.Sprintf("chunk%d.txt", i+1)
			_, _ = s3Client.PutObject(context.TODO(), &s3.PutObjectInput{
				Bucket: &bucket,
				Key:    &chunkKey,
				Body:   bytes.NewReader(content[start:end]),
			})
			chunkUrls = append(chunkUrls, fmt.Sprintf("s3://%s/%s", bucket, chunkKey))
		}

		c.JSON(http.StatusOK, gin.H{"chunks": chunkUrls})
	})
	r.Run(":8080")
}