# TritonTube: Distributed Video Storage and Streaming System

## Project Overview

TritonTube is a distributed video storage and streaming system developed in Go, implementing high-availability and scalable video content delivery services. The system adopts a microservices architecture, supporting video upload, DASH format conversion, distributed storage, and dynamic node management.

## System Architecture

### Core Components

1. **Web Server** (`cmd/web/`)
   - Provides HTTP API and web interface
   - Handles video upload and playback requests
   - Supports DASH format conversion
   - Integrates metadata and content services

2. **Storage Nodes** (`cmd/storage/`)
   - gRPC-based distributed storage services
   - Supports file read/write/delete operations
   - Provides video ID and file list queries

3. **Admin Service** (`cmd/admin/`)
   - Dynamic node addition and removal
   - Cluster state management
   - Data migration control

### Technology Stack

- **Language**: Go 1.24.1
- **Communication**: gRPC, HTTP/HTTPS
- **Database**: SQLite, etcd
- **Video Processing**: FFmpeg
- **Storage**: Distributed file system
- **Hash Algorithm**: SHA-256 consistent hashing

## Features

### Video Processing
- Supports multiple video format uploads
- Automatic DASH format conversion
- Segmented storage for optimized streaming
- Adaptive bitrate support

### Distributed Storage
- Consistent hashing algorithm for data distribution
- Dynamic node addition and removal
- Automatic data migration
- Fault tolerance mechanisms

### Management Features
- Web interface for video management
- Cluster status monitoring
- Node health checks
- Data consistency guarantees

## Quick Start

### Prerequisites

- Go 1.24.1 or higher
- FFmpeg (for video conversion)
- At least 3 storage nodes

### Install Dependencies

```bash
# Generate Protocol Buffers code
make proto

# Install Go dependencies
go mod download
```

### Start the System

#### 1. Start Storage Nodes

```bash
# Create storage directories
mkdir -p storage/8090 storage/8091 storage/8092

# Start three storage nodes
go run cmd/storage/main.go -port 8090 ./storage/8090 &
go run cmd/storage/main.go -port 8091 ./storage/8091 &
go run cmd/storage/main.go -port 8092 ./storage/8092 &
```

#### 2. Start Web Server

```bash
# Use SQLite and network storage
go run cmd/web/main.go -port 8080 sqlite ./metadata.db nw localhost:8081,localhost:8090,localhost:8091,localhost:8092 &
```

#### 3. Access the System

- Web Interface: http://localhost:8080
- Upload videos and test playback functionality

### Management Operations

#### Add Node

```bash
go run cmd/admin/main.go add localhost:8081 localhost:8093
```

#### Remove Node

```bash
go run cmd/admin/main.go remove localhost:8081 localhost:8093
```

#### List All Nodes

```bash
go run cmd/admin/main.go list localhost:8081
```

## Testing

### End-to-End Testing

Run the complete system test:

```bash
chmod +x end2end_test.sh
./end2end_test.sh
```

### Quick Test

```bash
chmod +x test.sh
./test.sh
```

## Project Structure

```
tritontube/
├── cmd/                    # Command line tools
│   ├── web/               # Web server
│   ├── storage/           # Storage node service
│   └── admin/             # Management tools
├── internal/              # Internal packages
│   ├── proto/             # Protocol Buffers definitions
│   ├── storage/           # Storage service implementation
│   ├── web/               # Web service implementation
│   └── video/             # Video processing logic
├── proto/                 # .proto files
├── storage/               # Storage directory
└── tmp/                   # Temporary files
```

## API Interfaces

### Web Interfaces

- `GET /` - Video list page
- `POST /upload` - Video upload
- `GET /videos/{videoId}` - Video playback page
- `GET /content/{videoId}/{filename}` - Video content access

### gRPC Interfaces

#### StorageService

- `Read(ReadRequest)` - Read file
- `Write(WriteRequest)` - Write file
- `Delete(DeleteRequest)` - Delete file
- `ListVideoIDs()` - List video IDs
- `ListFiles(ListFilesRequest)` - List files

#### VideoContentAdminService

- `AddNode(AddNodeRequest)` - Add node
- `RemoveNode(RemoveNodeRequest)` - Remove node
- `ListNodes()` - List nodes

## Consistent Hashing

The system uses SHA-256 consistent hashing to distribute data across storage nodes:

1. Each storage node is assigned a hash value based on its address
2. Files are assigned hash values based on `videoId/filename`
3. Files are stored on the first node clockwise from their hash position
4. When nodes are added/removed, only relevant data needs to be migrated

## Data Migration

When nodes change:

1. **Adding Node**: Migrate relevant data from other nodes to the new node
2. **Removing Node**: Migrate node data to other available nodes
3. **Migration Process**: Ensures no data loss and continuous system availability

## Performance Features

### Video Processing Optimization
- Efficient video conversion using FFmpeg
- DASH format supports adaptive bitrate
- Segmented storage improves concurrent access performance

### Storage Optimization
- Consistent hashing reduces data migration overhead
- gRPC provides high-performance RPC communication
- Supports horizontal scaling

## Fault Handling

### Node Failures
- Automatic detection of node unavailability
- Data automatically migrates to healthy nodes
- Continuous service availability

### Data Consistency
- Uses transactions to ensure data integrity
- Regular health checks
- Data backup and recovery mechanisms

## License

This project is for CS 224 course assignment purposes only.

## Detailed Documentation

See [README_DETAILED.md](./README_DETAILED.md) for complete project documentation, including detailed architecture explanations, configuration options, and development guidelines.
