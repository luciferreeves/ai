BINARY_NAME=ai
BUILD_PATH=bin/$(BINARY_NAME)
MAIN_PATH=$(BINARY_NAME)/main.go
ENV_PATH=.env

.PHONY: setup clean build run dev all

define ensure_setup
	@if [ ! -f $(ENV_PATH) ]; then \
		echo "Running setup first..."; \
		$(MAKE) -s setup; \
	fi
endef

setup:
	@echo "Setting up environment..."
	@go mod download
	@if [ ! -f $(ENV_PATH) ]; then cp .env.example $(ENV_PATH); fi
	@echo "Environment setup complete."

clean:
	@echo "Cleaning up..."
	@rm -rf bin
	@echo "Cleanup complete."

build:
	$(call ensure_setup)
	@echo "Building..."
	@go build -o $(BUILD_PATH) $(MAIN_PATH) || true
	@echo "Build complete."

run:
	$(call ensure_setup)
	@if [ ! -f $(BUILD_PATH) ]; then echo "Binary not found. Building binary..."; $(MAKE) -s build; fi
	@echo "Running..."
	@$(BUILD_PATH) || true

dev:
	$(call ensure_setup)
	@echo "Running in development mode..."
	@go run $(MAIN_PATH) 2>/dev/null || true

all: setup clean build run

.SILENT: