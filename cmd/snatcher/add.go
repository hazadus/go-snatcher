package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/spf13/cobra"

	"github.com/hazadus/go-snatcher/internal/data"
)

var addCmd = &cobra.Command{
	Use:   "add [file path]",
	Short: "Upload an mp3 file to S3 storage",
	Long:  `Upload an mp3 file to S3 storage with progress tracking.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		uploadToS3(args[0])
	},
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
	url := fmt.Sprintf("%s/%s/%s", cfg.AwsEndpoint, cfg.AwsBucketName, s3Key)
	fmt.Printf("   URL: %s\n", url)

	// Получаем реальные метаданные трека
	fileForMeta, err := os.Open(filePath)
	if err != nil {
		log.Printf("Ошибка открытия файла для метаданных: %v", err)
	}
	defer fileForMeta.Close()

	meta := getMetadataFromReader(fileForMeta, filePath)

	// Получаем длительность трека
	duration, err := getMP3Duration(filePath)
	if err != nil {
		log.Printf("Ошибка определения длительности трека: %v", err)
		duration = 0
	}

	track := data.TrackMetadata{
		Artist:   meta.Artist,
		Title:    meta.Title,
		Album:    meta.Album,
		Length:   int(duration.Seconds()),
		FileSize: fileSize,
		URL:      url,
	}

	// Добавляем трек
	appData.AddTrack(track)

	// Сохраняем данные
	if err := appData.SaveData(defaultDataFilePath); err != nil {
		fmt.Printf("\n❌ Ошибка сохранения данных: %v\n", err)
	} else {
		fmt.Printf("\n📦 Данные трека добавлены в %s\n", defaultDataFilePath)
	}
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
