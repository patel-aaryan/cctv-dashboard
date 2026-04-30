package internal

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Video struct {
	ID         int64
	CameraName string
	Timestamp  time.Time
	Filepath   string
}

const schema = `
CREATE TABLE IF NOT EXISTS videos (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    camera_name TEXT     NOT NULL,
    timestamp   DATETIME NOT NULL,
    filepath    TEXT     NOT NULL UNIQUE
);
CREATE INDEX IF NOT EXISTS idx_videos_timestamp ON videos(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_videos_camera    ON videos(camera_name);
`

func Open(path string) (*sql.DB, error) {
	dsn := fmt.Sprintf("%s?_busy_timeout=5000&_journal_mode=WAL&_foreign_keys=on", path)
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("apply schema: %w", err)
	}
	return db, nil
}

func insertVideo(db *sql.DB, v Video) (bool, error) {
	res, err := db.Exec(
		`INSERT OR IGNORE INTO videos (camera_name, timestamp, filepath) VALUES (?, ?, ?)`,
		v.CameraName, v.Timestamp, v.Filepath,
	)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func deleteVideoByPath(db *sql.DB, path string) error {
	_, err := db.Exec(`DELETE FROM videos WHERE filepath = ?`, path)
	return err
}

func ListVideos(db *sql.DB, limit int) ([]Video, error) {
	if limit <= 0 || limit > 1000 {
		limit = 500
	}
	rows, err := db.Query(
		`SELECT id, camera_name, timestamp, filepath
		   FROM videos
		   ORDER BY timestamp DESC
		   LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	videos := make([]Video, 0)
	for rows.Next() {
		var v Video
		if err := rows.Scan(&v.ID, &v.CameraName, &v.Timestamp, &v.Filepath); err != nil {
			return nil, err
		}
		videos = append(videos, v)
	}
	return videos, rows.Err()
}
