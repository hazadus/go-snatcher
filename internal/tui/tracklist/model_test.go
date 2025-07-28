package tracklist

import (
	"testing"

	"github.com/hazadus/go-snatcher/internal/data"
)

func TestNewModel(t *testing.T) {
	// Создаем тестовые данные
	appData := &data.AppData{
		Tracks: []data.TrackMetadata{
			{
				ID:     1,
				Artist: "Test Artist 1",
				Title:  "Test Track 1",
				Album:  "Test Album 1",
				Length: 180,
			},
			{
				ID:     2,
				Artist: "Test Artist 2",
				Title:  "Test Track 2",
				Album:  "Test Album 2",
				Length: 240,
			},
		},
	}

	// Создаем модель
	model := NewModel(appData)

	// Проверяем, что модель создалась корректно
	if model == nil {
		t.Fatal("NewModel returned nil")
	}

	if model.trackManager == nil {
		t.Fatal("trackManager is nil")
	}

	if model.list.Items() == nil {
		t.Fatal("list items is nil")
	}

	// Проверяем количество элементов в списке
	if len(model.list.Items()) != 2 {
		t.Fatalf("Expected 2 items, got %d", len(model.list.Items()))
	}
}
