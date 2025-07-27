.PHONY: proto gen build run-grpc run-http clean install-deps

# Install required tools
install-deps:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest
	go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest

# Generate protobuf code
proto:
	mkdir -p gen/api/v1
	protoc --proto_path=proto \
		--proto_path=third_party \
		--go_out=gen --go_opt=paths=source_relative \
		--go-grpc_out=gen --go-grpc_opt=paths=source_relative \
		--grpc-gateway_out=gen --grpc-gateway_opt=paths=source_relative \
		--openapiv2_out=gen --openapiv2_opt=logtostderr=true \
		proto/models/v1/models.proto

# Download third party protos (googleapis)
third-party:
	mkdir -p third_party/google/api
	curl -o third_party/google/api/annotations.proto \
		https://raw.githubusercontent.com/googleapis/googleapis/master/google/api/annotations.proto
	curl -o third_party/google/api/http.proto \
		https://raw.githubusercontent.com/googleapis/googleapis/master/google/api/http.proto


# Clean generated files
clean:
	rm -rf gen/
	rm -rf bin/
	rm -rf third_party/


gen: third-party proto

dev: clean gen build