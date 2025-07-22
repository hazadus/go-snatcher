run:
  go run cmd/snatcher/main.go

build:
  go build -o ./bin/snatcher cmd/snatcher/main.go

format:
  go fmt ./...

lint:
  golangci-lint run ./...
