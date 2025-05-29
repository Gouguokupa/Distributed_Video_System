// Lab 8: Implement a network video content service (client using consistent hashing)

package web

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"sort"
	"strings"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"tritontube/internal/proto"
)

// NetworkVideoContentService implements VideoContentService using distributed storage
type NetworkVideoContentService struct {
	proto.UnimplementedVideoContentAdminServiceServer
	mu sync.RWMutex

	// Admin server address
	adminAddr string

	// Map of node addresses to their gRPC clients
	clients map[string]proto.StorageServiceClient

	// Sorted list of node hashes for consistent hashing
	nodeHashes []uint64
	nodeMap    map[uint64]string

	migratedFiles map[string]bool // 记录已迁移的文件
}

// NewNetworkVideoContentService creates a new NetworkVideoContentService
func NewNetworkVideoContentService(options string) (*NetworkVideoContentService, error) {
	parts := strings.Split(options, ",")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid options format: %s", options)
	}

	adminAddr := parts[0]
	nodes := parts[1:]

	service := &NetworkVideoContentService{
		adminAddr:  adminAddr,
		clients:    make(map[string]proto.StorageServiceClient),
		nodeHashes: make([]uint64, 0, len(nodes)),
		nodeMap:    make(map[uint64]string),
	}

	// Connect to all nodes
	for _, node := range nodes {
		if err := service.addNode(node); err != nil {
			return nil, fmt.Errorf("failed to connect to node %s: %v", node, err)
		}
	}

	return service, nil
}

// hashStringToUint64 computes the hash of a string using SHA-256
func hashStringToUint64(s string) uint64 {
	sum := sha256.Sum256([]byte(s))
	return binary.BigEndian.Uint64(sum[:8])
}

// addNode adds a new node to the consistent hash ring
func (s *NetworkVideoContentService) addNode(nodeAddr string) error {
	// Connect to the node
	conn, err := grpc.Dial(nodeAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("failed to connect to node: %v", err)
	}

	client := proto.NewStorageServiceClient(conn)
	s.clients[nodeAddr] = client

	// Add to hash ring
	hash := hashStringToUint64(nodeAddr)
	s.nodeHashes = append(s.nodeHashes, hash)
	s.nodeMap[hash] = nodeAddr
	sort.Slice(s.nodeHashes, func(i, j int) bool {
		return s.nodeHashes[i] < s.nodeHashes[j]
	})

	return nil
}

// removeNode removes a node from the consistent hash ring
func (s *NetworkVideoContentService) removeNode(nodeAddr string) error {
	hash := hashStringToUint64(nodeAddr)

	// Remove from hash ring
	for i, h := range s.nodeHashes {
		if h == hash {
			s.nodeHashes = append(s.nodeHashes[:i], s.nodeHashes[i+1:]...)
			delete(s.nodeMap, hash)
			delete(s.clients, nodeAddr)
			return nil
		}
	}

	return fmt.Errorf("node not found: %s", nodeAddr)
}

// getNodeForKey returns the node that should store the given key
func (s *NetworkVideoContentService) getNodeForKey(key string) string {
	if len(s.nodeHashes) == 0 {
		return ""
	}

	hash := hashStringToUint64(key)

	// Find the first node with hash greater than the key's hash
	for _, nodeHash := range s.nodeHashes {
		if nodeHash >= hash {
			return s.nodeMap[nodeHash]
		}
	}

	// If no node has a greater hash, wrap around to the first node
	return s.nodeMap[s.nodeHashes[0]]
}

