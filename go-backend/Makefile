.PHONY: install proto build run run-dev test clean

BINARY_NAME := server
PROTO_DIR := proto
GENERATED_DIR := pkg/pb
CMD_DIR := cmd/server

install:
	@echo "Ustanovka..."
	go mod download
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	@echo "OK"

proto: install
	@echo "Generacija proto..."
	@mkdir -p $(GENERATED_DIR)
	protoc -I$(PROTO_DIR) \
		--go_out=$(GENERATED_DIR) --go_opt=paths=source_relative \
		--go-grpc_out=$(GENERATED_DIR) --go-grpc_opt=paths=source_relative \
		$(PROTO_DIR)/*.proto
	@echo "Gotovo: $(GENERATED_DIR)"

build: proto
	@echo "Sborka..."
	go build -o $(BINARY_NAME).exe $(CMD_DIR)/main.go
	@echo "OK: $(BINARY_NAME).exe"

run: build
	@echo "Zapusk..."
	./$(BINARY_NAME).exe -grpc-port=50051 -python-url=localhost:9000

run-dev:
	@echo "Dev mode..."
	go run $(CMD_DIR)/main.go -grpc-port=50051 -python-url=localhost:9000

test:
	@echo "Testy..."
	go test -v -cover ./...

clean:
	@echo "Ochistka..."
	del /Q $(BINARY_NAME).exe 2>nul || rm -f $(BINARY_NAME).exe
	go clean
	@echo "OK"

