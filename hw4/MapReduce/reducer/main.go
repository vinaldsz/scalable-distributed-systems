package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gin-gonic/gin"
)

type ReduceRequest struct {
    MapResultURLs []string `json:"map_result_urls" binding:"required"`
}

type ReduceResponse struct {
    FinalResultURL string `json:"final_result_url"`
}

func main() {
    router := gin.Default()
    
    router.POST("/reduce", func(c *gin.Context) {
        var req ReduceRequest
        if err := c.BindJSON(&req); err != nil {
            c.JSON(400, gin.H{"error": err.Error()})
            return
        }
        
        cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("us-west-2"))
        if err != nil {
            c.JSON(500, gin.H{"error": "Failed to load AWS config: " + err.Error()})
            return
        }
        client := s3.NewFromConfig(cfg)
        
        // Aggregate word counts from all mappers
        finalCount := make(map[string]int)
        
        for _, url := range req.MapResultURLs {
            // Parse S3 URL
            parts := strings.Split(url, "/")
            bucket := parts[2]
            key := strings.Join(parts[3:], "/")
            
            // Download mapper result
            result, err := client.GetObject(context.TODO(), &s3.GetObjectInput{
                Bucket: &bucket,
                Key:    &key,
            })
            if err != nil {
                c.JSON(500, gin.H{"error": "Failed to get S3 object: " + err.Error()})
                return
            }
            
            content, err := io.ReadAll(result.Body)
            result.Body.Close()
            if err != nil {
                c.JSON(500, gin.H{"error": "Failed to read object body: " + err.Error()})
                return
            }
            
            // Parse JSON
            var wordCount map[string]int
            if err := json.Unmarshal(content, &wordCount); err != nil {
                c.JSON(500, gin.H{"error": "Failed to parse JSON: " + err.Error()})
                return
            }
            
            // Aggregate
            for word, count := range wordCount {
                finalCount[word] += count
            }
        }
        
        // Convert to JSON
        resultJSON, err := json.Marshal(finalCount)
        if err != nil {
            c.JSON(500, gin.H{"error": "Failed to marshal result: " + err.Error()})
            return
        }
        
        // Upload final result
        bucket := strings.Split(req.MapResultURLs[0], "/")[2]
        finalKey := "results/final_result.json"
        
        _, err = client.PutObject(context.TODO(), &s3.PutObjectInput{
            Bucket: &bucket,
            Key:    &finalKey,
            Body:   strings.NewReader(string(resultJSON)),
        })
        if err != nil {
            c.JSON(500, gin.H{"error": "Failed to upload result: " + err.Error()})
            return
        }
        
        finalURL := fmt.Sprintf("s3://%s/%s", bucket, finalKey)
        c.JSON(200, ReduceResponse{FinalResultURL: finalURL})
    })
    
    router.Run(":8080")
}