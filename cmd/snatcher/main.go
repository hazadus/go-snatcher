package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
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
	Use:   "snatcher [mp3 file or URL]",
	Short: "Play an mp3 file from local path or URL",
	Long:  `A simple command line tool to play mp3 files from local path or URL.`,
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

func play(source string) {
	var reader io.ReadCloser
	var err error
	var isURL bool

	// Определяем, является ли источник URL или локальным файлом
	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		isURL = true
		fmt.Printf("🌐 Загружаем файл по URL: %s\n", source)
		reader, err = downloadFromURL(source)
		if err != nil {
			log.Fatal(err)
		}
		defer reader.Close()
	} else {
		isURL = false
		reader, err = os.Open(source)
		if err != nil {
			log.Fatal(err)
		}
		defer reader.Close()
	}

	// Читаем метаданные MP3 файла
	metadata := getMetadataFromReader(reader, source)
	
	// Сбрасываем позицию в reader для декодирования
	if seeker, ok := reader.(io.ReadSeeker); ok {
		seeker.Seek(0, 0)
	} else {
		// Если reader не поддерживает seek, создаем новый reader
		if isURL {
			reader.Close()
			reader, err = downloadFromURL(source)
			if err != nil {
				log.Fatal(err)
			}
			defer reader.Close()
		} else {
			reader.Close()
			reader, err = os.Open(source)
			if err != nil {
				log.Fatal(err)
			}
			defer reader.Close()
		}
	}

	streamer, format, err := mp3.Decode(reader)
	if err != nil {
		log.Fatal(err)
	}
	defer streamer.Close()

	err = speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))
	if err != nil {
		log.Fatal(err)
	}

	// Получаем длительность трека
	duration := format.SampleRate.D(streamer.Len())
	
	// Выводим информацию о треке
	fmt.Printf("🎵 Сейчас играет:\n")
	fmt.Printf("   Исполнитель: %s\n", metadata.Artist)
	fmt.Printf("   Название: %s\n", metadata.Title)
	fmt.Printf("   Альбом: %s\n", metadata.Album)
	fmt.Printf("   Продолжительность: %s\n", formatDuration(duration))
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
				
				// Проверяем, что длительность корректная
				if totalLen > 0 {
					// Очищаем строку и выводим прогресс
					fmt.Printf("\r⏱️  Прогресс: %s / %s", 
						formatDuration(currentPos), 
						formatDuration(totalLen))
				} else {
					// Если длительность не определена, показываем только текущую позицию
					fmt.Printf("\r⏱️  Воспроизведение: %s", 
						formatDuration(currentPos))
				}
			}
		}
	}()

	// Ждем завершения воспроизведения
	<-done
	fmt.Println("\n✅ Воспроизведение завершено")
}

// Функция для загрузки файла по URL
func downloadFromURL(url string) (io.ReadCloser, error) {
	client := &http.Client{
		Timeout: 60 * time.Second, // Увеличиваем таймаут для больших файлов
	}
	
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("ошибка при загрузке файла: %v", err)
	}
	
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("HTTP ошибка: %s", resp.Status)
	}
	
	// Проверяем Content-Type
	contentType := resp.Header.Get("Content-Type")
	if contentType != "" && !strings.Contains(contentType, "audio/") && !strings.Contains(contentType, "application/octet-stream") {
		fmt.Printf("⚠️  Предупреждение: неожиданный Content-Type: %s\n", contentType)
	}
	
	return resp.Body, nil
}

// Структура для хранения метаданных
type TrackMetadata struct {
	Artist string
	Title  string
	Album  string
}

// Функция для получения метаданных из reader
func getMetadataFromReader(reader io.ReadCloser, source string) TrackMetadata {
	// Создаем временный файл для чтения метаданных
	tempFile, err := os.CreateTemp("", "snatcher-*.mp3")
	if err != nil {
		return getDefaultMetadata(source)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	// Копируем данные в временный файл
	_, err = io.Copy(tempFile, reader)
	if err != nil {
		return getDefaultMetadata(source)
	}

	// Сбрасываем позицию в файле
	tempFile.Seek(0, 0)

	metadata, err := tag.ReadFrom(tempFile)
	if err != nil {
		// Если не удалось прочитать метаданные, возвращаем значения по умолчанию
		return getDefaultMetadata(source)
	}

	artist := metadata.Artist()
	title := metadata.Title()
	album := metadata.Album()

	// Если метаданные пустые, используем имя файла или URL как название
	if title == "" {
		title = getFileNameFromSource(source)
	}

	return TrackMetadata{
		Artist: artist,
		Title:  title,
		Album:  album,
	}
}


// Функция для получения имени файла из источника (локальный файл или URL)
func getFileNameFromSource(source string) string {
	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		// Для URL извлекаем имя файла из пути
		parts := strings.Split(source, "/")
		filename := parts[len(parts)-1]
		// Убираем параметры запроса
		if idx := strings.Index(filename, "?"); idx != -1 {
			filename = filename[:idx]
		}
		// Если имя файла пустое или это корневой путь, используем домен
		if filename == "" || filename == "/" {
			// Извлекаем домен из URL
			urlParts := strings.Split(source, "/")
			if len(urlParts) >= 3 {
				filename = urlParts[2] // domain
			} else {
				filename = "online_track"
			}
		}
		return strings.TrimSuffix(filename, ".mp3")
	} else {
		// Для локального файла
		return getFileNameWithoutExt(source)
	}
}

// Функция для получения метаданных по умолчанию
func getDefaultMetadata(source string) TrackMetadata {
	return TrackMetadata{
		Artist: "Неизвестный исполнитель",
		Title:  getFileNameFromSource(source),
		Album:  "Неизвестный альбом",
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