# Supported platforms
PLATFORMS := darwin/386 darwin/amd64

# ARM-based platforms and instruction set versions support
ARM_PLATFORMS := linux/arm
ARM_VERSIONS := 6 7

# Assign operating system and architecture
# Evaluated every time it's used, so use "=" and not ":="
# Requires standard platform description format ("os/arch")
OS = $(@D)
ARCH = $(@F)

# Build locations
RELEASE_DIR := release
BINARY_DIR := bin
INSTALL_DIR := $(HOME)/go/bin

# Build specifics
BINARY := dns-noise
MODULE := github.com/steventblack/$(BINARY)
MODULE_FILES := dns-noise.go domains.go pihole.go noisedb.go

# Build (local)
.PHONY: build
build:
	mkdir -p $(BINARY_DIR)
	go build  -o $(BINARY_DIR)/$(BINARY) $(MODULE_FILES)

# Run (local)
.PHONY: run
run:
	go run $(MODULE_FILES)

# Install (local)
.PHONY: install
install:
	go install $(MODULE)

# Test (local)
.PHONY: test
test: build
	go test $(MODULE)

# Cleanup
.PHONY: clean
clean: 
	go clean -i
	rm -f $(BINARY_DIR)/*
	rm -f $(RELEASE_DIR)/*

# Platform build rules (for release)
$(PLATFORMS):
	mkdir -p $(RELEASE_DIR)
	GOOS=$(OS) GOARCH=$(ARCH) go build -o $(RELEASE_DIR)/$(BINARY)-$(OS)-$(ARCH)

# ARM-based platform build rules (for release)
$(ARM_PLATFORMS):
	mkdir -p $(RELEASE_DIR)
	for ARMV in $(ARM_VERSIONS); do \
		GOOS=$(OS) GOARCH=$(ARCH) GOARM=$$ARMV go build -o $(RELEASE_DIR)/$(BINARY)-$(OS)-$(ARCH)v$$ARMV; \
	done

# Makes all supported platforms
# Other build rules act only locally (i.e. build for current machine, local installs, etc.)
.PHONY: release
release: $(PLATFORMS) $(ARM_PLATFORMS)

# Make builds for the raspberry pi only
.PHONY: pi
pi: linux/arm
