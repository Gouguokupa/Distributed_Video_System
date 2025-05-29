// Lab 7: Implement a web server

package web

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"tritontube/internal/proto"

	"google.golang.org/grpc"
)

type server struct {
	Addr string
	Port int

	metadataService VideoMetadataService
	contentService  VideoContentService

	mux        *http.ServeMux
	grpcServer *grpc.Server
}

func NewServer(
	metadataService VideoMetadataService,
	contentService VideoContentService,
) *server {
	return &server{
		metadataService: metadataService,
		contentService:  contentService,
		grpcServer:      grpc.NewServer(),
	}
}

func (s *server) Start(lis net.Listener) error {
	// Start HTTP server
	s.mux = http.NewServeMux()
	s.mux.HandleFunc("/upload", s.handleUpload)
	s.mux.HandleFunc("/videos/", s.handleVideo)
	s.mux.HandleFunc("/content/", s.handleVideoContent)
	s.mux.HandleFunc("/", s.handleIndex)

	// Start gRPC server
	if nwService, ok := s.contentService.(*NetworkVideoContentService); ok {
		proto.RegisterVideoContentAdminServiceServer(s.grpcServer, nwService)
		go func() {
			adminAddr := nwService.adminAddr
			adminLis, err := net.Listen("tcp", adminAddr)
			if err != nil {
				fmt.Printf("Failed to listen for admin service: %v\n", err)
				return
			}
			fmt.Printf("Starting admin service on %s\n", adminAddr)
			if err := s.grpcServer.Serve(adminLis); err != nil {
				fmt.Printf("Failed to serve admin service: %v\n", err)
			}
		}()
	}

	return http.Serve(lis, s.mux)
}

func (s *server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	videos, err := s.metadataService.List()
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	tmpl, err := template.New("index").Parse(indexHTML)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Add escaped ID for template
	type VideoWithEscapedID struct {
		VideoMetadata
		EscapedId  string
		UploadTime string
	}
	var videosWithEscapedID []VideoWithEscapedID
	for _, v := range videos {
		videosWithEscapedID = append(videosWithEscapedID, VideoWithEscapedID{
			VideoMetadata: v,
			EscapedId:     template.HTMLEscapeString(v.Id),
			UploadTime:    v.UploadedAt.Format("2006-01-02 15:04:05"),
		})
	}

	if err := tmpl.Execute(w, videosWithEscapedID); err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

func (s *server) convertToDASH(videoId string, inputPath string) error {
	// Create output directory for DASH files
	var outputDir string
	if fsService, ok := s.contentService.(*FSVideoContentService); ok {
		outputDir = filepath.Join(fsService.baseDir, videoId)
	} else {
		outputDir = filepath.Join("tmp", videoId)
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	manifestPath := filepath.Join(outputDir, "manifest.mpd")

	// FFmpeg command to convert to DASH format with recommended parameters
	cmd := exec.Command("ffmpeg",
		"-i", inputPath, // input file
		"-c:v", "libx264", // video codec
		"-c:a", "aac", // audio codec
		"-bf", "1", // max 1 b-frame
		"-keyint_min", "120", // minimum keyframe interval
		"-g", "120", // keyframe every 120 frames
		"-sc_threshold", "0", // scene change threshold
		"-b:v", "3000k", // video bitrate
		"-b:a", "128k", // audio bitrate
		"-f", "dash", // dash format
		"-use_timeline", "1", // use timeline
		"-use_template", "1", // use template
		"-init_seg_name", "init-$RepresentationID$.m4s", // init segment naming
		"-media_seg_name", "chunk-$RepresentationID$-$Number%05d$.m4s", // media segment naming
		"-seg_duration", "4", // segment duration in seconds
		manifestPath) // output file

	// Capture both stdout and stderr
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run the command
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to convert video: %v", err)
	}

	// Verify that the manifest file was created
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		return fmt.Errorf("manifest file was not created at %s", manifestPath)
	}

	// If using NetworkVideoContentService, write the files to the network storage
	if nwService, ok := s.contentService.(*NetworkVideoContentService); ok {
		// Read all files in the output directory
		files, err := ioutil.ReadDir(outputDir)
		if err != nil {
			return fmt.Errorf("failed to read output directory: %v", err)
		}

		// Write each file to the network storage
		for _, file := range files {
			if file.IsDir() {
				continue
			}
			data, err := ioutil.ReadFile(filepath.Join(outputDir, file.Name()))
			if err != nil {
				return fmt.Errorf("failed to read file %s: %v", file.Name(), err)
			}
			if err := nwService.Write(videoId, file.Name(), data); err != nil {
				return fmt.Errorf("failed to write file %s to network storage: %v", file.Name(), err)
			}
		}

		// Clean up local files after successful upload
		if err := os.RemoveAll(outputDir); err != nil {
			return fmt.Errorf("failed to clean up local files: %v", err)
		}
	}

	return nil
}

