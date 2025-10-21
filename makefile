BINARY_NAME=kobot

build:
	@go build -o $(BINARY_NAME) && echo "kobot binary successfully built"

install:
	@go install . && echo "kobot successfully install. Try running 'kobot version'"

# release:
# 	GOOS=linux GOARCH=amd64 go build -o $(BINARY_NAME)-linux-amd64
# 	GOOS=darwin GOARCH=arm64 go build -o $(BINARY_NAME)-darwin-arm64
# 	GOOS=windows GOARCH=amd64 go build -o $(BINARY_NAME)-windows-amd64.exe
