// Package tui содержит тесты для TUI компонентов
package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hazadus/go-snatcher/internal/data"
	"github.com/hazadus/go-snatcher/internal/tui/app"
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

	// Создаем функцию сохранения для тестов
	saveFunc := func() error {
		return nil
	}

	// Создаем главную модель
	model := app.NewMainModel(testData, saveFunc)

	// Проверяем начальное состояние
	// Поскольку поля модели теперь приватные, проверяем через поведение
	view := model.View()
	if view == "" {
		t.Error("Expected non-empty view for initial state")
	}

	// Тестируем переключение на экран плеера
	trackSelectedMsg := tracklist.TrackSelectedMsg{
		Track: testData.Tracks[0],
	}

	updatedModel, _ := model.Update(trackSelectedMsg)
	model = updatedModel.(*app.MainModel)

	// Проверяем, что экран переключился (через изменение view)
	newView := model.View()
	if newView == view {
		t.Error("Expected view to change after TrackSelectedMsg")
	}

	// Тестируем возврат к списку треков
	goBackMsg := player.GoBackMsg{}
	updatedModel, _ = model.Update(goBackMsg)
	model = updatedModel.(*app.MainModel)

	// Проверяем, что вернулись к исходному виду
	backView := model.View()
	if backView == newView {
		t.Error("Expected view to change back after GoBackMsg")
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

	// Создаем функцию сохранения для тестов
	saveFunc := func() error {
		return nil
	}

	model := app.NewMainModel(testData, saveFunc)

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
	model = updatedModel.(*app.MainModel)

	// Тестируем отображение плеера
	playerView := model.View()
	if playerView == "" {
		t.Error("Expected non-empty view for player screen")
	}

	// Проверяем, что вид изменился
	if playerView == view {
		t.Error("Expected different view for player screen compared to tracklist")
	}
}
