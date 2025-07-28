// Package tracklist содержит модель экрана списка треков для TUI
package tracklist

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hazadus/go-snatcher/internal/data"
	"github.com/hazadus/go-snatcher/internal/track"
	"github.com/hazadus/go-snatcher/internal/utils"
)

var (
	titleStyle        = lipgloss.NewStyle().MarginLeft(2)
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
	paginationStyle   = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	helpStyle         = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
	quitTextStyle     = lipgloss.NewStyle().Margin(1, 0, 2, 4)
)

// TrackSelectedMsg отправляется при выборе трека для воспроизведения
type TrackSelectedMsg struct {
	Track data.TrackMetadata
}

// TrackEditMsg отправляется при выборе трека для редактирования
type TrackEditMsg struct {
	Track data.TrackMetadata
}

// trackItem реализует интерфейс list.Item для трека
type trackItem struct {
	track data.TrackMetadata
}

func (i trackItem) FilterValue() string {
	return fmt.Sprintf("%s %s", i.track.Artist, i.track.Title)
}

// trackItemDelegate реализует отображение элементов списка
type trackItemDelegate struct{}

func (d trackItemDelegate) Height() int                             { return 1 }
func (d trackItemDelegate) Spacing() int                            { return 0 }
func (d trackItemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d trackItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(trackItem)
	if !ok {
		return
	}

	// Форматируем строку в виде таблицы: ID | Исполнитель | Название | Продолжительность
	duration := utils.FormatDurationFromSeconds(i.track.Length)
	str := fmt.Sprintf("%-4d %-20s %-50s %s",
		i.track.ID,
		utils.TruncateString(i.track.Artist, 20),
		utils.TruncateString(i.track.Title, 50),
		duration)

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render("> " + strings.Join(s, " "))
		}
	}

	fmt.Fprint(w, fn(str))
}

// Model представляет модель экрана списка треков
type Model struct {
	list         list.Model
	trackManager *track.Manager
	quitting     bool
}

// NewModel создает новую модель списка треков
func NewModel(appData *data.AppData) *Model {
	trackManager := track.NewManager(appData)
	tracks := trackManager.ListTracks()

	// Преобразуем треки в элементы списка
	items := make([]list.Item, len(tracks))
	for i, t := range tracks {
		items[i] = trackItem{track: t}
	}

	// Создаем список
	l := list.New(items, trackItemDelegate{}, 0, 0)
	l.Title = "Треки"
	l.SetShowStatusBar(false)
	l.SetShowTitle(true) // Убеждаемся, что заголовок отображается
	l.SetFilteringEnabled(true)
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle

	return &Model{
		list:         l,
		trackManager: trackManager,
	}
}

// Init инициализирует модель
func (m *Model) Init() tea.Cmd {
	return nil
}

// RefreshData обновляет данные модели без пересоздания
func (m *Model) RefreshData() {
	// Получаем актуальные треки
	tracks := m.trackManager.ListTracks()

	// Преобразуем треки в элементы списка
	items := make([]list.Item, len(tracks))
	for i, t := range tracks {
		items[i] = trackItem{track: t}
	}

	// Обновляем элементы в существующем списке
	m.list.SetItems(items)
}

// Update обрабатывает сообщения и обновляет модель
func (m *Model) Update(msg tea.Msg) (*Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		m.list.SetHeight(msg.Height - 4) // Оставляем место для заголовка и справки
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit

		case "enter":
			// Получаем выбранный элемент
			selectedItem := m.list.SelectedItem()
			if selectedItem != nil {
				if item, ok := selectedItem.(trackItem); ok {
					// Отправляем сообщение о выборе трека
					return m, func() tea.Msg {
						return TrackSelectedMsg{Track: item.track}
					}
				}
			}

		case "e":
			// Редактирование выбранного трека
			selectedItem := m.list.SelectedItem()
			if selectedItem != nil {
				if item, ok := selectedItem.(trackItem); ok {
					// Отправляем сообщение о редактировании трека
					return m, func() tea.Msg {
						return TrackEditMsg{Track: item.track}
					}
				}
			}
		}
	}

	// Обновляем список
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// View отображает модель
func (m *Model) View() string {
	if m.quitting {
		return quitTextStyle.Render("До свидания!")
	}

	view := m.list.View()
	// Добавляем дополнительную справку
	extraHelp := helpStyle.Render("Enter: воспроизвести • e: редактировать • q: выход")
	return view + "\n" + extraHelp
}
