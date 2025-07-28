// Package editor содержит модель экрана редактирования метаданных трека для TUI
package editor

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hazadus/go-snatcher/internal/data"
	"github.com/hazadus/go-snatcher/internal/track"
)

var (
	titleStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true).Margin(1, 0)
	labelStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Width(15)
	focusedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	blurredStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	helpStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Margin(1, 0)
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Margin(1, 0)
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Margin(1, 0)
	footerStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Margin(1, 0)
)

// TrackSavedMsg отправляется когда трек успешно сохранен
type TrackSavedMsg struct{}

// GoBackMsg отправляется при отмене редактирования
type GoBackMsg struct{}

// fieldType определяет тип поля для редактирования
type fieldType int

const (
	artistField fieldType = iota
	titleField
	albumField
	lengthField
	sourceURLField
	numFields
)

// Model представляет модель экрана редактирования трека
type Model struct {
	trackManager  *track.Manager
	originalTrack data.TrackMetadata
	inputs        []textinput.Model
	focusIndex    int
	err           string
	success       string
	quitting      bool
	saveFunc      func() error // Функция для сохранения данных в файл
}

// NewModel создает новую модель редактора трека
func NewModel(appData *data.AppData, trackToEdit data.TrackMetadata, saveFunc func() error) *Model {
	trackManager := track.NewManager(appData)

	// Создаем поля ввода
	inputs := make([]textinput.Model, numFields)

	// Поле Artist
	inputs[artistField] = textinput.New()
	inputs[artistField].Placeholder = "Введите исполнителя"
	inputs[artistField].SetValue(trackToEdit.Artist)
	inputs[artistField].Focus()
	inputs[artistField].PromptStyle = focusedStyle
	inputs[artistField].TextStyle = focusedStyle

	// Поле Title
	inputs[titleField] = textinput.New()
	inputs[titleField].Placeholder = "Введите название трека"
	inputs[titleField].SetValue(trackToEdit.Title)

	// Поле Album
	inputs[albumField] = textinput.New()
	inputs[albumField].Placeholder = "Введите название альбома"
	inputs[albumField].SetValue(trackToEdit.Album)

	// Поле Length (в секундах)
	inputs[lengthField] = textinput.New()
	inputs[lengthField].Placeholder = "Длительность в секундах"
	inputs[lengthField].SetValue(strconv.Itoa(trackToEdit.Length))

	// Поле Source URL
	inputs[sourceURLField] = textinput.New()
	inputs[sourceURLField].Placeholder = "URL источника"
	inputs[sourceURLField].SetValue(trackToEdit.SourceURL)

	return &Model{
		trackManager:  trackManager,
		originalTrack: trackToEdit,
		inputs:        inputs,
		focusIndex:    0,
		saveFunc:      saveFunc,
	}
}

// Init инициализирует модель
func (m *Model) Init() tea.Cmd {
	return textinput.Blink
}

// Update обрабатывает сообщения и обновляет модель
func (m *Model) Update(msg tea.Msg) (*Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			// Отменяем редактирование
			return m, func() tea.Msg {
				return GoBackMsg{}
			}

		case "ctrl+s":
			// Сохраняем изменения
			return m, m.saveTrack()

		case "tab", "shift+tab", "enter", "up", "down":
			s := msg.String()

			// Обработка навигации между полями
			if s == "enter" && m.focusIndex == len(m.inputs) {
				// Enter на кнопке Save
				return m, m.saveTrack()
			}

			// Перемещение фокуса
			if s == "up" || s == "shift+tab" {
				m.focusIndex--
			} else {
				m.focusIndex++
			}

			if m.focusIndex > len(m.inputs) {
				m.focusIndex = 0
			} else if m.focusIndex < 0 {
				m.focusIndex = len(m.inputs)
			}

			cmds := make([]tea.Cmd, len(m.inputs))
			for i := 0; i < len(m.inputs); i++ {
				if i == m.focusIndex {
					// Устанавливаем фокус на текущее поле
					cmds[i] = m.inputs[i].Focus()
					m.inputs[i].PromptStyle = focusedStyle
					m.inputs[i].TextStyle = focusedStyle
				} else {
					// Убираем фокус с остальных полей
					m.inputs[i].Blur()
					m.inputs[i].PromptStyle = blurredStyle
					m.inputs[i].TextStyle = blurredStyle
				}
			}

			return m, tea.Batch(cmds...)
		}

	case tea.WindowSizeMsg:
		// Обновляем ширину полей ввода
		for i := range m.inputs {
			m.inputs[i].Width = msg.Width - 20
		}
		return m, nil
	}

	// Обновляем активное поле ввода
	if m.focusIndex < len(m.inputs) {
		var cmd tea.Cmd
		m.inputs[m.focusIndex], cmd = m.inputs[m.focusIndex].Update(msg)
		return m, cmd
	}

	return m, nil
}

