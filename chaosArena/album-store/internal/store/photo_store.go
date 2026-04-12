package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vinaldsouza/album-store/internal/models"
)

type PhotoStore struct {
	pool *pgxpool.Pool
}

func NewPhotoStore(pool *pgxpool.Pool) *PhotoStore {
	return &PhotoStore{pool: pool}
}

// InsertWithSeq atomically increments the album's photo_seq and inserts the photo
// in a single transaction. Returns the photo with its assigned seq number.
func (s *PhotoStore) InsertWithSeq(ctx context.Context, photoID, albumID, s3Key string) (models.Photo, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return models.Photo{}, err
	}
	defer tx.Rollback(ctx)

	var seq int64
	err = tx.QueryRow(ctx, `
		UPDATE albums SET photo_seq = photo_seq + 1 WHERE album_id = $1 RETURNING photo_seq
	`, albumID).Scan(&seq)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.Photo{}, ErrNotFound
	}
	if err != nil {
		return models.Photo{}, err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO photos (photo_id, album_id, seq, status, s3_key)
		VALUES ($1, $2, $3, 'processing', $4)
	`, photoID, albumID, seq, s3Key)
	if err != nil {
		return models.Photo{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return models.Photo{}, err
	}

	return models.Photo{
		PhotoID: photoID,
		AlbumID: albumID,
		Seq:     seq,
		Status:  "processing",
	}, nil
}

func (s *PhotoStore) GetByID(ctx context.Context, photoID, albumID string) (models.Photo, error) {
	var p models.Photo
	var url *string
	err := s.pool.QueryRow(ctx, `
		SELECT photo_id, album_id, seq, status, url FROM photos
		WHERE photo_id = $1 AND album_id = $2
	`, photoID, albumID).Scan(&p.PhotoID, &p.AlbumID, &p.Seq, &p.Status, &url)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.Photo{}, ErrNotFound
	}
	if err != nil {
		return models.Photo{}, err
	}
	if url != nil {
		p.URL = *url
	}
	return p, nil
}

func (s *PhotoStore) UpdateStatus(ctx context.Context, photoID, status, url string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE photos SET status = $1, url = $2 WHERE photo_id = $3
	`, status, url, photoID)
	return err
}

func (s *PhotoStore) GetS3Key(ctx context.Context, photoID string) (string, error) {
	var s3Key string
	err := s.pool.QueryRow(ctx, `
		SELECT s3_key FROM photos WHERE photo_id = $1
	`, photoID).Scan(&s3Key)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return s3Key, err
}

func (s *PhotoStore) Delete(ctx context.Context, photoID, albumID string) error {
	tag, err := s.pool.Exec(ctx, `
		DELETE FROM photos WHERE photo_id = $1 AND album_id = $2
	`, photoID, albumID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
