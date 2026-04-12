package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vinaldsouza/album-store/internal/config"
	"github.com/vinaldsouza/album-store/internal/db"
	"github.com/vinaldsouza/album-store/internal/handlers"
	"github.com/vinaldsouza/album-store/internal/s3client"
	"github.com/vinaldsouza/album-store/internal/store"
	"github.com/vinaldsouza/album-store/internal/worker"
)

func main() {
	cfg := config.Load()

	ctx := context.Background()
	pool, err := db.NewPool(ctx, cfg.DatabaseURL, cfg.DBMaxConns, cfg.DBMinConns)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer pool.Close()
	log.Println("DB connected")

	s3, err := s3client.New(ctx, cfg.AWSRegion, cfg.S3Bucket)
	if err != nil {
		log.Fatalf("s3 client: %v", err)
	}
	log.Println("S3 client ready")

	workerPool := worker.New(cfg.WorkerCount, cfg.WorkerQueueCap, cfg.MaxConcurrentUploads)

	albumStore := store.NewAlbumStore(pool)
	photoStore := store.NewPhotoStore(pool)

	albumHandler := handlers.NewAlbumHandler(albumStore)
	photoHandler := handlers.NewPhotoHandler(photoStore, albumStore, s3, workerPool)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.GET("/health", handlers.Health)
	r.PUT("/albums/:album_id", albumHandler.Upsert)
	r.GET("/albums/:album_id", albumHandler.GetByID)
	r.GET("/albums", albumHandler.List)
	r.POST("/albums/:album_id/photos", photoHandler.Upload)
	r.GET("/albums/:album_id/photos/:photo_id", photoHandler.GetByID)
	r.DELETE("/albums/:album_id/photos/:photo_id", photoHandler.Delete)

	srv := &http.Server{
		Addr:         ":" + cfg.AppPort,
		Handler:      r,
		ReadTimeout:  10 * time.Minute,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start server in background
	go func() {
		log.Printf("Server listening on :%s", cfg.AppPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	// Wait for SIGTERM or SIGINT
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down...")

	// Give active requests 30s to finish
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("server shutdown error: %v", err)
	}

	// Drain worker pool — wait for in-flight S3 uploads to finish
	workerPool.Shutdown()
	log.Println("Shutdown complete")
}