func (s *server) handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse multipart form
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	// Get file from form
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Read file data
	data, err := ioutil.ReadAll(file)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Generate video ID from filename
	videoId := strings.TrimSuffix(header.Filename, filepath.Ext(header.Filename))
	if videoId == "" {
		http.Error(w, "Invalid filename", http.StatusBadRequest)
		return
	}

	// Check if video already exists
	metadata, err := s.metadataService.Read(videoId)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if metadata != nil {
		http.Error(w, "Video already exists", http.StatusConflict)
		return
	}

	// Create video metadata
	if err := s.metadataService.Create(videoId, time.Now()); err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Create temp directory based on content service type
	var tempDir string
	if fsService, ok := s.contentService.(*FSVideoContentService); ok {
		tempDir = filepath.Join(fsService.baseDir, "temp")
	} else {
		tempDir = filepath.Join("tmp", "temp")
	}

	if err := os.MkdirAll(tempDir, 0755); err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	tempFile := filepath.Join(tempDir, header.Filename)
	if err := ioutil.WriteFile(tempFile, data, 0644); err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer os.Remove(tempFile)

	// Convert to DASH format
	if err := s.convertToDASH(videoId, tempFile); err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Redirect to home page
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *server) handleVideo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	videoId := r.URL.Path[len("/videos/"):]
	if videoId == "" {
		http.Error(w, "Invalid video ID", http.StatusBadRequest)
		return
	}

	// Get video metadata
	metadata, err := s.metadataService.Read(videoId)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if metadata == nil {
		http.NotFound(w, r)
		return
	}

	// Add formatted time for template
	type VideoWithFormattedTime struct {
		VideoMetadata
		UploadedAt string
	}
	data := VideoWithFormattedTime{
		VideoMetadata: *metadata,
		UploadedAt:    metadata.UploadedAt.Format("2006-01-02 15:04:05"),
	}

	// Render video page
	tmpl, err := template.New("video").Parse(videoHTML)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

func (s *server) handleVideoContent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// parse /content/<videoId>/<filename>
	path := r.URL.Path[len("/content/"):]
	parts := strings.Split(path, "/")
	if len(parts) != 2 {
		http.Error(w, "Invalid content path", http.StatusBadRequest)
		return
	}
	videoId := parts[0]
	filename := parts[1]

	// Check if video exists
	if _, err := s.metadataService.Read(videoId); err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Get content using the interface method
	data, err := s.contentService.Read(videoId, filename)
	if err != nil {
		if os.IsNotExist(err) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Set content type based on file extension
	switch filepath.Ext(filename) {
	case ".mpd":
		w.Header().Set("Content-Type", "application/dash+xml")
	case ".m4s":
		w.Header().Set("Content-Type", "video/mp4")
	default:
		w.Header().Set("Content-Type", "application/octet-stream")
	}

	w.Write(data)
}
