package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/dhowden/tag"
	"github.com/gopxl/beep"
	"github.com/gopxl/beep/mp3"
	"github.com/gopxl/beep/speaker"
	"github.com/hazadus/go-snatcher/internal/config"
	"github.com/spf13/cobra"
)

const (
	defaultConfigPath = "~/.snatcher"
)

var (
	cfg    *config.Config
	addCmd = &cobra.Command{
		Use:   "add [file path]",
		Short: "Upload an mp3 file to S3 storage",
		Long:  `Upload an mp3 file to S3 storage with progress tracking.`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			uploadToS3(args[0])
		},
	}
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
	var err error

	// Загружаем конфигурацию
	if cfg, err = config.LoadConfig(defaultConfigPath); err != nil {
		log.Fatalf("Ошибка загрузки конфигурации: %v", err)
	}

	fmt.Println("Используется бакет:", cfg.AwsBucketName)

	// Добавляем команду add к корневой команде
	rootCmd.AddCommand(addCmd)

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

// Функция для загрузки файла в S3 с отображением прогресса
func uploadToS3(filePath string) {
	// Проверяем существование файла
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		log.Fatalf("Файл не найден: %s", filePath)
	}

	// Открываем файл
	file, err := os.Open(filePath)
	if err != nil {
		log.Fatalf("Ошибка открытия файла: %v", err)
	}
	defer file.Close()

	// Получаем размер файла
	fileInfo, err := file.Stat()
	if err != nil {
		log.Fatalf("Ошибка получения информации о файле: %v", err)
	}
	fileSize := fileInfo.Size()

	// Создаем AWS сессию
	awsConfig := &aws.Config{
		Region: aws.String(cfg.AwsRegion),
		Credentials: credentials.NewStaticCredentials(
			cfg.AwsAccessKey,
			cfg.AwsSecretKey,
			"",
		),
	}

	// Если указан endpoint, добавляем его
	if cfg.AwsEndpoint != "" {
		awsConfig.Endpoint = aws.String(cfg.AwsEndpoint)
		awsConfig.S3ForcePathStyle = aws.Bool(true)
	}

	sess, err := session.NewSession(awsConfig)
	if err != nil {
		log.Fatalf("Ошибка создания AWS сессии: %v", err)
	}

	// Создаем S3 uploader
	uploader := s3manager.NewUploader(sess)

	// Получаем имя файла для ключа в S3
	fileName := getFileNameWithoutExt(filePath)
	s3Key := fileName + ".mp3"

	fmt.Printf("📤 Загружаем файл в S3:\n")
	fmt.Printf("   Файл: %s\n", filePath)
	fmt.Printf("   Размер: %s\n", formatFileSize(fileSize))
	fmt.Printf("   Бакет: %s\n", cfg.AwsBucketName)
	fmt.Printf("   Ключ: %s\n", s3Key)
	fmt.Println()

	// Создаем канал для отслеживания прогресса
	progressChan := make(chan int64)

	// Запускаем горутину для отображения прогресса
	go func() {
		startTime := time.Now()

		for progress := range progressChan {
			if progress > 0 {
				elapsed := time.Since(startTime)
				percentage := float64(progress) / float64(fileSize) * 100

				// Вычисляем скорость загрузки
				speed := float64(progress) / elapsed.Seconds()

				// Вычисляем оставшееся время
				remainingBytes := fileSize - progress
				var remainingTime time.Duration
				if speed > 0 {
					remainingTime = time.Duration(float64(remainingBytes)/speed) * time.Second
				}

				// Очищаем строку и выводим прогресс
				fmt.Printf("\r📊 Прогресс: %.1f%% | Скорость: %s/s | Прошло: %s | Осталось: %s",
					percentage,
					formatFileSize(int64(speed)),
					formatDuration(elapsed),
					formatDuration(remainingTime))
			}
		}
	}()

	// Создаем кастомный reader для отслеживания прогресса
	progressReader := &ProgressReader{
		Reader: file,
		Size:   fileSize,
		OnProgress: func(bytesRead int64) {
			progressChan <- bytesRead
		},
	}

	// Выполняем загрузку
	_, err = uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(cfg.AwsBucketName),
		Key:    aws.String(s3Key),
		Body:   progressReader,
	})

	// Закрываем канал прогресса
	close(progressChan)

	if err != nil {
		fmt.Printf("\n❌ Ошибка загрузки: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n✅ Файл успешно загружен в S3!\n")
	fmt.Printf("   URL: s3://%s/%s\n", cfg.AwsBucketName, s3Key)
}

// ProgressReader структура для отслеживания прогресса чтения
type ProgressReader struct {
	io.Reader
	Size       int64
	OnProgress func(int64)
	bytesRead  int64
}

func (pr *ProgressReader) Read(p []byte) (n int, err error) {
	n, err = pr.Reader.Read(p)
	pr.bytesRead += int64(n)
	if pr.OnProgress != nil {
		pr.OnProgress(pr.bytesRead)
	}
	return n, err
}

// Функция для форматирования размера файла
func formatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
