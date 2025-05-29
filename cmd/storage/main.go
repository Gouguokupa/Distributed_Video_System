package main

import (
	"flag"
	"fmt"
	"log"
	"net"

	"google.golang.org/grpc"

	pb "tritontube/internal/proto"
	"tritontube/internal/storage"
)

func main() {
	host := flag.String("host", "localhost", "Host to listen on")
	port := flag.Int("port", 8090, "Port to listen on")
	flag.Parse()

	if flag.NArg() != 1 {
		log.Fatal("Usage: storage -host HOST -port PORT STORAGE_DIR")
	}

	storageDir := flag.Arg(0)
	server, err := storage.NewStorageServer(storageDir)
	if err != nil {
		log.Fatalf("Failed to create storage server: %v", err)
	}

	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", *host, *port))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterStorageServiceServer(s, server)

	log.Printf("Storage server listening on %s:%d", *host, *port)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
