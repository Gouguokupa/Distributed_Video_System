#!/bin/bash

set -e
rm -rf tmp  # comment out this line if you want to keep the data from ./tmp during testing

# Ctrl+C or exit
cleanup() {
    echo "🧹 Cleaning up background processes..."
    ps aux | grep go-build | awk '{print $2}' | xargs kill 2>/dev/null || true
    ps aux | grep cmd/storage | awk '{print $2}' | xargs kill 2>/dev/null || true
    echo "😉 Cleanup complete."
}
trap cleanup EXIT

# 🚫 pre-cleanup
ps aux | grep go-build | awk '{print $2}' | xargs kill 2>/dev/null || true
ps aux | grep cmd/storage | awk '{print $2}' | xargs kill 2>/dev/null || true

mkdir -p tmp
> tmp/test.log

echo "🚀 Step 1: Launching 8 storage nodes on ports 8090–8097..."

for i in {0..7}; do
    port=$((8090 + i))
    mkdir -p tmp/$port
    echo "> Starting storage node $((i+1)) on port $port..."
    go run cmd/storage/main.go -port $port tmp/$port >> tmp/test.log 2>&1 &
done

sleep 2  # Give more time for storage nodes to start

echo ""
echo "🌐 Step 2: Starting web server (admin on 8081, using nodes 8090–8092)..."
go run cmd/web/main.go sqlite ./tmp/metadata.db nw localhost:8081,localhost:8090,localhost:8091,localhost:8092 >> tmp/test.log 2>&1 &
sleep 2  # Give more time for web server to start

echo
echo "📂 Step 3: Please RENAME your test video as 'PIKACHU.mp4' and upload it to:"
echo "http://localhost:8080"
echo "⌛ WAIT until the video link show up, then press ENTER to continue..."
read

echo "🔍 Step 4: Checking whether manifest is stored on node2 (port 8091)..."
if [ -f tmp/8091/PIKACHU/manifest.mpd ]; then
    echo "✅ PASS: PIKACHU/manifest.mpd found in node2 (tmp/test2)"
else
    echo "❌ FAIL: PIKACHU/manifest.mpd NOT found in node2"
fi

echo
echo "📋 Step 5: Listing nodes (expecting 3)..."
go run cmd/admin/main.go list localhost:8081

echo
echo "➕ Step 6: Adding nodes 8093–8097..."
for port in {8093..8097}; do
    go run cmd/admin/main.go add localhost:8081 localhost:$port
done

echo
echo "📋 Step 7: Listing nodes (expecting 8)..."
go run cmd/admin/main.go list localhost:8081

echo
echo "🧪 Step 8: Please visit your video in browser:"
echo "http://localhost:8080/videos/PIKACHU"
echo "✅ If playback works correctly, press ENTER to continue..."
read

echo "➖ Step 9: Removing nodes 8093–8097..."
for port in {8093..8097}; do
    go run cmd/admin/main.go remove localhost:8081 localhost:$port
done

echo
echo "📋 Step 10: Checking if manifest still in node2..."
if [ -f tmp/8091/PIKACHU/manifest.mpd ]; then
    echo "✅ PASS: PIKACHU/manifest.mpd still in node2"
else
    echo "❌ FAIL: PIKACHU/manifest.mpd is not in node2"
fi

echo
echo "➖ Step 11: Removing nodes 8090 and 8091..."
go run cmd/admin/main.go remove localhost:8081 localhost:8090
go run cmd/admin/main.go remove localhost:8081 localhost:8091

echo
echo "📋 Step 12: Final node list (should be only node3)..."
go run cmd/admin/main.go list localhost:8081

echo
echo "🌐 Step 13: Reset and reboot the whole cluster, then check consistency"
go run cmd/admin/main.go add localhost:8081 localhost:8090
go run cmd/admin/main.go add localhost:8081 localhost:8091

# clean up
ps aux | grep go-build | awk '{print $2}' | xargs kill 2>/dev/null || true
ps aux | grep cmd/storage | awk '{print $2}' | xargs kill 2>/dev/null || true

for i in {0..2}; do
    port=$((8090 + i))
    echo "> Rebooting storage node $((i+1)) on port $port..."
    go run cmd/storage/main.go -port $port tmp/$port >> tmp/test.log 2>&1 &
done

sleep 2  # Give more time for storage nodes to start

echo "> Rebooting web server on port 8080..."
go run cmd/web/main.go sqlite ./tmp/metadata.db nw localhost:8081,localhost:8090,localhost:8091,localhost:8092 >> tmp/test.log 2>&1 &
sleep 2  # Give more time for web server to start

echo
echo "🧪 Please visit your video in browser again:"
echo "http://localhost:8080/videos/PIKACHU"
echo "✅ If playback works correctly, press ENTER to continue..."
read
