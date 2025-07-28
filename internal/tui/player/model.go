// Package player содержит модель экрана воспроизведения для TUI
package player

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hazadus/go-snatcher/internal/data"
	"github.com/hazadus/go-snatcher/internal/player"
	"github.com/hazadus/go-snatcher/internal/utils"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#0000ff")).
			MarginBottom(1)

	trackInfoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			MarginBottom(1)

	statusStyle = lipgloss.NewStyle().
			Bold(true).
			MarginTop(1).
			MarginBottom(1)

	controlsStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666")).
			MarginTop(1)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ff0000")).
			Bold(true)
)

// GoBackMsg отправляется для возврата к списку треков
type GoBackMsg struct{}

// ProgressMsg содержит обновления прогресса воспроизведения
type ProgressMsg struct {
	Status player.Status
}

// PlaybackFinishedMsg отправляется при завершении воспроизведения
type PlaybackFinishedMsg struct{}

// PlaybackErrorMsg отправляется при ошибке воспроизведения
type PlaybackErrorMsg struct {
	Error error
}

// Model представляет модель экрана воспроизведения
type Model struct {
	track       data.TrackMetadata
	player      *player.Player
	progressBar progress.Model
	status      player.Status
	isPlaying   bool
	error       error
	width       int
	height      int
	appData     *data.AppData // Ссылка на данные приложения для сохранения позиции
	saveFunc    func() error  // Функция для сохранения данных
}

// NewModel создает новую модель плеера
func NewModel(track data.TrackMetadata, appData *data.AppData, saveFunc func() error) *Model {
	// Создаем прогресс-бар
	prog := progress.New(progress.WithDefaultGradient())
	prog.Width = 40

	return &Model{
		track:       track,
		player:      player.NewPlayer(),
		progressBar: prog,
		isPlaying:   false,
		appData:     appData,
		saveFunc:    saveFunc,
	}
}

// NewModelWithPlayer создает новую модель плеера с использованием существующего плеера
func NewModelWithPlayer(track data.TrackMetadata, existingPlayer *player.Player, appData *data.AppData, saveFunc func() error) *Model {
	// Создаем прогресс-бар
	prog := progress.New(progress.WithDefaultGradient())
	prog.Width = 40

	return &Model{
		track:       track,
		player:      existingPlayer,
		progressBar: prog,
		isPlaying:   false,
		appData:     appData,
		saveFunc:    saveFunc,
	}
}

// Init инициализирует модель и запускает воспроизведение
func (m *Model) Init() tea.Cmd {
	// Возвращаем команду для запуска воспроизведения
	return tea.Batch(
		m.startPlayback(),
		m.listenForProgress(),
	)
}

// saveCurrentPosition сохраняет текущую позицию воспроизведения
func (m *Model) saveCurrentPosition() {
	if m.appData != nil && m.saveFunc != nil && m.player != nil {
		currentPos := int(m.status.Current.Seconds())
		// Сохраняем позицию только если прошло больше 5 секунд воспроизведения
		// и не достигли конца трека (оставшееся время больше 10 секунд)
		remaining := m.status.Total - m.status.Current
		if currentPos > 5 && remaining.Seconds() > 10 {
			err := m.appData.UpdateTrackPlaybackPosition(m.track.ID, currentPos)
			if err == nil {
				// Игнорируем ошибку сохранения, так как это не критично для воспроизведения
				_ = m.saveFunc()
			}
		}
	}
}

// Update обрабатывает сообщения и обновляет модель
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Обновляем ширину прогресс-бара
		m.progressBar.Width = min(60, msg.Width-10)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			// Сохраняем позицию перед выходом
			m.saveCurrentPosition()
			// Останавливаем плеер и возвращаемся к списку треков
			m.player.Stop()
			return m, func() tea.Msg {
				return GoBackMsg{}
			}

		case " ":
			// Пауза/воспроизведение
			m.player.Pause()
			m.isPlaying = !m.isPlaying
			return m, nil
		}

	case ProgressMsg:
		// Обновляем статус и прогресс-бар
		m.status = msg.Status
		m.isPlaying = msg.Status.IsPlaying

		// Вычисляем прогресс в процентах
		var percent float64
		if msg.Status.Total > 0 {
			percent = float64(msg.Status.Current) / float64(msg.Status.Total)
		}

		// Обновляем прогресс-бар и возвращаем команду для продолжения прослушивания
		return m, tea.Batch(
			m.progressBar.SetPercent(percent),
			m.listenForProgress(),
		)

	case PlaybackFinishedMsg:
		// Воспроизведение завершено, возвращаемся к списку
		m.isPlaying = false
		return m, func() tea.Msg {
			return GoBackMsg{}
		}

	case PlaybackErrorMsg:
		// Ошибка воспроизведения
		m.error = msg.Error
		m.isPlaying = false
		return m, nil

	case progress.FrameMsg:
		// Обновляем прогресс-бар
		progressModel, cmd := m.progressBar.Update(msg)
		m.progressBar = progressModel.(progress.Model)
		return m, cmd
	}

	return m, nil
}

