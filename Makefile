.PHONY: all build test clean run benchmark serve docker-build docker-run

BINARY_NAME=hallucidetect

all: test build

build:
	go build -v -ldflags="-s -w" -o $(BINARY_NAME) ./cmd/hallucidetect

test:
	go test -v -race ./...

benchmark: build
	./$(BINARY_NAME) benchmark

serve: build
	./$(BINARY_NAME) serve --port 8080

clean:
	go clean
	rm -f $(BINARY_NAME) $(BINARY_NAME).exe
	rm -rf dist/

docker-build:
	docker build -t hallucidetect:latest .

docker-run:
	docker run -p 8080:8080 hallucidetect:latest
