package track

import (
	"testing"

	"github.com/hazadus/go-snatcher/internal/data"
)

func TestAddTrack(t *testing.T) {
	// Создаем новый AppData
	appData := data.NewAppData()
	manager := NewManager(appData)

	// Создаем тестовый трек
	track := data.TrackMetadata{
		Artist:    "Test Artist",
		Title:     "Test Title",
		Album:     "Test Album",
		Length:    180,
		FileSize:  1024000,
		URL:       "https://s3.example.com/test.mp3",
		SourceURL: "https://example.com/source",
	}

	// Добавляем трек через AppData
	appData.AddTrack(track)

	// Проверяем, что трек был добавлен
	tracks := manager.ListTracks()
	if len(tracks) != 1 {
		t.Errorf("Ожидался 1 трек, получено %d", len(tracks))
	}

	// Проверяем, что данные трека корректны
	addedTrack := tracks[0]
	if addedTrack.Artist != track.Artist {
		t.Errorf("Ожидался Artist: %s, получено: %s", track.Artist, addedTrack.Artist)
	}
	if addedTrack.Title != track.Title {
		t.Errorf("Ожидался Title: %s, получено: %s", track.Title, addedTrack.Title)
	}
	if addedTrack.ID != 1 {
		t.Errorf("Ожидался ID: 1, получено: %d", addedTrack.ID)
	}
}

func TestDeleteTrack(t *testing.T) {
	// Создаем новый AppData
	appData := data.NewAppData()
	manager := NewManager(appData)

	// Добавляем несколько треков
	track1 := data.TrackMetadata{
		Artist: "Artist 1",
		Title:  "Title 1",
	}
	track2 := data.TrackMetadata{
		Artist: "Artist 2",
		Title:  "Title 2",
	}

	appData.AddTrack(track1)
	appData.AddTrack(track2)

	// Проверяем, что треки добавлены
	if len(manager.ListTracks()) != 2 {
		t.Errorf("Ожидалось 2 трека, получено %d", len(manager.ListTracks()))
	}

	// Удаляем первый трек (ID = 1)
	err := appData.DeleteTrackByID(1)
	if err != nil {
		t.Errorf("Ошибка при удалении трека: %v", err)
	}

	// Проверяем, что трек был удален
	tracks := manager.ListTracks()
	if len(tracks) != 1 {
		t.Errorf("Ожидался 1 трек после удаления, получено %d", len(tracks))
	}

	// Проверяем, что оставшийся трек имеет правильный ID
	if tracks[0].ID != 2 {
		t.Errorf("Ожидался ID: 2, получено: %d", tracks[0].ID)
	}
}

func TestGetAllTracks(t *testing.T) {
	// Создаем новый AppData
	appData := data.NewAppData()
	manager := NewManager(appData)

	// Проверяем, что изначально список пуст
	tracks := manager.ListTracks()
	if len(tracks) != 0 {
		t.Errorf("Ожидался пустой список треков, получено %d", len(tracks))
	}

	// Добавляем несколько треков
	track1 := data.TrackMetadata{
		Artist: "Artist 1",
		Title:  "Title 1",
	}
	track2 := data.TrackMetadata{
		Artist: "Artist 2",
		Title:  "Title 2",
	}
	track3 := data.TrackMetadata{
		Artist: "Artist 3",
		Title:  "Title 3",
	}

	appData.AddTrack(track1)
	appData.AddTrack(track2)
	appData.AddTrack(track3)

	// Проверяем, что все треки получены
	tracks = manager.ListTracks()
	if len(tracks) != 3 {
		t.Errorf("Ожидалось 3 трека, получено %d", len(tracks))
	}

	// Проверяем, что ID присваиваются последовательно
	for i, track := range tracks {
		expectedID := i + 1
		if track.ID != expectedID {
			t.Errorf("Трек %d: ожидался ID %d, получено %d", i+1, expectedID, track.ID)
		}
	}
}

func TestFindTrack(t *testing.T) {
	// Создаем новый AppData
	appData := data.NewAppData()

	// Добавляем тестовые треки
	track1 := data.TrackMetadata{
		Artist: "The Beatles",
		Title:  "Hey Jude",
		Album:  "The Beatles 1967-1970",
	}
	track2 := data.TrackMetadata{
		Artist: "Queen",
		Title:  "Bohemian Rhapsody",
		Album:  "A Night at the Opera",
	}
	track3 := data.TrackMetadata{
		Artist: "The Beatles",
		Title:  "Let It Be",
		Album:  "Let It Be",
	}

	appData.AddTrack(track1)
	appData.AddTrack(track2)
	appData.AddTrack(track3)

	// Тестируем поиск по ID
	foundTrack, err := appData.TrackByID(2)
	if err != nil {
		t.Errorf("Ошибка при поиске трека по ID: %v", err)
	}
	if foundTrack.Artist != "Queen" {
		t.Errorf("Ожидался Artist: Queen, получено: %s", foundTrack.Artist)
	}

	// Тестируем поиск несуществующего трека
	_, err = appData.TrackByID(999)
	if err == nil {
		t.Error("Ожидалась ошибка при поиске несуществующего трека")
	}
}

func TestAddDuplicateTrack(t *testing.T) {
	// Создаем новый AppData
	appData := data.NewAppData()

	// Добавляем первый трек
	track := data.TrackMetadata{
		Artist: "Test Artist",
		Title:  "Test Title",
		URL:    "https://s3.example.com/test.mp3",
	}

	appData.AddTrack(track)

	// Добавляем тот же трек снова (дубликат)
	appData.AddTrack(track)

	// Проверяем, что оба трека добавлены (система позволяет дубликаты)
	tracks := appData.Tracks
	if len(tracks) != 2 {
		t.Errorf("Ожидалось 2 трека после добавления дубликата, получено %d", len(tracks))
	}

	// Проверяем, что ID присваиваются корректно
	if tracks[0].ID != 1 {
		t.Errorf("Первый трек: ожидался ID 1, получено %d", tracks[0].ID)
	}
	if tracks[1].ID != 2 {
		t.Errorf("Второй трек: ожидался ID 2, получено %d", tracks[1].ID)
	}
}
