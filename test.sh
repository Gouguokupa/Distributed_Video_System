mkdir -p storage/8090 storage/8091 storage/8092
go run ./cmd/storage -port 8090 "./storage/8090"
go run ./cmd/storage -port 8091 "./storage/8091"
go run ./cmd/storage -port 8092 "./storage/8092"