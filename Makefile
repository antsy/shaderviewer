# Please note that all this is for Windows

BUILD_TIME=$(shell date +"%Y.%m.%d-%H:%M:%S")
GIT_HASH=$(shell git log --pretty=format:%H -n 1)
BUILD_DIR=bin
GOCMD=go
GOBUILD=$(GOCMD) build
GOOS=windows
GOARCH=amd64

compile:
	$(GOBUILD) -buildmode=exe -ldflags="-X 'main.BuildTime=$(BUILD_TIME)' -X 'main.CommitHash=$(GIT_HASH)'" -o $(BUILD_DIR)/shaderviewer.exe src/shaderviewer.go

compilewingui:
# Will be built without console output and with size optimizing flags
	$(GOBUILD) -buildmode=exe -ldflags="-H=windowsgui -s -w -X 'main.BuildTime=$(BUILD_TIME)' -X 'main.CommitHash=$(GIT_HASH)'" -o $(BUILD_DIR)/shaderviewer.exe src/shaderviewer.go

deps:
	go mod tidy

run:
	bin/shaderviewer.exe -f data\\debug.frag -r

build: | deps compile run

