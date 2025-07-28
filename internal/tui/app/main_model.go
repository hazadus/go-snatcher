// Package app содержит основную логику TUI приложения
package app

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/hazadus/go-snatcher/internal/data"
	"github.com/hazadus/go-snatcher/internal/player"
	"github.com/hazadus/go-snatcher/internal/tui/editor"
	tuiPlayer "github.com/hazadus/go-snatcher/internal/tui/player"
	"github.com/hazadus/go-snatcher/internal/tui/tracklist"
)

// ScreenType определяет тип текущего экрана
type ScreenType int

// Константы для типов экранов
const (
	// TracklistScreen - экран списка треков
	TracklistScreen ScreenType = iota
	// PlayerScreen - экран плеера
	PlayerScreen
	// EditorScreen - экран редактирования
	EditorScreen
)

// MainModel представляет главную модель TUI
type MainModel struct {
	appData        *data.AppData
	currentScreen  ScreenType
	tracklistModel *tracklist.Model
	playerModel    *tuiPlayer.Model
	editorModel    *editor.Model
	globalPlayer   *player.Player // Глобальный плеер для переиспользования
	saveFunc       func() error   // Функция для сохранения данных
}

// NewMainModel создает новую главную модель
func NewMainModel(appData *data.AppData, saveFunc func() error) *MainModel {
	// Создаем модель списка треков
	tracklistModel := tracklist.NewModel(appData)

	// Создаем глобальный плеер один раз
	globalPlayer := player.NewPlayer()

	return &MainModel{
		appData:        appData,
		currentScreen:  TracklistScreen,
		tracklistModel: tracklistModel,
		playerModel:    nil, // Будет создана при выборе трека
		editorModel:    nil, // Будет создана при редактировании трека
		globalPlayer:   globalPlayer,
		saveFunc:       saveFunc,
	}
}

// Init инициализирует модель
func (m *MainModel) Init() tea.Cmd {
	// Инициализируем модель списка треков
	return m.tracklistModel.Init()
}

// Update обрабатывает сообщения
func (m *MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		m.currentScreen = PlayerScreen
		m.playerModel = tuiPlayer.NewModelWithPlayer(msg.Track, m.globalPlayer, m.appData, m.saveFunc)
		return m, m.playerModel.Init()

	case tracklist.TrackEditMsg:
		// Переключаемся на экран редактирования с выбранным треком
		m.currentScreen = EditorScreen
		m.editorModel = editor.NewModel(m.appData, msg.Track, m.saveFunc)
		return m, m.editorModel.Init()

	case tuiPlayer.GoBackMsg:
		// Возвращаемся к списку треков
		m.currentScreen = TracklistScreen
		m.playerModel = nil
		return m, nil

	case editor.GoBackMsg:
		// Возвращаемся к списку треков из редактора
		m.currentScreen = TracklistScreen
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
		case TracklistScreen:
			var tracklistCmd tea.Cmd
			m.tracklistModel, tracklistCmd = m.tracklistModel.Update(msg)
			return m, tracklistCmd
		case PlayerScreen:
			if m.playerModel != nil {
				var playerCmd tea.Cmd
				updatedModel, playerCmd := m.playerModel.Update(msg)
				if playerModel, ok := updatedModel.(*tuiPlayer.Model); ok {
					m.playerModel = playerModel
				}
				return m, playerCmd
			}
		case EditorScreen:
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
	case TracklistScreen:
		var tracklistCmd tea.Cmd
		m.tracklistModel, tracklistCmd = m.tracklistModel.Update(msg)
		cmd = tracklistCmd

	case PlayerScreen:
		if m.playerModel != nil {
			var playerCmd tea.Cmd
			updatedModel, playerCmd := m.playerModel.Update(msg)
			if playerModel, ok := updatedModel.(*tuiPlayer.Model); ok {
				m.playerModel = playerModel
			}
			cmd = playerCmd
		}

	case EditorScreen:
		if m.editorModel != nil {
			var editorCmd tea.Cmd
			m.editorModel, editorCmd = m.editorModel.Update(msg)
			cmd = editorCmd
		}
	}

	return m, cmd
}

// View отображает интерфейс
func (m *MainModel) View() string {
	switch m.currentScreen {
	case TracklistScreen:
		return m.tracklistModel.View()

	case PlayerScreen:
		if m.playerModel != nil {
			return m.playerModel.View()
		}
		return "Ошибка: модель плеера не инициализирована"

	case EditorScreen:
		if m.editorModel != nil {
			return m.editorModel.View()
		}
		return "Ошибка: модель редактора не инициализирована"

	default:
		return "Неизвестный экран"
	}
}

// Close закрывает ресурсы главной модели
func (m *MainModel) Close() {
	if m.globalPlayer != nil {
		m.globalPlayer.Close()
	}
}
