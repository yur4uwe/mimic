# Define output binary names
BINARY_NAME=mimic
BUILD_DIR=build

# Detect the operating system
UNAME_S := $(shell uname -s)

# Set GOOS and GOARCH based on the detected OS
ifeq ($(UNAME_S), Linux)
	GOOS := linux
	GOARCH := amd64
	OUTPUT := $(BUILD_DIR)/$(BINARY_NAME)_linux
else ifeq ($(UNAME_S), Darwin)
	GOOS := darwin
	GOARCH := amd64
	OUTPUT := $(BUILD_DIR)/$(BINARY_NAME)_darwin
else ifneq (,$(findstring MINGW64_NT,$(UNAME_S)))
	GOOS := windows
	GOARCH := amd64
	OUTPUT := $(BUILD_DIR)/$(BINARY_NAME)_windows.exe
else
	$(error Unsupported OS: $(UNAME_S))
endif

# Default target: build for the current platform
all: $(OUTPUT)

# Build for the current platform
$(OUTPUT):
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o $@ ./cmd/main

# Build for Linux
linux:
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)_linux ./cmd/main

# Build for macOS
darwin:
	GOOS=darwin GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)_darwin ./cmd/main

# Build for Windows
windows:
	GOOS=windows GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)_windows.exe ./cmd/main

# Clean build artifacts
clean:
	rm -rf $(BUILD_DIR)

# Phony targets
.PHONY: all linux darwin windows clean