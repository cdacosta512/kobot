BINARY_NAME="kobot"
VERSION=""

build:
	@go build -o $(BINARY_NAME) && echo "kobot binary successfully built"

install:
ifeq ($(VERSION), "")
	@echo "WARN: No version was passed. Using default version v0.0.0"
	@go install . && echo "kobot successfully installed." && kobot version
else
	@go install -ldflags="-X=gitlab.com/kobot/kobot/cmd.CliVersion=$(VERSION)" . && echo "kobot successfully installed." && kobot version
endif


# release:
# 	GOOS=linux GOARCH=amd64 go build -o $(BINARY_NAME)-linux-amd64
# 	GOOS=darwin GOARCH=arm64 go build -o $(BINARY_NAME)-darwin-arm64
# 	GOOS=windows GOARCH=amd64 go build -o $(BINARY_NAME)-windows-amd64.exe
