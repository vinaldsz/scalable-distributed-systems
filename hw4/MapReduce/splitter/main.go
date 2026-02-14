package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gin-gonic/gin"
)

type SplitRequest struct {
    InputURL string `json:"input_url" binding:"required"`
    NumChunks int   `json:"num_chunks" binding:"required"`
}

type SplitResponse struct {
    ChunkURLs []string `json:"chunk_urls"`
}

func main() {
    router := gin.Default()
    
    router.POST("/split", func(c *gin.Context) {
        var req SplitRequest
        if err := c.BindJSON(&req); err != nil {
            c.JSON(400, gin.H{"error": err.Error()})
            return
        }
        
        // Parse S3 URL
        parts := strings.Split(req.InputURL, "/")
        bucket := parts[2]
        key := strings.Join(parts[3:], "/")
        
        // Initialize S3 client
        cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("us-west-2"))
        if err != nil {
            c.JSON(500, gin.H{"error": "Failed to load AWS config: " + err.Error()})
            return
        }
        client := s3.NewFromConfig(cfg)
        
        // Download file from S3
        result, err := client.GetObject(context.TODO(), &s3.GetObjectInput{
            Bucket: &bucket,
            Key:    &key,
        })
        if err != nil {
            c.JSON(500, gin.H{"error": "Failed to download from S3"})
            return
        }
        defer result.Body.Close()
        
        // Read content
        content, err := io.ReadAll(result.Body)
        if err != nil {
            c.JSON(500, gin.H{"error": "Failed to read file content: " + err.Error()})
            return
        }
        text := string(content)
        
        // Split into chunks
        chunkSize := len(text) / req.NumChunks
        var chunkURLs []string
        
        start := 0
        for i := 0; i < req.NumChunks; i++ {
            var end int
            if i == req.NumChunks-1 {
                // Last chunk gets everything remaining
                end = len(text)
            } else {
                // Find approximate end position
                end = start + chunkSize
                
                // Move forward to the next whitespace to avoid splitting words
                for end < len(text) && !isWhitespace(text[end]) {
                    end++
                }
                // Skip the whitespace itself
                for end < len(text) && isWhitespace(text[end]) {
                    end++
                }
            }
            
            chunk := text[start:end]
            chunkKey := fmt.Sprintf("chunks/chunk_%d.txt", i+1)
            
            // Upload chunk to S3
            _, err := client.PutObject(context.TODO(), &s3.PutObjectInput{
                Bucket: &bucket,
                Key:    &chunkKey,
                Body:   bytes.NewReader([]byte(chunk)),
            })
            if err != nil {
                c.JSON(500, gin.H{"error": "Failed to upload chunk"})
                return
            }
            
            chunkURL := fmt.Sprintf("s3://%s/%s", bucket, chunkKey)
            chunkURLs = append(chunkURLs, chunkURL)
            
            // Update start for next chunk
            start = end
        }
        
        c.JSON(200, SplitResponse{ChunkURLs: chunkURLs})
    })
    
    router.Run(":8080")
}

// isWhitespace checks if a byte is a whitespace character
func isWhitespace(b byte) bool {
    return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}