.PHONY: proto clean build run dev

# 编译 proto 文件
proto:
	@echo "Compiling proto files..."
	@mkdir -p grpc/gen/registry grpc/gen/recommendation
	@protoc --go_out=grpc/gen/registry --go_opt=paths=source_relative \
		--go-grpc_out=grpc/gen/registry --go-grpc_opt=paths=source_relative \
		--proto_path=grpc/proto grpc/proto/registry.proto
	@protoc --go_out=grpc/gen/recommendation --go_opt=paths=source_relative \
		--go-grpc_out=grpc/gen/recommendation --go-grpc_opt=paths=source_relative \
		--proto_path=doc/proto doc/proto/recommendation.proto

# 清理生成的文件
clean:
	@echo "Cleaning generated files..."
	@find . -name "*.pb.go" -delete

# 构建项目（包含 proto 编译）
build: proto
	@echo "Building project..."
	@go build -o bin/amway .

# 运行项目（包含 proto 编译）
run: proto
	@echo "Running project..."
	@go run .

# 开发模式（监听文件变化）
dev: proto
	@echo "Starting development mode..."
	@go run .