// saveTrack сохраняет изменения трека
func (m *Model) saveTrack() tea.Cmd {
	return func() tea.Msg {
		// Валидируем и парсим поля
		artist := strings.TrimSpace(m.inputs[artistField].Value())
		title := strings.TrimSpace(m.inputs[titleField].Value())
		album := strings.TrimSpace(m.inputs[albumField].Value())
		lengthStr := strings.TrimSpace(m.inputs[lengthField].Value())
		sourceURL := strings.TrimSpace(m.inputs[sourceURLField].Value())

		// Проверяем обязательные поля
		if artist == "" {
			m.err = "Поле 'Исполнитель' не может быть пустым"
			m.success = ""
			return nil
		}

		if title == "" {
			m.err = "Поле 'Название' не может быть пустым"
			m.success = ""
			return nil
		}

		// Парсим длительность
		length, err := strconv.Atoi(lengthStr)
		if err != nil || length < 0 {
			m.err = "Длительность должна быть положительным числом"
			m.success = ""
			return nil
		}

		// Создаем обновленный трек
		updatedTrack := m.originalTrack
		updatedTrack.Artist = artist
		updatedTrack.Title = title
		updatedTrack.Album = album
		updatedTrack.Length = length
		updatedTrack.SourceURL = sourceURL

		// Сохраняем изменения в памяти
		err = m.trackManager.UpdateTrack(updatedTrack)
		if err != nil {
			m.err = fmt.Sprintf("Ошибка обновления трека: %v", err)
			m.success = ""
			return nil
		}

		// Сохраняем данные в файл
		if m.saveFunc != nil {
			err = m.saveFunc()
			if err != nil {
				m.err = fmt.Sprintf("Ошибка сохранения в файл: %v", err)
				m.success = ""
				return nil
			}
		}

		m.err = ""
		m.success = "Трек успешно сохранен!"

		// Возвращаемся к списку треков через небольшую задержку
		return tea.Tick(time.Second, func(time.Time) tea.Msg {
			return GoBackMsg{}
		})()
	}
}

// View отображает модель
func (m *Model) View() string {
	if m.quitting {
		return "Отмена редактирования...\n"
	}

	var b strings.Builder

	// Заголовок
	b.WriteString(titleStyle.Render(fmt.Sprintf("Редактирование трека #%d", m.originalTrack.ID)))
	b.WriteString("\n\n")

	// Поля ввода
	labels := []string{"Исполнитель:", "Название:", "Альбом:", "Длительность:", "URL источника:"}

	for i, input := range m.inputs {
		b.WriteString(labelStyle.Render(labels[i]))
		b.WriteString(" ")
		b.WriteString(input.View())
		b.WriteString("\n\n")
	}

	// Кнопка сохранения
	saveButton := "[ Сохранить ]"
	if m.focusIndex == len(m.inputs) {
		saveButton = focusedStyle.Render("[ Сохранить ]")
	} else {
		saveButton = blurredStyle.Render(saveButton)
	}
	b.WriteString(saveButton)
	b.WriteString("\n\n")

	// Сообщения об ошибках или успехе
	if m.err != "" {
		b.WriteString(errorStyle.Render(m.err))
		b.WriteString("\n")
	}

	if m.success != "" {
		b.WriteString(successStyle.Render(m.success))
		b.WriteString("\n")
	}

	// Справка
	b.WriteString(helpStyle.Render("Tab/Enter: следующее поле • Shift+Tab: предыдущее поле"))
	b.WriteString("\n")
	b.WriteString(footerStyle.Render("Ctrl+S: сохранить • Esc: отмена"))

	return b.String()
}
