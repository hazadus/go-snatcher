// Package main - snatcher CLI tool
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/hazadus/go-snatcher/internal/config"
	"github.com/hazadus/go-snatcher/internal/data"
	"github.com/spf13/cobra"
)

const (
	defaultConfigPath   = "~/.snatcher"
	defaultDataFilePath = "~/.snatcher_data"
)

// Application содержит все зависимости приложения
type Application struct {
	Config *config.Config
	Data   *data.AppData
}

// NewApplication создает новый экземпляр приложения
func NewApplication() *Application {
	return &Application{}
}

// Initialize инициализирует приложение - загружает конфигурацию и данные
func (app *Application) Initialize() error {
	var err error

	// Загружаем конфигурацию приложения
	if app.Config, err = config.LoadConfig(defaultConfigPath); err != nil {
		return fmt.Errorf("ошибка загрузки конфигурации: %w", err)
	}

	// Инициализируем структуру данных приложения
	app.Data = data.NewAppData()
	if err := app.Data.LoadData(defaultDataFilePath); err != nil {
		return fmt.Errorf("ошибка загрузки данных приложения: %w", err)
	}

	return nil
}

// SaveData сохраняет данные приложения
func (app *Application) SaveData() error {
	return app.Data.SaveData(defaultDataFilePath)
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Создаем контекст с обработкой сигналов прерывания
	ctx, cancel := createContextWithSignalHandling()
	defer cancel()

	// Создаем экземпляр приложения
	app := NewApplication()

	// Инициализируем приложение
	if err := app.Initialize(); err != nil {
		return err
	}

	// Создаем корневую команду
	rootCmd := app.createRootCommand(ctx)

	return rootCmd.Execute()
}

// createContextWithSignalHandling создает контекст с обработкой сигналов прерывания
func createContextWithSignalHandling() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())

	// Создаем канал для сигналов прерывания
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Запускаем горутину для обработки сигналов
	go func() {
		<-sigChan
		fmt.Println("\n🚫 Получен сигнал прерывания, отменяем операции...")
		cancel()
	}()

	return ctx, cancel
}

// createRootCommand создает корневую команду с настроенными подкомандами
func (app *Application) createRootCommand(ctx context.Context) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "snatcher",
		Short: "A simple command line tool to manage and play mp3 files",
		Long:  `A simple command line tool to manage and play mp3 files from local path or URL.`,
	}

	// Добавляем команды, передавая в них экземпляр приложения и контекст
	rootCmd.AddCommand(app.createAddCommand(ctx))
	rootCmd.AddCommand(app.createListCommand())
	rootCmd.AddCommand(app.createPlayCommand(ctx))
	rootCmd.AddCommand(app.createDownloadCommand(ctx))

	return rootCmd
}
