#!/bin/bash

# Docker测试脚本
echo "=== TritonTube Docker 测试脚本 ==="

# 检查Docker是否安装
if ! command -v docker &> /dev/null; then
    echo "❌ Docker 未安装，请先安装Docker"
    exit 1
fi

echo "✅ Docker 已安装"

# 构建镜像
echo "🔨 构建TritonTube Docker镜像..."
docker build -t tritontube:latest .

if [ $? -eq 0 ]; then
    echo "✅ Docker镜像构建成功"
else
    echo "❌ Docker镜像构建失败"
    exit 1
fi

# 测试镜像
echo "🧪 测试Docker镜像..."
docker run --rm tritontube:latest /bin/bash -c "echo 'Docker镜像运行正常'"

if [ $? -eq 0 ]; then
    echo "✅ Docker镜像测试通过"
else
    echo "❌ Docker镜像测试失败"
    exit 1
fi

# 显示镜像信息
echo "📋 Docker镜像信息:"
docker images tritontube:latest

echo ""
echo "🎉 所有测试通过！"
echo ""
echo "🚀 使用以下命令启动服务:"
echo "  docker run -d -p 8080:8080 --name tritontube-web tritontube:latest"
echo ""
echo "🔧 进入容器调试:"
echo "  docker exec -it tritontube-web /bin/bash"
