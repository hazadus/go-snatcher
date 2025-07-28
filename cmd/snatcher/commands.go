package main

import (
	"context"

	"github.com/spf13/cobra"
)

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
	rootCmd.AddCommand(app.createDeleteCommand(ctx))
	rootCmd.AddCommand(app.createTUICommand())

	return rootCmd
}
