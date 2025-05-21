// Lab 7: Implement a local filesystem video content service

package web

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

// FSVideoContentService implements VideoContentService using the local filesystem.
type FSVideoContentService struct {
	baseDir string
}

// Uncomment the following line to ensure FSVideoContentService implements VideoContentService
// var _ VideoContentService = (*FSVideoContentService)(nil)

func NewFSVideoContentService(baseDir string) (*FSVideoContentService, error) {
	// Create base directory if it doesn't exist
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base directory: %v", err)
	}
	return &FSVideoContentService{baseDir: baseDir}, nil
}

func (s *FSVideoContentService) Read(videoId string, filename string) ([]byte, error) {
	path := filepath.Join(s.baseDir, videoId, filename)
	data, err := ioutil.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	return data, nil
}

func (s *FSVideoContentService) Write(videoId string, filename string, data []byte) error {
	// Create video directory if it doesn't exist
	videoDir := filepath.Join(s.baseDir, videoId)
	if err := os.MkdirAll(videoDir, 0755); err != nil {
		return fmt.Errorf("failed to create video directory: %v", err)
	}

	// Write the file
	path := filepath.Join(videoDir, filename)
	return ioutil.WriteFile(path, data, 0644)
}
