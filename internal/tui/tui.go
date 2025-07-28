// Package tui содержит компоненты для текстового пользовательского интерфейса
package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/hazadus/go-snatcher/internal/data"
	"github.com/hazadus/go-snatcher/internal/player"
	"github.com/hazadus/go-snatcher/internal/tui/editor"
	tuiPlayer "github.com/hazadus/go-snatcher/internal/tui/player"
	"github.com/hazadus/go-snatcher/internal/tui/tracklist"
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
func (app *App) Run() error {
	// Создаем модель для Bubble Tea
	model := newMainModel(app.appData, app.saveFunc)

	// Создаем программу Bubble Tea
	p := tea.NewProgram(model, tea.WithAltScreen())

	// Запускаем программу
	_, err := p.Run()

	// Закрываем плеер после завершения программы
	if model.globalPlayer != nil {
		model.globalPlayer.Close()
	}

	return err
}

// screenType определяет тип текущего экрана
type screenType int

const (
	tracklistScreen screenType = iota
	playerScreen
	editorScreen
)

// mainModel представляет главную модель TUI
type mainModel struct {
	appData        *data.AppData
	currentScreen  screenType
	tracklistModel *tracklist.Model
	playerModel    *tuiPlayer.Model
	editorModel    *editor.Model
	globalPlayer   *player.Player // Глобальный плеер для переиспользования
	saveFunc       func() error   // Функция для сохранения данных
}

// newMainModel создает новую главную модель
func newMainModel(appData *data.AppData, saveFunc func() error) *mainModel {
	// Создаем модель списка треков
	tracklistModel := tracklist.NewModel(appData)

	// Создаем глобальный плеер один раз
	globalPlayer := player.NewPlayer()

	return &mainModel{
		appData:        appData,
		currentScreen:  tracklistScreen,
		tracklistModel: tracklistModel,
		playerModel:    nil, // Будет создана при выборе трека
		editorModel:    nil, // Будет создана при редактировании трека
		globalPlayer:   globalPlayer,
		saveFunc:       saveFunc,
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
			// Останавливаем плеер перед выходом
			if m.globalPlayer != nil {
				m.globalPlayer.Stop()
			}
			return m, tea.Quit
		}

	case tracklist.TrackSelectedMsg:
		// Переключаемся на экран плеера с выбранным треком
		m.currentScreen = playerScreen
		m.playerModel = tuiPlayer.NewModelWithPlayer(msg.Track, m.globalPlayer)
		return m, m.playerModel.Init()

	case tracklist.TrackEditMsg:
		// Переключаемся на экран редактирования с выбранным треком
		m.currentScreen = editorScreen
		m.editorModel = editor.NewModel(m.appData, msg.Track, m.saveFunc)
		return m, m.editorModel.Init()

	case tuiPlayer.GoBackMsg:
		// Возвращаемся к списку треков
		m.currentScreen = tracklistScreen
		m.playerModel = nil
		return m, nil

	case editor.GoBackMsg:
		// Возвращаемся к списку треков из редактора
		m.currentScreen = tracklistScreen
		m.editorModel = nil
		// Обновляем данные в существующей модели списка треков
		m.tracklistModel.RefreshData()
		return m, nil

	case editor.TrackSavedMsg:
		// Трек сохранен - остаемся в редакторе, но можно добавить уведомление
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
				if playerModel, ok := updatedModel.(*tuiPlayer.Model); ok {
					m.playerModel = playerModel
				}
				return m, playerCmd
			}
		case editorScreen:
			if m.editorModel != nil {
				var editorCmd tea.Cmd
				m.editorModel, editorCmd = m.editorModel.Update(msg)
				return m, editorCmd
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
			if playerModel, ok := updatedModel.(*tuiPlayer.Model); ok {
				m.playerModel = playerModel
			}
			cmd = playerCmd
		}

	case editorScreen:
		if m.editorModel != nil {
			var editorCmd tea.Cmd
			m.editorModel, editorCmd = m.editorModel.Update(msg)
			cmd = editorCmd
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

	case editorScreen:
		if m.editorModel != nil {
			return m.editorModel.View()
		}
		return "Ошибка: модель редактора не инициализирована"

	default:
		return "Неизвестный экран"
	}
}
