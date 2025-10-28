# ============================================================
#  Kobot Build + Release Makefile (Windows + ZIP + S3 Upload)
# ============================================================

BINARY_NAME := kobot
APP_NAME := kobot
PKG_PATH := gitlab.com/kobot/kobot/cmd
DIST := dist
ARCH := amd64
OS := windows
BINNAME := $(APP_NAME)-$(VERSION)-$(OS)-$(ARCH)
EXE := $(BINNAME).exe
ZIP := $(DIST)/$(BINNAME).zip

# Defaults
VERSION ?= v0.0.0
GOFLAGS ?=
CGO_ENABLED ?= 0
BUCKET ?=

# ============================================================
#  ANSI Color Codes
# ============================================================
GREEN  := \033[1;32m
YELLOW := \033[1;33m
BLUE   := \033[1;34m
RED    := \033[1;31m
RESET  := \033[0m
BOLD   := \033[1m

# ============================================================
#  Build (local)
# ============================================================
build:
	@echo "$(BLUE)[INFO]$(RESET) Building $(BINARY_NAME)..."
	@go build -o $(BINARY_NAME)
	@echo "$(GREEN)[SUCCESS]$(RESET) $(BINARY_NAME) binary successfully built."

# ============================================================
#  Install locally with version injection
# ============================================================
install:
ifeq ($(VERSION), "")
	@echo "$(YELLOW)[WARN]$(RESET) No version provided. Using default version v0.0.0"
	@go install . && echo "$(GREEN)[SUCCESS]$(RESET) $(BINARY_NAME) installed." && $(BINARY_NAME) version
else
	@echo "$(BLUE)[INFO]$(RESET) Installing $(BINARY_NAME) $(VERSION)..."
	@go install -ldflags="-X=$(PKG_PATH).CliVersion=$(VERSION)" . \
	&& echo "$(GREEN)[SUCCESS]$(RESET) $(BINARY_NAME) installed." && $(BINARY_NAME) version
endif

# ============================================================
#  Build Windows binary
# ============================================================
build-windows:
	@mkdir -p $(DIST)
	@echo ""
	@echo "$(BOLD)----------------------------------------$(RESET)"
	@echo "$(BLUE)[INFO]$(RESET) Building $(APP_NAME) for $(OS)/$(ARCH)"
	@echo "$(BLUE)[INFO]$(RESET) Version: $(VERSION)"
	@echo "$(BOLD)----------------------------------------$(RESET)"
	@GOOS=$(OS) GOARCH=$(ARCH) CGO_ENABLED=$(CGO_ENABLED) \
	go build $(GOFLAGS) -ldflags="-X=$(PKG_PATH).CliVersion=$(VERSION)" -o $(DIST)/$(EXE)
	@echo "$(GREEN)[SUCCESS]$(RESET) Binary created at $(DIST)/$(EXE)"
	@echo ""

# ============================================================
#  Zip the Windows binary
# ============================================================
zip-windows: build-windows
	@echo "$(BLUE)[INFO]$(RESET) Creating ZIP archive..."
	@cd $(DIST) && zip -q $(BINNAME).zip $(EXE)
	@echo "$(GREEN)[SUCCESS]$(RESET) Created archive: $(ZIP)"

# ============================================================
#  Upload ZIP to S3 (if BUCKET is set)
# ============================================================
upload: zip-windows
ifndef BUCKET
	$(error $(RED)[ERROR]$(RESET) BUCKET is not set. Example: make upload BUCKET=my-bucket)
endif
	@echo "$(BLUE)[INFO]$(RESET) Uploading $(ZIP) to s3://$(BUCKET)/ ..."
	@aws s3 cp $(ZIP) s3://$(BUCKET)/ --acl bucket-owner-full-control
	@echo "$(GREEN)[SUCCESS]$(RESET) Upload complete: s3://$(BUCKET)/$(BINNAME).zip"

# ============================================================
#  Release (build + zip + optional upload)
# ============================================================
release:
ifeq ($(strip $(BUCKET)),)
	@$(MAKE) zip-windows
	@echo "$(YELLOW)[WARN]$(RESET) No BUCKET provided — skipping upload."
	@echo "$(BLUE)[INFO]$(RESET) Artifacts available in: $(DIST)/"
	@ls -lah $(DIST)
else
	@$(MAKE) upload
endif

# ============================================================
#  Clean artifacts
# ============================================================
clean:
	@echo "$(BLUE)[INFO]$(RESET) Cleaning $(DIST)..."
	@rm -rf $(DIST)
	@echo "$(GREEN)[SUCCESS]$(RESET) Done."
