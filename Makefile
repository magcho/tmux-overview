.PHONY: build install clean

BINARY_NAME=tov
BUILD_DIR=.

build:
	go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/tov

install: build
	mkdir -p ~/.local/bin
	cp $(BUILD_DIR)/$(BINARY_NAME) ~/.local/bin/$(BINARY_NAME)

clean:
	rm -f $(BUILD_DIR)/$(BINARY_NAME)