// View отображает модель
func (m *Model) View() string {
	if m.error != nil {
		return fmt.Sprintf(
			"%s\n\n%s\n\n%s",
			titleStyle.Render("❌ Ошибка воспроизведения"),
			errorStyle.Render(m.error.Error()),
			controlsStyle.Render("Нажмите 'q' или 'esc' для возврата"),
		)
	}

	// Заголовок
	title := titleStyle.Render("🎵 Воспроизведение")

	// Информация о треке
	trackInfoBuilder := fmt.Sprintf(
		"🎤 %s\n🎵 %s\n💿 %s",
		m.track.Artist,
		m.track.Title,
		m.track.Album,
	)

	// Добавляем информацию о сохраненной позиции, если есть
	// Примечание: в текущей версии воспроизведение начинается с начала для потоковых треков
	if m.track.PlaybackPosition > 0 {
		trackInfoBuilder += fmt.Sprintf(
			"\n📍 Сохраненная позиция: %s (воспроизведение с начала из-за ограничений потокового воспроизведения)",
			utils.FormatDuration(time.Duration(m.track.PlaybackPosition)*time.Second),
		)
	}

	trackInfo := trackInfoStyle.Render(trackInfoBuilder)

	// Статус воспроизведения
	var statusIcon string
	if m.isPlaying {
		statusIcon = "▶️"
	} else {
		statusIcon = "⏸️"
	}

	statusText := statusStyle.Render(fmt.Sprintf("%s %s", statusIcon, formatStatus(m.isPlaying)))

	// Прогресс-бар
	progressView := m.progressBar.View()

	// Время
	timeText := fmt.Sprintf(
		"%s / %s",
		utils.FormatDuration(m.status.Current),
		utils.FormatDuration(m.status.Total),
	)

	// Элементы управления
	controls := controlsStyle.Render(
		"Пробел: пауза/воспроизведение • q/esc/ctrl+c: назад к списку (с сохранением позиции)",
	)

	return fmt.Sprintf(
		"%s\n\n%s\n\n%s\n\n%s\n%s\n\n%s",
		title,
		trackInfo,
		statusText,
		progressView,
		timeText,
		controls,
	)
}

// Close очищает ресурсы модели
func (m *Model) Close() error {
	if m.player != nil {
		return m.player.Close()
	}
	return nil
}

// startPlayback запускает воспроизведение трека
func (m *Model) startPlayback() tea.Cmd {
	return func() tea.Msg {
		// Получаем актуальную информацию о треке из данных приложения
		var startPosition int
		if m.appData != nil {
			// Ищем актуальную позицию воспроизведения в данных приложения
			for _, track := range m.appData.Tracks {
				if track.ID == m.track.ID {
					startPosition = track.PlaybackPosition
					// Обновляем локальную копию трека с актуальной позицией
					m.track.PlaybackPosition = track.PlaybackPosition
					break
				}
			}
		} else {
			// Если нет доступа к данным приложения, используем позицию из трека
			startPosition = m.track.PlaybackPosition
		}

		var err error
		// Примечание: в текущей версии startPosition игнорируется плеером для потоковых треков
		// из-за ограничений библиотеки beep при работе с потоковыми данными
		if startPosition > 0 {
			err = m.player.PlayFromPosition(&m.track, startPosition)
		} else {
			err = m.player.Play(&m.track)
		}

		if err != nil {
			return PlaybackErrorMsg{Error: err}
		}
		m.isPlaying = true
		return nil
	}
}

// listenForProgress слушает обновления прогресса от плеера
func (m *Model) listenForProgress() tea.Cmd {
	return func() tea.Msg {
		select {
		case status, ok := <-m.player.Progress():
			if !ok {
				return PlaybackFinishedMsg{}
			}
			return ProgressMsg{Status: status}

		case _, ok := <-m.player.Done():
			if !ok {
				return PlaybackFinishedMsg{}
			}
			return PlaybackFinishedMsg{}
		}
	}
}

// Вспомогательные функции

func formatStatus(isPlaying bool) string {
	if isPlaying {
		return "Воспроизведение"
	}
	return "Пауза"
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
