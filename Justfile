run:
  go run cmd/snatcher/main.go

build:
  go build -o ./bin/snatcher cmd/snatcher/main.go

format:
  go fmt ./...

lint:
  golangci-lint run ./...

# Сгенерировать сообщение коммита (см. https://github.com/hazadus/gh-commitmsg)
commitmsg:
    gh commitmsg --language russian --examples