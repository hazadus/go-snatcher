// Package tui содержит компоненты для текстового пользовательского интерфейса
package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/hazadus/go-snatcher/internal/data"
	"github.com/hazadus/go-snatcher/internal/tui/player"
	"github.com/hazadus/go-snatcher/internal/tui/tracklist"
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

// screenType определяет тип текущего экрана
type screenType int

const (
	tracklistScreen screenType = iota
	playerScreen
)

// mainModel представляет главную модель TUI
type mainModel struct {
	appData        *data.AppData
	currentScreen  screenType
	tracklistModel *tracklist.Model
	playerModel    *player.Model
}

// newMainModel создает новую главную модель
func newMainModel(appData *data.AppData) *mainModel {
	// Создаем модель списка треков
	tracklistModel := tracklist.NewModel(appData)

	return &mainModel{
		appData:        appData,
		currentScreen:  tracklistScreen,
		tracklistModel: tracklistModel,
		playerModel:    nil, // Будет создана при выборе трека
	}
}

// Init инициализирует модель
func (m *mainModel) Init() tea.Cmd {
	// Инициализируем модель списка треков
	return m.tracklistModel.Init()
}

// Update обрабатывает сообщения
func (m *mainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Глобальные горячие клавиши
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		}

	case tracklist.TrackSelectedMsg:
		// Переключаемся на экран плеера с выбранным треком
		m.currentScreen = playerScreen
		m.playerModel = player.NewModel(msg.Track)
		return m, m.playerModel.Init()

	case player.GoBackMsg:
		// Возвращаемся к списку треков
		m.currentScreen = tracklistScreen
		m.playerModel = nil
		return m, nil

	case tea.WindowSizeMsg:
		// Передаем размеры окна активной модели
		switch m.currentScreen {
		case tracklistScreen:
			var tracklistCmd tea.Cmd
			m.tracklistModel, tracklistCmd = m.tracklistModel.Update(msg)
			return m, tracklistCmd
		case playerScreen:
			if m.playerModel != nil {
				var playerCmd tea.Cmd
				updatedModel, playerCmd := m.playerModel.Update(msg)
				if playerModel, ok := updatedModel.(*player.Model); ok {
					m.playerModel = playerModel
				}
				return m, playerCmd
			}
		}
		return m, nil
	}

	// Передаем сообщение активной модели
	switch m.currentScreen {
	case tracklistScreen:
		var tracklistCmd tea.Cmd
		m.tracklistModel, tracklistCmd = m.tracklistModel.Update(msg)
		cmd = tracklistCmd

	case playerScreen:
		if m.playerModel != nil {
			var playerCmd tea.Cmd
			updatedModel, playerCmd := m.playerModel.Update(msg)
			if playerModel, ok := updatedModel.(*player.Model); ok {
				m.playerModel = playerModel
			}
			cmd = playerCmd
		}
	}

	return m, cmd
}

// View отображает интерфейс
func (m *mainModel) View() string {
	switch m.currentScreen {
	case tracklistScreen:
		return m.tracklistModel.View()

	case playerScreen:
		if m.playerModel != nil {
			return m.playerModel.View()
		}
		return "Ошибка: модель плеера не инициализирована"

	default:
		return "Неизвестный экран"
	}
}
