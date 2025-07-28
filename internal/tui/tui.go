// Package tui содержит компоненты для текстового пользовательского интерфейса
package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hazadus/go-snatcher/internal/data"
)

// App представляет основное TUI приложение
type App struct {
	appData *data.AppData
}

// NewApp создает новый экземпляр TUI приложения
func NewApp(appData *data.AppData) *App {
	return &App{
		appData: appData,
	}
}

// Run запускает TUI приложение
func (app *App) Run() error {
	// Создаем модель для Bubble Tea
	model := newMainModel(app.appData)

	// Создаем программу Bubble Tea
	p := tea.NewProgram(model, tea.WithAltScreen())

	// Запускаем программу
	_, err := p.Run()
	return err
}

// mainModel представляет главную модель TUI
type mainModel struct {
	appData *data.AppData
}

// newMainModel создает новую главную модель
func newMainModel(appData *data.AppData) *mainModel {
	return &mainModel{
		appData: appData,
	}
}

// Init инициализирует модель
func (m *mainModel) Init() tea.Cmd {
	return nil
}

// Update обрабатывает сообщения
func (m *mainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		// Можно будет использовать для адаптивного дизайна
	}

	return m, nil
}

// View отображает интерфейс
func (m *mainModel) View() string {
	style := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#0000ff")).
		Padding(1, 2)

	title := style.Render("🎵 Snatcher TUI")

	content := fmt.Sprintf(
		"%s\n\n"+
			"Добро пожаловать в Snatcher TUI!\n"+
			"Треков в библиотеке: %d\n\n"+
			"Нажмите 'q' для выхода",
		title,
		len(m.appData.Tracks),
	)

	return content
}
