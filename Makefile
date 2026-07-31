.PHONY: all build test clean

BINARY := bin/etsy

all: build

build:
	@mkdir -p bin
	go build -o $(BINARY) .

test:
	go test ./...

clean:
	rm -f $(BINARY)
