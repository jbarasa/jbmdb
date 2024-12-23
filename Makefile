# PostgreSQL Migrations
postgres-migration:
	@if [ -z "$(name)" ]; then \
		echo "Error: Migration name is required"; \
		echo "Usage: make postgres-migration name=<migration_name>"; \
		exit 1; \
	fi
	@if echo "$(name)" | grep -q -E 'create_[a-z]+_table' && ! echo "$(name)" | grep -q -E 'create_[a-z]+s_table'; then \
		echo "Error: Single table names should be plural"; \
		echo "Example: 'create_user_table' should be 'create_users_table'"; \
		exit 1; \
	fi
	@if echo "$(name)" | grep -q -E 'create_[a-z]+_[a-z]+[^s]_table$$'; then \
		echo "Error: In relation tables, names after the first word should be plural"; \
		echo "Example: 'create_user_comment_table' should be 'create_user_comments_table'"; \
		exit 1; \
	fi
	@if ! echo "$(name)" | grep -q '^create_[a-z_]\+_table$$'; then \
		echo "Error: Migration name must follow format: create_<name>_table"; \
		echo "Example: create_users_table, create_post_comments_table"; \
		exit 1; \
	fi
	go run ./migrations/main.go postgres-migration $(name)

.PHONY: postgres-migration

postgres-migrate:
	go run ./migrations/main.go postgres-migrate

.PHONY: postgres-migrate

postgres-rollback:
	go run ./migrations/main.go postgres-rollback

.PHONY: postgres-rollback

postgres-fresh:
	go run ./migrations/main.go postgres-fresh

.PHONY: postgres-fresh

postgres-list:
	go run ./migrations/main.go postgres-list

.PHONY: postgres-list

# ScyllaDB Migrations
scylla-migration:
	@if [ -z "$(name)" ]; then \
		echo "Error: Migration name is required"; \
		echo "Usage: make scylla-migration name=<migration_name>"; \
		exit 1; \
	fi
	@if echo "$(name)" | grep -q -E 'create_[a-z]+_table' && ! echo "$(name)" | grep -q -E 'create_[a-z]+s_table'; then \
		echo "Error: Single table names should be plural"; \
		echo "Example: 'create_user_table' should be 'create_users_table'"; \
		exit 1; \
	fi
	@if echo "$(name)" | grep -q -E 'create_[a-z]+_[a-z]+[^s]_table$$'; then \
		echo "Error: In relation tables, names after the first word should be plural"; \
		echo "Example: 'create_user_comment_table' should be 'create_user_comments_table'"; \
		exit 1; \
	fi
	@if ! echo "$(name)" | grep -q '^create_[a-z_]\+_table$$'; then \
		echo "Error: Migration name must follow format: create_<name>_table"; \
		echo "Example: create_users_table, create_post_comments_table"; \
		exit 1; \
	fi
	go run ./migrations/main.go scylla-migration $(name)

.PHONY: scylla-migration

scylla-migrate:
	go run ./migrations/main.go scylla-migrate

.PHONY: scylla-migrate

scylla-rollback:
	go run ./migrations/main.go scylla-rollback

.PHONY: scylla-rollback

scylla-fresh:
	go run ./migrations/main.go scylla-fresh

.PHONY: scylla-fresh

scylla-list:
	go run ./migrations/main.go scylla-list

.PHONY: scylla-list

# Configuration
postgres-init:
	go run ./migrations/main.go postgres-init

.PHONY: postgres-init

scylla-init:
	go run ./migrations/main.go scylla-init

.PHONY: scylla-init

config:
	go run ./migrations/main.go config

.PHONY: config

# Cross-platform builds
BINARY_NAME=jbmdb
VERSION=1.0.0
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
	@echo "Testing PostgreSQL commands..."
	./$(BUILD_DIR)/$(BINARY_NAME)-linux config
	./$(BUILD_DIR)/$(BINARY_NAME)-linux postgres-init
	./$(BUILD_DIR)/$(BINARY_NAME)-linux postgres-make create_tests_table
	./$(BUILD_DIR)/$(BINARY_NAME)-linux postgres-list
	@echo "\nTesting ScyllaDB commands..."
	./$(BUILD_DIR)/$(BINARY_NAME)-linux scylla-init
	./$(BUILD_DIR)/$(BINARY_NAME)-linux scylla-migration create_tests_table
	./$(BUILD_DIR)/$(BINARY_NAME)-linux scylla-list
