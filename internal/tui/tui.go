// Package tui содержит компоненты для текстового пользовательского интерфейса
package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/hazadus/go-snatcher/internal/data"
	"github.com/hazadus/go-snatcher/internal/tui/app"
)

// App представляет основное TUI приложение
type App struct {
	appData  *data.AppData
	saveFunc func() error // Функция для сохранения данных
}

// NewApp создает новый экземпляр TUI приложения
func NewApp(appData *data.AppData, saveFunc func() error) *App {
	return &App{
		appData:  appData,
		saveFunc: saveFunc,
	}
}

// Run запускает TUI приложение
func (tuiApp *App) Run() error {
	// Создаем модель для Bubble Tea
	model := app.NewMainModel(tuiApp.appData, tuiApp.saveFunc)

	// Создаем программу Bubble Tea
	p := tea.NewProgram(model, tea.WithAltScreen())

	// Запускаем программу
	_, err := p.Run()

	// Закрываем плеер после завершения программы
	model.Close()

	return err
}
