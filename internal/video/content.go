package video

import (
	"io/ioutil"
	"os"
	"path/filepath"
)

// VideoContentService defines the interface for video content storage
type VideoContentService interface {
	// Write saves video content to storage
	Write(videoId string, filename string, data []byte) error
	// Read retrieves video content from storage
	Read(videoId string) ([]byte, error)
	// Delete removes video content from storage
	Delete(videoId string) error
	// List returns a list of all video IDs in storage
	List() ([]string, error)
	// GetManifestPath returns the path to the DASH manifest file
	GetManifestPath(videoId string) string
}

// FSVideoContentService implements VideoContentService using the filesystem
type FSVideoContentService struct {
	baseDir string
}

// NewFSVideoContentService creates a new filesystem-based video content service
func NewFSVideoContentService(baseDir string) (*FSVideoContentService, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, err
	}
	return &FSVideoContentService{baseDir: baseDir}, nil
}

// Write saves video content to the filesystem
func (s *FSVideoContentService) Write(videoId string, filename string, data []byte) error {
	// This method is no longer used directly as we now use DASH format
	return nil
}

// Read retrieves video content from the filesystem
func (s *FSVideoContentService) Read(videoId string) ([]byte, error) {
	manifestPath := s.GetManifestPath(videoId)
	return ioutil.ReadFile(manifestPath)
}

// Delete removes video content from the filesystem
func (s *FSVideoContentService) Delete(videoId string) error {
	videoDir := filepath.Join(s.baseDir, videoId)
	return os.RemoveAll(videoDir)
}

// List returns a list of all video IDs in storage
func (s *FSVideoContentService) List() ([]string, error) {
	entries, err := ioutil.ReadDir(s.baseDir)
	if err != nil {
		return nil, err
	}

	var videoIds []string
	for _, entry := range entries {
		if entry.IsDir() && entry.Name() != "temp" {
			// Check if the directory contains a manifest file
			manifestPath := filepath.Join(s.baseDir, entry.Name(), "manifest.mpd")
			if _, err := os.Stat(manifestPath); err == nil {
				videoIds = append(videoIds, entry.Name())
			}
		}
	}
	return videoIds, nil
}

// GetManifestPath returns the path to the DASH manifest file
func (s *FSVideoContentService) GetManifestPath(videoId string) string {
	return filepath.Join(s.baseDir, videoId, "manifest.mpd")
}
