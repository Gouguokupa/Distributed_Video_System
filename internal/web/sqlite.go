// Lab 7: Implement a SQLite video metadata service

package web

import (
	"database/sql"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type SQLiteVideoMetadataService struct {
	db *sql.DB
}

func NewSQLiteVideoMetadataService(dbPath string) (*SQLiteVideoMetadataService, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	// Create videos table if it doesn't exist
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS videos (
			id TEXT PRIMARY KEY,
			uploaded_at DATETIME NOT NULL
		)
	`)
	if err != nil {
		db.Close()
		return nil, err
	}

	return &SQLiteVideoMetadataService{db: db}, nil
}

func (s *SQLiteVideoMetadataService) Read(id string) (*VideoMetadata, error) {
	var metadata VideoMetadata
	err := s.db.QueryRow("SELECT id, uploaded_at FROM videos WHERE id = ?", id).
		Scan(&metadata.Id, &metadata.UploadedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &metadata, nil
}

func (s *SQLiteVideoMetadataService) List() ([]VideoMetadata, error) {
	rows, err := s.db.Query("SELECT id, uploaded_at FROM videos ORDER BY uploaded_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var videos []VideoMetadata
	for rows.Next() {
		var video VideoMetadata
		if err := rows.Scan(&video.Id, &video.UploadedAt); err != nil {
			return nil, err
		}
		videos = append(videos, video)
	}
	return videos, rows.Err()
}

func (s *SQLiteVideoMetadataService) Create(videoId string, uploadedAt time.Time) error {
	_, err := s.db.Exec("INSERT INTO videos (id, uploaded_at) VALUES (?, ?)",
		videoId, uploadedAt)
	return err
}

// Close closes the database connection
func (s *SQLiteVideoMetadataService) Close() error {
	return s.db.Close()
}

var _ VideoMetadataService = (*SQLiteVideoMetadataService)(nil)
