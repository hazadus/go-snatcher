package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/dhowden/tag"
	"github.com/gopxl/beep"
	"github.com/gopxl/beep/mp3"
	"github.com/gopxl/beep/speaker"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "snatcher [mp3 file]",
	Short: "Play an mp3 file",
	Long:  `A simple command line tool to play mp3 files.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		play(args[0])
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func main() {
	Execute()
}

func play(filepath string) {
	// Читаем метаданные MP3 файла
	metadata := getMetadata(filepath)
	
	// Открываем файл для воспроизведения
	f, err := os.Open(filepath)
	if err != nil {
		log.Fatal(err)
	}

	streamer, format, err := mp3.Decode(f)
	if err != nil {
		log.Fatal(err)
	}
	defer streamer.Close()

	err = speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))
	if err != nil {
		log.Fatal(err)
	}

	// Выводим информацию о треке
	fmt.Printf("🎵 Сейчас играет:\n")
	fmt.Printf("   Исполнитель: %s\n", metadata.Artist)
	fmt.Printf("   Название: %s\n", metadata.Title)
	fmt.Printf("   Альбом: %s\n", metadata.Album)
	fmt.Printf("   Продолжительность: %s\n", formatDuration(format.SampleRate.D(streamer.Len())))
	fmt.Println()

	// Создаем канал для сигнала завершения
	done := make(chan bool)
	
	// Запускаем воспроизведение с callback для завершения
	speaker.Play(beep.Seq(streamer, beep.Callback(func() {
		done <- true
	})))

	// Запускаем горутину для отображения прогресса
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				speaker.Lock()
				currentPos := format.SampleRate.D(streamer.Position())
				totalLen := format.SampleRate.D(streamer.Len())
				speaker.Unlock()
				
				// Очищаем строку и выводим прогресс
				fmt.Printf("\r⏱️  Прогресс: %s / %s", 
					formatDuration(currentPos), 
					formatDuration(totalLen))
			}
		}
	}()

	// Ждем завершения воспроизведения
	<-done
	fmt.Println("\n✅ Воспроизведение завершено")
}

// Структура для хранения метаданных
type TrackMetadata struct {
	Artist string
	Title  string
	Album  string
}

// Функция для получения метаданных из MP3 файла
func getMetadata(filepath string) TrackMetadata {
	file, err := os.Open(filepath)
	if err != nil {
		return TrackMetadata{
			Artist: "Неизвестный исполнитель",
			Title:  "Неизвестный трек",
			Album:  "Неизвестный альбом",
		}
	}
	defer file.Close()

	metadata, err := tag.ReadFrom(file)
	if err != nil {
		return TrackMetadata{
			Artist: "Неизвестный исполнитель",
			Title:  "Неизвестный трек",
			Album:  "Неизвестный альбом",
		}
	}

	artist := metadata.Artist()
	title := metadata.Title()
	album := metadata.Album()

	// Если метаданные пустые, используем имя файла как название
	if title == "" {
		title = getFileNameWithoutExt(filepath)
	}

	return TrackMetadata{
		Artist: artist,
		Title:  title,
		Album:  album,
	}
}

// Функция для форматирования длительности
func formatDuration(d time.Duration) string {
	minutes := int(d.Minutes())
	seconds := int(d.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}

// Функция для получения имени файла без расширения
func getFileNameWithoutExt(filepath string) string {
	parts := strings.Split(filepath, "/")
	filename := parts[len(parts)-1]
	return strings.TrimSuffix(filename, ".mp3")
}