// Read implements VideoContentService.Read
func (s *NetworkVideoContentService) Read(videoID string, filename string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := fmt.Sprintf("%s/%s", videoID, filename)
	nodeAddr := s.getNodeForKey(key)
	if nodeAddr == "" {
		return nil, fmt.Errorf("no storage nodes available")
	}

	client := s.clients[nodeAddr]
	resp, err := client.Read(context.Background(), &proto.ReadRequest{
		VideoId:  videoID,
		Filename: filename,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to read from node %s: %v", nodeAddr, err)
	}

	return resp.Content, nil
}

// Write implements VideoContentService.Write
func (s *NetworkVideoContentService) Write(videoID string, filename string, content []byte) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := fmt.Sprintf("%s/%s", videoID, filename)
	nodeAddr := s.getNodeForKey(key)
	if nodeAddr == "" {
		return fmt.Errorf("no storage nodes available")
	}

	client := s.clients[nodeAddr]
	_, err := client.Write(context.Background(), &proto.WriteRequest{
		VideoId:  videoID,
		Filename: filename,
		Content:  content,
	})
	if err != nil {
		return fmt.Errorf("failed to write to node %s: %v", nodeAddr, err)
	}

	return nil
}

// Delete implements VideoContentService.Delete
func (s *NetworkVideoContentService) Delete(videoID string, filename string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := fmt.Sprintf("%s/%s", videoID, filename)
	nodeAddr := s.getNodeForKey(key)
	if nodeAddr == "" {
		return fmt.Errorf("no storage nodes available")
	}

	client := s.clients[nodeAddr]
	_, err := client.Delete(context.Background(), &proto.DeleteRequest{
		VideoId:  videoID,
		Filename: filename,
	})
	if err != nil {
		return fmt.Errorf("failed to delete from node %s: %v", nodeAddr, err)
	}

	return nil
}

// listNodesInternal returns the list of nodes in the cluster
func (s *NetworkVideoContentService) listNodesInternal() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	nodes := make([]string, 0, len(s.nodeHashes))
	for _, hash := range s.nodeHashes {
		nodes = append(nodes, s.nodeMap[hash])
	}
	return nodes
}

// AddNode implements VideoContentAdminServiceServer.AddNode
func (s *NetworkVideoContentService) AddNode(ctx context.Context, req *proto.AddNodeRequest) (*proto.AddNodeResponse, error) {
	count, err := s.addNodeInternal(req.NodeAddress)
	if err != nil {
		return nil, err
	}
	return &proto.AddNodeResponse{MigratedFileCount: int32(count)}, nil
}

// RemoveNode implements VideoContentAdminServiceServer.RemoveNode
func (s *NetworkVideoContentService) RemoveNode(ctx context.Context, req *proto.RemoveNodeRequest) (*proto.RemoveNodeResponse, error) {
	count, err := s.removeNodeInternal(req.NodeAddress)
	if err != nil {
		return nil, err
	}
	return &proto.RemoveNodeResponse{MigratedFileCount: int32(count)}, nil
}

// ListNodes implements VideoContentAdminServiceServer.ListNodes
func (s *NetworkVideoContentService) ListNodes(ctx context.Context, req *proto.ListNodesRequest) (*proto.ListNodesResponse, error) {
	nodes := s.listNodesInternal()
	return &proto.ListNodesResponse{Nodes: nodes}, nil
}

// Rename the internal methods
func (s *NetworkVideoContentService) addNodeInternal(nodeAddr string) (int, error) {
	// 1. 先把新节点加到哈希环
	if err := s.addNode(nodeAddr); err != nil {
		return 0, err
	}

	migratedCount := 0
	migratedFiles := make(map[string]bool)

	// 2. 遍历所有节点，收集所有文件
	for srcAddr, client := range s.clients {
		if srcAddr == nodeAddr {
			// 新节点上本来就没有文件，跳过
			continue
		}
		videoIDs, err := s.listVideoIDs(client)
		if err != nil {
			return migratedCount, err
		}
		for _, videoID := range videoIDs {
			files, err := s.listFiles(client, videoID)
			if err != nil {
				return migratedCount, err
			}
			for _, filename := range files {
				key := fmt.Sprintf("%s/%s", videoID, filename)
				if migratedFiles[key] {
					continue
				}
				// 3. 计算该文件现在应该属于哪个节点
				targetAddr := s.getNodeForKey(key)
				if targetAddr == nodeAddr {
					// 4. 迁移：只迁移那些目标节点是新节点的文件
					// 读取
					resp, err := client.Read(context.Background(), &proto.ReadRequest{
						VideoId:  videoID,
						Filename: filename,
					})
					if err != nil {
						return migratedCount, err
					}
					// 写入
					newClient := s.clients[nodeAddr]
					_, err = newClient.Write(context.Background(), &proto.WriteRequest{
						VideoId:  videoID,
						Filename: filename,
						Content:  resp.Content,
					})
					if err != nil {
						return migratedCount, err
					}
					// 删除
					_, err = client.Delete(context.Background(), &proto.DeleteRequest{
						VideoId:  videoID,
						Filename: filename,
					})
					if err != nil {
						return migratedCount, err
					}
					migratedFiles[key] = true
					migratedCount++
					fmt.Printf("[MIGRATE-ADD] %s from %s to %s\n", key, srcAddr, nodeAddr)
				}
			}
		}
	}
	return migratedCount, nil
}

