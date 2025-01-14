# Cross-platform builds
BINARY_NAME=jbmdb
VERSION=2.0.0
BUILD_DIR=build
BUILD_FLAGS=-ldflags="-X main.Version=$(VERSION)"

.PHONY: all build clean build-linux build-windows build-darwin install-linux test

all: clean build

build: build-linux build-windows build-darwin

clean:
	rm -rf $(BUILD_DIR)
	mkdir -p $(BUILD_DIR)

build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux ./migrations

build-windows:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build $(BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows.exe ./migrations

build-darwin:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build $(BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin ./migrations

install-linux: build-linux
	sudo mv $(BUILD_DIR)/$(BINARY_NAME)-linux /usr/local/bin/$(BINARY_NAME)
	sudo chmod +x /usr/local/bin/$(BINARY_NAME)

version:
	@echo $(VERSION)

# Test the binary
test: build-linux
	@echo "Testing command execution (no database connections)..."
	@echo "\nTesting configuration..."
	./$(BUILD_DIR)/$(BINARY_NAME)-linux config

	@echo "\nTesting PostgreSQL user and database management..."
	./$(BUILD_DIR)/$(BINARY_NAME)-linux postgres-create-user:read
	./$(BUILD_DIR)/$(BINARY_NAME)-linux postgres-create-db
	./$(BUILD_DIR)/$(BINARY_NAME)-linux postgres-grant:read
	@echo "\nTesting PostgreSQL migration commands..."
	./$(BUILD_DIR)/$(BINARY_NAME)-linux postgres-migration create_tests_table
	./$(BUILD_DIR)/$(BINARY_NAME)-linux postgres-list || true

	@echo "\nTesting MySQL user and database management..."
	./$(BUILD_DIR)/$(BINARY_NAME)-linux mysql-create-user:read
	./$(BUILD_DIR)/$(BINARY_NAME)-linux mysql-create-db
	./$(BUILD_DIR)/$(BINARY_NAME)-linux mysql-grant:read
	@echo "\nTesting MySQL migration commands..."
	./$(BUILD_DIR)/$(BINARY_NAME)-linux mysql-migration create_tests_table
	./$(BUILD_DIR)/$(BINARY_NAME)-linux mysql-list || true

	@echo "\nTesting CQL keyspace and user management..."
	./$(BUILD_DIR)/$(BINARY_NAME)-linux cql-create-keyspace:SimpleStrategy:3
	./$(BUILD_DIR)/$(BINARY_NAME)-linux cql-create-user:read
	@echo "\nTesting CQL migration commands..."
	./$(BUILD_DIR)/$(BINARY_NAME)-linux cql-migration create_tests_table
	./$(BUILD_DIR)/$(BINARY_NAME)-linux cql-list || true

	@echo "\nVerifying directory structure..."
	@ls -R $(shell pwd)/psql/sql $(shell pwd)/squel/sql $(shell pwd)/cassy/cql || true
	@echo "\nTest completed successfully!"
