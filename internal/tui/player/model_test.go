package player

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hazadus/go-snatcher/internal/data"
)

func TestNewModel(t *testing.T) {
	track := data.TrackMetadata{
		ID:     1,
		Artist: "Test Artist",
		Title:  "Test Title",
		Album:  "Test Album",
		Length: 180,
		URL:    "https://example.com/test.mp3",
	}

	model := NewModel(track)

	if model == nil {
		t.Fatal("NewModel returned nil")
	}

	if model.track.ID != track.ID {
		t.Errorf("Expected track ID %d, got %d", track.ID, model.track.ID)
	}

	if model.isPlaying {
		t.Error("Expected isPlaying to be false initially")
	}

	if model.player == nil {
		t.Error("Player should be initialized")
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		expected string
	}{
		{0, "00:00"},
		{30 * time.Second, "00:30"},
		{1 * time.Minute, "01:00"},
		{1*time.Minute + 30*time.Second, "01:30"},
		{10*time.Minute + 5*time.Second, "10:05"},
	}

	for _, test := range tests {
		result := formatDuration(test.duration)
		if result != test.expected {
			t.Errorf("formatDuration(%v) = %s, expected %s", test.duration, result, test.expected)
		}
	}
}

func TestFormatStatus(t *testing.T) {
	if formatStatus(true) != "Воспроизведение" {
		t.Error("Expected 'Воспроизведение' for playing status")
	}

	if formatStatus(false) != "Пауза" {
		t.Error("Expected 'Пауза' for paused status")
	}
}

func TestUpdateWindowSize(t *testing.T) {
	track := data.TrackMetadata{
		ID:     1,
		Artist: "Test Artist",
		Title:  "Test Title",
	}

	model := NewModel(track)

	// Тестируем обновление размера окна
	msg := tea.WindowSizeMsg{Width: 100, Height: 40}
	updatedModel, _ := model.Update(msg)

	// Приводим к нужному типу
	playerModel := updatedModel.(*Model)

	if playerModel.width != 100 {
		t.Errorf("Expected width 100, got %d", playerModel.width)
	}

	if playerModel.height != 40 {
		t.Errorf("Expected height 40, got %d", playerModel.height)
	}
}

func TestKeyHandling(t *testing.T) {
	track := data.TrackMetadata{
		ID:     1,
		Artist: "Test Artist",
		Title:  "Test Title",
	}

	model := NewModel(track)

	// Тестируем нажатие 'q' - должно вернуть команду для GoBackMsg
	// Создаем KeyMsg для 'q'
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}

	// Проверяем, что модель корректно обрабатывает ключи
	_, cmd := model.Update(keyMsg)

	if cmd == nil {
		t.Error("Expected command to be returned for 'q' key")
	}
}
