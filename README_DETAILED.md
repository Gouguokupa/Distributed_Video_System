# TritonTube: 分布式视频存储与流媒体系统

## 项目概述

TritonTube 是一个基于 Go 语言开发的分布式视频存储与流媒体系统，实现了高可用性、可扩展的视频内容分发服务。该系统采用微服务架构，支持视频上传、DASH 格式转换、分布式存储和动态节点管理。

## 系统架构

### 核心组件

1. **Web 服务器** (`cmd/web/`)
   - 提供 HTTP API 和 Web 界面
   - 处理视频上传和播放请求
   - 支持 DASH 格式转换
   - 集成元数据服务和内容服务

2. **存储节点** (`cmd/storage/`)
   - 基于 gRPC 的分布式存储服务
   - 支持文件的读写删除操作
   - 提供视频 ID 和文件列表查询

3. **管理服务** (`cmd/admin/`)
   - 节点动态添加和移除
   - 集群状态管理
   - 数据迁移控制

### 技术栈

- **语言**: Go 1.24.1
- **通信**: gRPC, HTTP/HTTPS
- **数据库**: SQLite, etcd
- **视频处理**: FFmpeg
- **存储**: 分布式文件系统
- **哈希算法**: SHA-256 一致性哈希

## 功能特性

### 视频处理
- 支持多种视频格式上传
- 自动转换为 DASH 格式
- 分片存储优化流媒体播放
- 支持自适应码率

### 分布式存储
- 一致性哈希算法分布数据
- 动态节点添加和移除
- 自动数据迁移
- 故障容错机制

### 管理功能
- Web 界面视频管理
- 集群状态监控
- 节点健康检查
- 数据一致性保证

## 项目结构

```
tritontube/
├── cmd/                    # 命令行工具
│   ├── web/               # Web 服务器
│   ├── storage/           # 存储节点服务
│   └── admin/             # 管理工具
├── internal/              # 内部包
│   ├── proto/             # Protocol Buffers 定义
│   ├── storage/           # 存储服务实现
│   ├── web/               # Web 服务实现
│   └── video/             # 视频处理逻辑
├── proto/                 # .proto 文件
├── storage/               # 存储目录
└── tmp/                   # 临时文件
```

## 快速开始

### 环境要求

- Go 1.24.1 或更高版本
- FFmpeg (用于视频转换)
- 至少 3 个存储节点

### 安装依赖

```bash
# 生成 Protocol Buffers 代码
make proto

# 安装 Go 依赖
go mod download
```

### 启动系统

#### 1. 启动存储节点

```bash
# 创建存储目录
mkdir -p storage/8090 storage/8091 storage/8092

# 启动三个存储节点
go run cmd/storage/main.go -port 8090 ./storage/8090 &
go run cmd/storage/main.go -port 8091 ./storage/8091 &
go run cmd/storage/main.go -port 8092 ./storage/8092 &
```

#### 2. 启动 Web 服务器

```bash
# 使用 SQLite 和网络存储
go run cmd/web/main.go -port 8080 sqlite ./metadata.db nw localhost:8081,localhost:8090,localhost:8091,localhost:8092 &
```

#### 3. 访问系统

- Web 界面: http://localhost:8080
- 上传视频并查看播放效果

### 管理操作

#### 添加节点

```bash
go run cmd/admin/main.go add localhost:8081 localhost:8093
```

#### 移除节点

```bash
go run cmd/admin/main.go remove localhost:8081 localhost:8093
```

#### 列出所有节点

```bash
go run cmd/admin/main.go list localhost:8081
```

## API 接口

### Web 接口

- `GET /` - 视频列表页面
- `POST /upload` - 视频上传
- `GET /videos/{videoId}` - 视频播放页面
- `GET /content/{videoId}/{filename}` - 视频内容访问

### gRPC 接口

#### StorageService

- `Read(ReadRequest)` - 读取文件
- `Write(WriteRequest)` - 写入文件
- `Delete(DeleteRequest)` - 删除文件
- `ListVideoIDs()` - 列出视频 ID
- `ListFiles(ListFilesRequest)` - 列出文件

#### VideoContentAdminService

- `AddNode(AddNodeRequest)` - 添加节点
- `RemoveNode(RemoveNodeRequest)` - 移除节点
- `ListNodes()` - 列出节点

## 一致性哈希算法

系统使用 SHA-256 一致性哈希算法来分布数据：

1. 每个存储节点根据其地址计算哈希值
2. 文件根据 `videoId/filename` 计算哈希值
3. 文件存储在哈希环上顺时针方向的第一个节点
4. 节点添加/移除时自动迁移相关数据

## 数据迁移策略

当节点发生变化时：

1. **添加节点**: 从其他节点迁移部分数据到新节点
2. **移除节点**: 将节点数据迁移到其他可用节点
3. **迁移过程**: 保证数据不丢失，系统持续可用

## 测试

### 端到端测试

运行完整的系统测试：

```bash
chmod +x end2end_test.sh
./end2end_test.sh
```

测试流程包括：
1. 启动 8 个存储节点
2. 启动 Web 服务器
3. 上传测试视频
4. 动态添加/移除节点
5. 验证数据一致性和播放功能

### 单元测试

```bash
go test ./...
```

## 配置选项

### Web 服务器配置

```bash
go run cmd/web/main.go [OPTIONS] METADATA_TYPE METADATA_OPTIONS CONTENT_TYPE CONTENT_OPTIONS

选项:
  -host string    服务器地址 (默认 "0.0.0.0")
  -port int       端口号 (默认 8080)

参数:
  METADATA_TYPE    元数据服务类型 (sqlite, etcd)
  METADATA_OPTIONS 元数据服务选项
  CONTENT_TYPE     内容服务类型 (fs, nw)
  CONTENT_OPTIONS  内容服务选项
```

### 存储节点配置

```bash
go run cmd/storage/main.go [OPTIONS] STORAGE_DIR

选项:
  -host string    监听地址 (默认 "localhost")
  -port int       监听端口 (默认 8090)
```

## 性能优化

### 视频处理优化

- 使用 FFmpeg 进行高效的视频转换
- DASH 格式支持自适应码率
- 分片存储提高并发访问性能

### 存储优化

- 一致性哈希减少数据迁移开销
- gRPC 提供高性能的 RPC 通信
- 支持水平扩展

## 故障处理

### 节点故障

- 自动检测节点不可用
- 数据自动迁移到健康节点
- 服务持续可用

### 数据一致性

- 使用事务保证数据完整性
- 定期健康检查
- 数据备份和恢复机制

## 开发指南

### 添加新功能

1. 在 `proto/` 目录定义新的 gRPC 接口
2. 运行 `make proto` 生成代码
3. 实现服务逻辑
4. 更新测试用例

### 代码结构

- `internal/proto/` - 自动生成的 Protocol Buffers 代码
- `internal/storage/` - 存储服务实现
- `internal/web/` - Web 服务实现
- `cmd/` - 命令行工具入口

## 许可证

本项目为 CS 224 课程作业，仅用于教育目的。

## 贡献

欢迎提交 Issue 和 Pull Request 来改进项目。

## 联系方式

如有问题，请联系课程教师或查看课程文档。
