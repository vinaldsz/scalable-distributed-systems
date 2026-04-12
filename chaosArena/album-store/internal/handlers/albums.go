package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vinaldsouza/album-store/internal/models"
	"github.com/vinaldsouza/album-store/internal/store"
)

type AlbumHandler struct {
	store *store.AlbumStore
}

func NewAlbumHandler(s *store.AlbumStore) *AlbumHandler {
	return &AlbumHandler{store: s}
}

func (h *AlbumHandler) Upsert(c *gin.Context) {
	albumID := c.Param("album_id")

	var body models.Album
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	body.AlbumID = albumID

	album, err := h.store.Upsert(c.Request.Context(), body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, album)
}

func (h *AlbumHandler) GetByID(c *gin.Context) {
	albumID := c.Param("album_id")

	album, err := h.store.GetByID(c.Request.Context(), albumID)
	if errors.Is(err, store.ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "album not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, album)
}

func (h *AlbumHandler) List(c *gin.Context) {
	albums, err := h.store.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, albums)
}
