package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all tracks from the library",
	Long:  `Display a list of all tracks stored in the application data.`,
	Run: func(cmd *cobra.Command, args []string) {
		listTracks()
	},
}

func listTracks() {
	if len(appData.Tracks) == 0 {
		fmt.Println("📚 Библиотека пуста. Добавьте треки с помощью команды 'add'.")
		return
	}

	fmt.Printf("📚 Найдено треков: %d\n\n", len(appData.Tracks))

	// Выводим заголовок таблицы
	fmt.Printf("%-4s %-30s %-30s %-20s %-10s %-12s\n",
		"ID", "Исполнитель", "Название", "Альбом", "Длительность", "Размер")
	fmt.Println(strings.Repeat("-", 120))

	// Выводим каждый трек
	for _, track := range appData.Tracks {
		// Форматируем длительность
		duration := formatDurationFromSeconds(track.Length)
		if track.Length == 0 {
			duration = "N/A"
		}

		// Форматируем размер файла
		fileSize := formatFileSize(track.FileSize)

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
	return s[:maxLen-3] + "..."
}

// Функция для форматирования длительности из секунд
func formatDurationFromSeconds(seconds int) string {
	if seconds == 0 {
		return "N/A"
	}

	minutes := seconds / 60
	remainingSeconds := seconds % 60
	return fmt.Sprintf("%02d:%02d", minutes, remainingSeconds)
}
