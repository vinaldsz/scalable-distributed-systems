package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gin-gonic/gin"
)

type MapRequest struct {
    ChunkURL string `json:"chunk_url" binding:"required"`
}

type MapResponse struct {
    ResultURL string `json:"result_url"`
}

func main() {
    router := gin.Default()
    
    router.POST("/map", func(c *gin.Context) {
        var req MapRequest
        if err := c.BindJSON(&req); err != nil {
            c.JSON(400, gin.H{"error": err.Error()})
            return
        }
        
        // Parse S3 URL
        parts := strings.Split(req.ChunkURL, "/")
        bucket := parts[2]
        key := strings.Join(parts[3:], "/")
        
        // Initialize S3 client
        cfg, _ := config.LoadDefaultConfig(context.TODO(), config.WithRegion("us-west-2"))
        client := s3.NewFromConfig(cfg)
        
        // Download chunk from S3
        result, err := client.GetObject(context.TODO(), &s3.GetObjectInput{
            Bucket: &bucket,
            Key:    &key,
        })
        if err != nil {
            c.JSON(500, gin.H{"error": "Failed to download chunk"})
            return
        }
        defer result.Body.Close()
        
        // Read content
        content, _ := io.ReadAll(result.Body)
        text := string(content)
        
        // Count words
        wordCount := make(map[string]int)
        re := regexp.MustCompile(`\w+`)
        words := re.FindAllString(strings.ToLower(text), -1)
        
        for _, word := range words {
            wordCount[word]++
        }
        
        // Convert to JSON
        resultJSON, _ := json.Marshal(wordCount)
        
        // Upload result to S3
        resultKey := strings.Replace(key, "chunks/", "results/map_", 1)
        resultKey = strings.Replace(resultKey, ".txt", ".json", 1)
        
        _, err = client.PutObject(context.TODO(), &s3.PutObjectInput{
            Bucket: &bucket,
            Key:    &resultKey,
            Body:   strings.NewReader(string(resultJSON)),
        })
        
        resultURL := fmt.Sprintf("s3://%s/%s", bucket, resultKey)
        c.JSON(200, MapResponse{ResultURL: resultURL})
    })
    
    router.Run(":8080")
}