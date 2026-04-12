package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vinaldsouza/album-store/internal/models"
)

var ErrNotFound = errors.New("not found")

type AlbumStore struct {
	pool *pgxpool.Pool
}

func NewAlbumStore(pool *pgxpool.Pool) *AlbumStore {
	return &AlbumStore{pool: pool}
}

func (s *AlbumStore) Upsert(ctx context.Context, a models.Album) (models.Album, error) {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO albums (album_id, title, description, owner)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (album_id) DO UPDATE
		  SET title = EXCLUDED.title,
		      description = EXCLUDED.description,
		      owner = EXCLUDED.owner
	`, a.AlbumID, a.Title, a.Description, a.Owner)
	if err != nil {
		return models.Album{}, err
	}
	return a, nil
}

func (s *AlbumStore) GetByID(ctx context.Context, albumID string) (models.Album, error) {
	var a models.Album
	err := s.pool.QueryRow(ctx, `
		SELECT album_id, title, description, owner FROM albums WHERE album_id = $1
	`, albumID).Scan(&a.AlbumID, &a.Title, &a.Description, &a.Owner)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.Album{}, ErrNotFound
	}
	return a, err
}

func (s *AlbumStore) List(ctx context.Context) ([]models.Album, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT album_id, title, description, owner FROM albums ORDER BY album_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var albums []models.Album
	for rows.Next() {
		var a models.Album
		if err := rows.Scan(&a.AlbumID, &a.Title, &a.Description, &a.Owner); err != nil {
			return nil, err
		}
		albums = append(albums, a)
	}
	if albums == nil {
		albums = []models.Album{}
	}
	return albums, rows.Err()
}
