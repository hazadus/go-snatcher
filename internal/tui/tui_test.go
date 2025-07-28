// Package tui содержит тесты для TUI компонентов
package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hazadus/go-snatcher/internal/data"
	"github.com/hazadus/go-snatcher/internal/tui/player"
	"github.com/hazadus/go-snatcher/internal/tui/tracklist"
)

func TestMainModelRouting(t *testing.T) {
	// Создаем тестовые данные
	testData := &data.AppData{
		Tracks: []data.TrackMetadata{
			{
				ID:     1,
				Title:  "Test Track",
				Artist: "Test Artist",
				Album:  "Test Album",
			},
		},
	}

	// Создаем главную модель
	model := newMainModel(testData)

	// Проверяем начальное состояние
	if model.currentScreen != tracklistScreen {
		t.Errorf("Expected initial screen to be tracklistScreen, got %v", model.currentScreen)
	}

	if model.tracklistModel == nil {
		t.Error("Expected tracklistModel to be initialized")
	}

	if model.playerModel != nil {
		t.Error("Expected playerModel to be nil initially")
	}

	// Тестируем переключение на экран плеера
	trackSelectedMsg := tracklist.TrackSelectedMsg{
		Track: testData.Tracks[0],
	}

	updatedModel, _ := model.Update(trackSelectedMsg)
	model = updatedModel.(*mainModel)

	if model.currentScreen != playerScreen {
		t.Errorf("Expected screen to be playerScreen after TrackSelectedMsg, got %v", model.currentScreen)
	}

	if model.playerModel == nil {
		t.Error("Expected playerModel to be initialized after TrackSelectedMsg")
	}

	// Тестируем возврат к списку треков
	goBackMsg := player.GoBackMsg{}
	updatedModel, _ = model.Update(goBackMsg)
	model = updatedModel.(*mainModel)

	if model.currentScreen != tracklistScreen {
		t.Errorf("Expected screen to be tracklistScreen after GoBackMsg, got %v", model.currentScreen)
	}

	if model.playerModel != nil {
		t.Error("Expected playerModel to be nil after GoBackMsg")
	}

	// Тестируем глобальные горячие клавиши
	ctrlCMsg := tea.KeyMsg{Type: tea.KeyCtrlC}
	_, cmd := model.Update(ctrlCMsg)

	if cmd == nil {
		t.Error("Expected tea.Quit command after Ctrl+C")
	}
}

func TestMainModelView(t *testing.T) {
	// Создаем тестовые данные
	testData := &data.AppData{
		Tracks: []data.TrackMetadata{
			{
				ID:     1,
				Title:  "Test Track",
				Artist: "Test Artist",
				Album:  "Test Album",
			},
		},
	}

	model := newMainModel(testData)

	// Тестируем отображение списка треков
	view := model.View()
	if view == "" {
		t.Error("Expected non-empty view for tracklist screen")
	}

	// Переключаемся на экран плеера
	trackSelectedMsg := tracklist.TrackSelectedMsg{
		Track: testData.Tracks[0],
	}
	updatedModel, _ := model.Update(trackSelectedMsg)
	model = updatedModel.(*mainModel)

	// Тестируем отображение плеера
	view = model.View()
	if view == "" {
		t.Error("Expected non-empty view for player screen")
	}

	// Тестируем состояние с несуществующим экраном
	model.currentScreen = screenType(999)
	view = model.View()
	expectedError := "Неизвестный экран"
	if view != expectedError {
		t.Errorf("Expected '%s' for unknown screen, got '%s'", expectedError, view)
	}
}
