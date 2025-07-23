package main

import (
	"fmt"
	"log"
	"os"

	"github.com/hazadus/go-snatcher/internal/config"
)

const (
	defaultConfigPath = "~/.snatcher"
)

var (
	cfg *config.Config
)

func init() {
	rootCmd.AddCommand(addCmd)
}

func main() {
	var err error

	// Загружаем конфигурацию приложения
	if cfg, err = config.LoadConfig(defaultConfigPath); err != nil {
		log.Fatalf("Ошибка загрузки конфигурации: %v", err)
	}

	fmt.Println("Используется бакет:", cfg.AwsBucketName)

	execute()
}

func execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
