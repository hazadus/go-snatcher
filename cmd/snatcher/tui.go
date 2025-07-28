package main

import (
	"github.com/hazadus/go-snatcher/internal/tui"
	"github.com/spf13/cobra"
)

// createTUICommand создает команду tui с привязкой к экземпляру приложения
func (app *Application) createTUICommand() *cobra.Command {
	return &cobra.Command{
		Use:   "tui",
		Short: "Launch TUI (Terminal User Interface)",
		Long:  `Launch interactive terminal user interface for managing and playing tracks.`,
		Run: func(_ *cobra.Command, _ []string) {
			app.launchTUI()
		},
	}
}

func (app *Application) launchTUI() {
	// Создаем экземпляр TUI приложения
	tuiApp := tui.NewApp(app.Data)

	// Запускаем TUI
	if err := tuiApp.Run(); err != nil {
		// Если есть ошибка, выводим её и выходим
		// В реальном приложении можно было бы обработать это лучше
		panic(err)
	}
}
