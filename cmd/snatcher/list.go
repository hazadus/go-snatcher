package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/hazadus/go-snatcher/internal/track"
	"github.com/hazadus/go-snatcher/internal/uploader"
	"github.com/hazadus/go-snatcher/internal/utils"
)

// createListCommand создает команду list с привязкой к экземпляру приложения
func (app *Application) createListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all tracks from the library",
		Long:  `Display a list of all tracks stored in the application data.`,
		Run: func(_ *cobra.Command, _ []string) {
			app.listTracks()
		},
	}
}

func (app *Application) listTracks() {
	// Создаем менеджер треков
	trackManager := track.NewManager(app.Data)
	tracks := trackManager.ListTracks()

	if len(tracks) == 0 {
		fmt.Println("📚 Библиотека пуста. Добавьте треки с помощью команды 'add'.")
		return
	}

	fmt.Printf("📚 Найдено треков: %d\n\n", len(tracks))

	// Выводим заголовок таблицы
	fmt.Printf("%-4s %-30s %-30s %-20s %-10s %-12s\n",
		"ID", "Исполнитель", "Название", "Альбом", "Длительность", "Размер")
	fmt.Println(strings.Repeat("-", 120))

	// Выводим каждый трек
	for _, track := range tracks {
		// Форматируем длительность
		duration := utils.FormatDurationFromSeconds(track.Length)
		if track.Length == 0 {
			duration = "N/A"
		}

		// Форматируем размер файла
		fileSize := uploader.FormatFileSize(track.FileSize)

		// Обрезаем длинные строки для красивого отображения
		artist := truncateString(track.Artist, 28)
		title := truncateString(track.Title, 28)
		album := truncateString(track.Album, 18)

		fmt.Printf("%-4d %-30s %-30s %-20s %-10s %-12s\n",
			track.ID, artist, title, album, duration, fileSize)
	}

	fmt.Println()
	fmt.Println("💡 Используйте 'snatcher play [ID]' для воспроизведения трека")
}

// Функция для обрезки строки до указанной длины
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
