# Go パラメータ
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
BINARY_PATH=bin
BINARY_NAME=obs-controller
BINARY_UNIX=$(BINARY_NAME).linux

all: build build-linux
build:
	$(GOBUILD) -o $(BINARY_PATH)/$(BINARY_NAME) -v
test:
	$(GOTEST) -v ./...
clean:
	$(GOCLEAN)
	rm -f $(BINARY_PATH)/$(BINARY_NAME)
	rm -f $(BINARY_PATH)/$(BINARY_UNIX)
run:
	$(GOBUILD) -o $(BINARY_PATH)/$(BINARY_NAME) -v ./...
	./$(BINARY_NAME)
deps:
	$(GOGET) github.com/markbates/goth
	$(GOGET) github.com/markbates/pop


# cross compile
build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BINARY_PATH)/$(BINARY_UNIX) -v
