package main

import (
	"fmt"
	"log"
	"os"

	"github.com/hazadus/go-snatcher/internal/config"
	"github.com/hazadus/go-snatcher/internal/data"
)

const (
	defaultConfigPath   = "~/.snatcher"
	defaultDataFilePath = "~/.snatcher_data"
)

var (
	cfg     *config.Config
	appData *data.AppData
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

	// Инициализируем структуру данных приложения
	appData = data.NewAppData()
	if err := appData.LoadData(defaultDataFilePath); err != nil {
		log.Fatalf("Ошибка загрузки данных приложения: %v", err)
	}

	execute()
}

func execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
