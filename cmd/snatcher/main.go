// Package main - snatcher CLI tool
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/hazadus/go-snatcher/internal/config"
	"github.com/hazadus/go-snatcher/internal/data"
	"github.com/spf13/cobra"
)

const (
	defaultConfigPath   = "~/.snatcher"
	defaultDataFilePath = "~/.snatcher_data"
)

var (
	cfg     *config.Config
	appData *data.AppData
	rootCmd = &cobra.Command{
		Use:   "snatcher",
		Short: "A simple command line tool to manage and play mp3 files",
		Long:  `A simple command line tool to manage and play mp3 files from local path or URL.`,
	}
)

func init() {
	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(playCmd)
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
