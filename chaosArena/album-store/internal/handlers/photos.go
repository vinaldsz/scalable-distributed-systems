package handlers

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vinaldsouza/album-store/internal/s3client"
	"github.com/vinaldsouza/album-store/internal/store"
	"github.com/vinaldsouza/album-store/internal/worker"
)

type PhotoHandler struct {
	photoStore *store.PhotoStore
	albumStore *store.AlbumStore
	s3         *s3client.Client
	pool       *worker.Pool
}

func NewPhotoHandler(ps *store.PhotoStore, as *store.AlbumStore, s3 *s3client.Client, pool *worker.Pool) *PhotoHandler {
	return &PhotoHandler{photoStore: ps, albumStore: as, s3: s3, pool: pool}
}

// Upload handles POST /albums/:album_id/photos
// 1. Verify album exists
// 2. Read multipart file into memory
// 3. Atomically assign seq and insert photo row (status=processing)
// 4. Submit S3 upload to worker pool
// 5. Return 202 immediately
func (h *PhotoHandler) Upload(c *gin.Context) {
	albumID := c.Param("album_id")

	// Verify album exists
	if _, err := h.albumStore.GetByID(c.Request.Context(), albumID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "album not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Parse multipart form (max 200 MB)
	if err := c.Request.ParseMultipartForm(200 << 20); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse form"})
		return
	}

	file, header, err := c.Request.FormFile("photo")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "photo field required"})
		return
	}
	defer file.Close()

	// Pre-allocate buffer using known file size — avoids repeated alloc+copy of io.ReadAll
	data := make([]byte, header.Size)
	if _, err := io.ReadFull(file, data); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read file"})
		return
	}

	photoID := uuid.New().String()
	s3Key := fmt.Sprintf("albums/%s/photos/%s", albumID, photoID)
	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// Atomically assign seq and insert photo row
	photo, err := h.photoStore.InsertWithSeq(c.Request.Context(), photoID, albumID, s3Key)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Submit background S3 upload to worker pool
	h.pool.Submit(worker.Job{
		PhotoID: photoID,
		AlbumID: albumID,
		S3Key:   s3Key,
		Data:    data,
		ProcessFn: func(ctx context.Context, job worker.Job) error {
			url, err := h.s3.Upload(ctx, job.S3Key, bytes.NewReader(job.Data), contentType)
			if err != nil {
				_ = h.photoStore.UpdateStatus(ctx, job.PhotoID, "failed", "")
				return err
			}
			if err := h.photoStore.UpdateStatus(ctx, job.PhotoID, "completed", url); err != nil {
				log.Printf("update status failed for %s: %v", job.PhotoID, err)
			}
			return nil
		},
	})

	c.JSON(http.StatusAccepted, gin.H{
		"photo_id": photo.PhotoID,
		"seq":      photo.Seq,
		"status":   photo.Status,
	})
}

// GetByID handles GET /albums/:album_id/photos/:photo_id
func (h *PhotoHandler) GetByID(c *gin.Context) {
	albumID := c.Param("album_id")
	photoID := c.Param("photo_id")

	photo, err := h.photoStore.GetByID(c.Request.Context(), photoID, albumID)
	if errors.Is(err, store.ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "photo not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, photo)
}

// Delete handles DELETE /albums/:album_id/photos/:photo_id
// Runs S3 delete and DB delete concurrently.
func (h *PhotoHandler) Delete(c *gin.Context) {
	albumID := c.Param("album_id")
	photoID := c.Param("photo_id")

	s3Key, err := h.photoStore.GetS3Key(c.Request.Context(), photoID)
	if errors.Is(err, store.ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "photo not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Run S3 delete and DB delete concurrently
	ctx, cancel := context.WithTimeout(c.Request.Context(), 4500*time.Millisecond)
	defer cancel()

	s3ErrCh := make(chan error, 1)
	dbErrCh := make(chan error, 1)

	go func() { s3ErrCh <- h.s3.Delete(ctx, s3Key) }()
	go func() { dbErrCh <- h.photoStore.Delete(ctx, photoID, albumID) }()

	s3Err := <-s3ErrCh
	dbErr := <-dbErrCh

	if dbErr != nil {
		if errors.Is(dbErr, store.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "photo not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": dbErr.Error()})
		return
	}
	if s3Err != nil {
		log.Printf("s3 delete failed for %s: %v", s3Key, s3Err)
	}

	c.Status(http.StatusNoContent)
}
