CREATE TABLE IF NOT EXISTS albums (
    album_id    TEXT PRIMARY KEY,
    title       TEXT NOT NULL,
    description TEXT NOT NULL,
    owner       TEXT NOT NULL,
    photo_seq   BIGINT NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS photos (
    photo_id  TEXT PRIMARY KEY,
    album_id  TEXT NOT NULL REFERENCES albums(album_id),
    seq       BIGINT NOT NULL,
    status    TEXT NOT NULL DEFAULT 'processing',
    s3_key    TEXT,
    url       TEXT,
    UNIQUE (album_id, seq)
);

CREATE INDEX IF NOT EXISTS idx_photos_album_id ON photos(album_id);
