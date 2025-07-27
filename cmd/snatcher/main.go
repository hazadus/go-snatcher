// Package main - snatcher CLI tool
package main

import (
	"fmt"
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
	rootCmd.AddCommand(downloadCmd)
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	var err error

	// Загружаем конфигурацию приложения
	if cfg, err = config.LoadConfig(defaultConfigPath); err != nil {
		return fmt.Errorf("ошибка загрузки конфигурации: %w", err)
	}

	// Инициализируем структуру данных приложения
	appData = data.NewAppData()
	if err := appData.LoadData(defaultDataFilePath); err != nil {
		return fmt.Errorf("ошибка загрузки данных приложения: %w", err)
	}

	return execute()
}

func execute() error {
	if err := rootCmd.Execute(); err != nil {
		return fmt.Errorf("ошибка выполнения команды: %w", err)
	}
	return nil
}
