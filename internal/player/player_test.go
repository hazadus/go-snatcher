package player

import (
	"testing"
	"time"

	"github.com/hazadus/go-snatcher/internal/data"
)

func TestPlay(t *testing.T) {
	player := NewPlayer()
	defer player.Close()

	// Создаем тестовый трек
	track := &data.TrackMetadata{
		ID:        1,
		Artist:    "Test Artist",
		Title:     "Test Title",
		Length:    180,
		URL:       "https://example.com/test.mp3",
		SourceURL: "https://example.com/source",
	}

	// Пытаемся воспроизвести трек
	err := player.Play(track)

	// Ожидаем ошибку, так как URL не является валидным аудиофайлом
	if err == nil {
		t.Error("Ожидалась ошибка при воспроизведении невалидного URL")
	}

	// Проверяем, что трек установлен как текущий
	currentTrack := player.CurrentTrack()
	if currentTrack == nil {
		t.Error("Трек должен быть установлен как текущий")
	} else if currentTrack.ID != track.ID {
		t.Errorf("Ожидался трек с ID %d, получен трек с ID %d", track.ID, currentTrack.ID)
	}
}

func TestPauseResume(t *testing.T) {
	player := NewPlayer()
	defer player.Close()

	// Создаем тестовый трек
	track := &data.TrackMetadata{
		ID:     1,
		Artist: "Test Artist",
		Title:  "Test Title",
		Length: 180,
		URL:    "https://example.com/test.mp3",
	}

	// Пытаемся воспроизвести трек (ожидаем ошибку)
	_ = player.Play(track)

	// Проверяем начальное состояние
	if player.IsPlaying() {
		t.Error("Плеер не должен воспроизводить при ошибке загрузки")
	}

	// Тестируем паузу (должна работать даже без активного воспроизведения)
	player.Pause()

	// Проверяем, что состояние не изменилось
	if player.IsPlaying() {
		t.Error("Плеер не должен воспроизводить после паузы")
	}

	// Тестируем возобновление
	player.Pause()

	// Проверяем, что состояние не изменилось
	if player.IsPlaying() {
		t.Error("Плеер не должен воспроизводить после возобновления")
	}
}

func TestStop(t *testing.T) {
	player := NewPlayer()
	defer player.Close()

	// Создаем тестовый трек
	track := &data.TrackMetadata{
		ID:     1,
		Artist: "Test Artist",
		Title:  "Test Title",
		Length: 180,
		URL:    "https://example.com/test.mp3",
	}

	// Пытаемся воспроизвести трек
	_ = player.Play(track)

	// Останавливаем воспроизведение
	player.Stop()

	// Проверяем, что воспроизведение остановлено
	if player.IsPlaying() {
		t.Error("Плеер не должен воспроизводить после остановки")
	}

	// Проверяем, что текущий трек очищен
	currentTrack := player.CurrentTrack()
	if currentTrack != nil {
		t.Error("Текущий трек должен быть очищен после остановки")
	}
}

func TestPlayNonExistentFile(t *testing.T) {
	player := NewPlayer()
	defer player.Close()

	// Создаем трек с несуществующим URL
	track := &data.TrackMetadata{
		ID:        1,
		Artist:    "Test Artist",
		Title:     "Test Title",
		Length:    180,
		URL:       "https://non-existent-domain.com/test.mp3",
		SourceURL: "https://example.com/source",
	}

	// Пытаемся воспроизвести трек
	err := player.Play(track)

	// Ожидаем ошибку
	if err == nil {
		t.Error("Ожидалась ошибка при воспроизведении несуществующего файла")
	}

	// Проверяем сообщение об ошибке
	if err != nil {
		errorMsg := err.Error()
		if !contains(errorMsg, "ошибка создания потокового ридера") &&
			!contains(errorMsg, "ошибка декодирования MP3") {
			t.Errorf("Неожиданное сообщение об ошибке: %v", err)
		}
	}
}

func TestPlayerChannels(t *testing.T) {
	player := NewPlayer()
	defer player.Close()

	// Проверяем, что каналы созданы
	progressChan := player.Progress()
	if progressChan == nil {
		t.Error("Канал прогресса не должен быть nil")
	}

	doneChan := player.Done()
	if doneChan == nil {
		t.Error("Канал завершения не должен быть nil")
	}

	// Проверяем, что каналы не закрыты изначально
	select {
	case <-progressChan:
		t.Error("Канал прогресса не должен быть закрыт изначально")
	default:
		// Ожидаемое поведение
	}

	select {
	case <-doneChan:
		t.Error("Канал завершения не должен быть закрыт изначально")
	default:
		// Ожидаемое поведение
	}
}

func TestPlayerStateManagement(t *testing.T) {
	player := NewPlayer()
	defer player.Close()

	// Проверяем начальное состояние
	if player.IsPlaying() {
		t.Error("Плеер не должен воспроизводить в начальном состоянии")
	}

	if player.CurrentTrack() != nil {
		t.Error("Текущий трек должен быть nil в начальном состоянии")
	}

	// Создаем тестовый трек
	track := &data.TrackMetadata{
		ID:     1,
		Artist: "Test Artist",
		Title:  "Test Title",
		Length: 180,
		URL:    "https://example.com/test.mp3",
	}

	// Пытаемся воспроизвести трек
	_ = player.Play(track)

	// Проверяем, что трек установлен
	currentTrack := player.CurrentTrack()
	if currentTrack == nil {
		t.Error("Трек должен быть установлен после Play")
	} else if currentTrack.ID != track.ID {
		t.Errorf("Ожидался трек с ID %d, получен трек с ID %d", track.ID, currentTrack.ID)
	}

	// Останавливаем воспроизведение
	player.Stop()

	// Проверяем, что состояние сброшено
	if player.IsPlaying() {
		t.Error("Плеер не должен воспроизводить после остановки")
	}

	if player.CurrentTrack() != nil {
		t.Error("Текущий трек должен быть nil после остановки")
	}
}

func TestPlayerConcurrentAccess(t *testing.T) {
	player := NewPlayer()
	defer player.Close()

	// Создаем тестовый трек
	track := &data.TrackMetadata{
		ID:     1,
		Artist: "Test Artist",
		Title:  "Test Title",
		Length: 180,
		URL:    "https://example.com/test.mp3",
	}

	// Запускаем несколько горутин для тестирования конкурентного доступа
	done := make(chan bool, 3)

	// Горутина 1: воспроизведение
	go func() {
		_ = player.Play(track)
		done <- true
	}()

	// Горутина 2: пауза
	go func() {
		time.Sleep(10 * time.Millisecond)
		player.Pause()
		done <- true
	}()

	// Горутина 3: остановка
	go func() {
		time.Sleep(20 * time.Millisecond)
		player.Stop()
		done <- true
	}()

	// Ждем завершения всех горутин
	for i := 0; i < 3; i++ {
		select {
		case <-done:
			// Успешно
		case <-time.After(1 * time.Second):
			t.Error("Таймаут при тестировании конкурентного доступа")
		}
	}

	// Проверяем финальное состояние
	if player.IsPlaying() {
		t.Error("Плеер не должен воспроизводить после конкурентных операций")
	}
}

// Вспомогательная функция для проверки содержимого строки
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) && (s[:len(substr)] == substr ||
			s[len(s)-len(substr):] == substr ||
			containsSubstring(s, substr))))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
