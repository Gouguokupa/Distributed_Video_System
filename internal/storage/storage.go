// Lab 8: Implement a network video content service (server)

package storage

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	pb "tritontube/internal/proto"
)

type StorageServer struct {
	pb.UnimplementedStorageServiceServer
	storageDir string
}

func NewStorageServer(storageDir string) (*StorageServer, error) {
	// Create storage directory if it doesn't exist
	if err := os.MkdirAll(storageDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %v", err)
	}
	return &StorageServer{storageDir: storageDir}, nil
}

func (s *StorageServer) getFilePath(videoID, filename string) string {
	return filepath.Join(s.storageDir, videoID, filename)
}

func (s *StorageServer) Read(ctx context.Context, req *pb.ReadRequest) (*pb.ReadResponse, error) {
	filePath := s.getFilePath(req.VideoId, req.Filename)
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %v", err)
	}
	return &pb.ReadResponse{Content: content}, nil
}

func (s *StorageServer) Write(ctx context.Context, req *pb.WriteRequest) (*pb.WriteResponse, error) {
	filePath := s.getFilePath(req.VideoId, req.Filename)

	// Create directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %v", err)
	}

	// Write file
	if err := ioutil.WriteFile(filePath, req.Content, 0644); err != nil {
		return nil, fmt.Errorf("failed to write file: %v", err)
	}

	return &pb.WriteResponse{Success: true}, nil
}

func (s *StorageServer) Delete(ctx context.Context, req *pb.DeleteRequest) (*pb.DeleteResponse, error) {
	filePath := s.getFilePath(req.VideoId, req.Filename)

	if err := os.Remove(filePath); err != nil {
		return nil, fmt.Errorf("failed to delete file: %v", err)
	}

	// Try to remove the video directory if it's empty
	videoDir := filepath.Dir(filePath)
	if err := os.Remove(videoDir); err != nil && !os.IsNotExist(err) {
		// Ignore error if directory is not empty
	}

	return &pb.DeleteResponse{Success: true}, nil
}

func (s *StorageServer) ListVideoIDs(ctx context.Context, req *pb.ListVideoIDsRequest) (*pb.ListVideoIDsResponse, error) {
	// Read all directories in the storage directory
	entries, err := ioutil.ReadDir(s.storageDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read storage directory: %v", err)
	}

	// Collect all directory names as video IDs
	videoIDs := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			videoIDs = append(videoIDs, entry.Name())
		}
	}

	return &pb.ListVideoIDsResponse{VideoIds: videoIDs}, nil
}

func (s *StorageServer) ListFiles(ctx context.Context, req *pb.ListFilesRequest) (*pb.ListFilesResponse, error) {
	videoDir := filepath.Join(s.storageDir, req.VideoId)

	// Check if video directory exists
	if _, err := os.Stat(videoDir); os.IsNotExist(err) {
		return &pb.ListFilesResponse{Filenames: []string{}}, nil
	}

	// Read all files in the video directory
	entries, err := ioutil.ReadDir(videoDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read video directory: %v", err)
	}

	// Collect all file names
	filenames := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			filenames = append(filenames, entry.Name())
		}
	}

	return &pb.ListFilesResponse{Filenames: filenames}, nil
}
