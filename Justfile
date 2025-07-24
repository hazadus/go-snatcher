run:
  go run cmd/snatcher/main.go

build:
  go build -o ./bin/snatcher cmd/snatcher/*.go

format:
  go fmt ./...

lint:
  golangci-lint run ./...

# Сгенерировать сообщение коммита (см. https://github.com/hazadus/gh-commitmsg)
commitmsg:
    gh commitmsg --language russian --examples

# Посчитать строки кода в проекте и сохранить в файл
cloc:
    cloc --fullpath --exclude-list-file=.clocignore --md . > cloc.md