func (s *NetworkVideoContentService) removeNodeInternal(nodeAddr string) (int, error) {
	client := s.clients[nodeAddr]
	if client == nil {
		return 0, fmt.Errorf("node not found: %s", nodeAddr)
	}

	migratedCount := 0
	migratedFiles := make(map[string]bool)

	// 1. 获取要移除节点上的所有文件
	videoIDs, err := s.listVideoIDs(client)
	if err != nil {
		return 0, err
	}
	for _, videoID := range videoIDs {
		files, err := s.listFiles(client, videoID)
		if err != nil {
			return migratedCount, err
		}
		for _, filename := range files {
			key := fmt.Sprintf("%s/%s", videoID, filename)
			if migratedFiles[key] {
				continue
			}
			// 2. 计算该文件移除节点后应该属于哪个节点
			// 先临时把节点移除哈希环
			s.removeNode(nodeAddr)
			targetAddr := s.getNodeForKey(key)
			// 再加回来，保证后续循环不影响
			s.addNode(nodeAddr)
			if targetAddr == nodeAddr {
				// 该文件依然属于本节点，不迁移
				continue
			}
			// 3. 迁移到新节点
			resp, err := client.Read(context.Background(), &proto.ReadRequest{
				VideoId:  videoID,
				Filename: filename,
			})
			if err != nil {
				return migratedCount, err
			}
			targetClient := s.clients[targetAddr]
			_, err = targetClient.Write(context.Background(), &proto.WriteRequest{
				VideoId:  videoID,
				Filename: filename,
				Content:  resp.Content,
			})
			if err != nil {
				return migratedCount, err
			}
			_, err = client.Delete(context.Background(), &proto.DeleteRequest{
				VideoId:  videoID,
				Filename: filename,
			})
			if err != nil {
				return migratedCount, err
			}
			migratedFiles[key] = true
			migratedCount++
			fmt.Printf("[MIGRATE-REMOVE] %s from %s to %s\n", key, nodeAddr, targetAddr)
		}
	}
	// 4. 最后真正移除节点
	s.removeNode(nodeAddr)
	return migratedCount, nil
}

// listVideoIDs returns a list of all video IDs stored on a node
func (s *NetworkVideoContentService) listVideoIDs(client proto.StorageServiceClient) ([]string, error) {
	resp, err := client.ListVideoIDs(context.Background(), &proto.ListVideoIDsRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to list video IDs: %v", err)
	}
	return resp.VideoIds, nil
}

// listFiles returns a list of all files for a video stored on a node
func (s *NetworkVideoContentService) listFiles(client proto.StorageServiceClient, videoID string) ([]string, error) {
	resp, err := client.ListFiles(context.Background(), &proto.ListFilesRequest{
		VideoId: videoID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %v", err)
	}
	return resp.Filenames, nil
}

// Uncomment the following line to ensure NetworkVideoContentService implements VideoContentService
// var _ VideoContentService = (*NetworkVideoContentService)(nil)
