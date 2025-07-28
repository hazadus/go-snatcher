package main

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/hazadus/go-snatcher/internal/metadata"
	"github.com/hazadus/go-snatcher/internal/s3"
	"github.com/hazadus/go-snatcher/internal/uploader"
)

// createAddCommand создает команду add с привязкой к экземпляру приложения
func (app *Application) createAddCommand(ctx context.Context) *cobra.Command {
	return &cobra.Command{
		Use:   "add [file path]",
		Short: "Upload an mp3 file to S3 storage",
		Long:  `Upload an mp3 file to S3 storage with progress tracking.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			// Создаем контекст с таймаутом для загрузки (10 минут)
			uploadCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
			defer cancel()
			return app.uploadToS3(uploadCtx, args[0])
		},
	}
}

// uploadToS3 загружает файл в S3 с отображением прогресса
func (app *Application) uploadToS3(ctx context.Context, filePath string) error {
	// Создаем S3 uploader
	s3Config := &s3.Config{
		Region:     app.Config.AwsRegion,
		AccessKey:  app.Config.AwsAccessKey,
		SecretKey:  app.Config.AwsSecretKey,
		Endpoint:   app.Config.AwsEndpoint,
		BucketName: app.Config.AwsBucketName,
	}

	s3Uploader, err := s3.NewUploader(s3Config)
	if err != nil {
		return fmt.Errorf("ошибка создания S3 uploader: %w", err)
	}

	// Создаем сервис загрузки
	uploadService := uploader.NewService(s3Uploader, app.Data)

	// Получаем информацию о файле для отображения
	metadataExtractor := metadata.NewExtractor()
	fileInfo, err := metadataExtractor.GetFileInfo(filePath)
	if err != nil {
		return fmt.Errorf("ошибка получения информации о файле: %w", err)
	}

	// Отображаем информацию о загрузке
	fmt.Printf("📤 Загружаем файл в S3:\n")
	fmt.Printf("   Файл: %s\n", filePath)
	fmt.Printf("   Размер: %s\n", uploader.FormatFileSize(fileInfo.Size))
	fmt.Printf("   Бакет: %s\n", app.Config.AwsBucketName)
	fmt.Println()

	// Создаем канал для отслеживания прогресса
	progressChan := make(chan int64)

	// Запускаем горутину для отображения прогресса
	go func() {
		startTime := time.Now()

		for {
			select {
			case progress, ok := <-progressChan:
				if !ok {
					return // Канал закрыт
				}
				if progress > 0 {
					elapsed := time.Since(startTime)
					percentage := float64(progress) / float64(fileInfo.Size) * 100

					// Вычисляем скорость загрузки
					speed := float64(progress) / elapsed.Seconds()

					// Вычисляем оставшееся время
					remainingBytes := fileInfo.Size - progress
					var remainingTime time.Duration
					if speed > 0 {
						remainingTime = time.Duration(float64(remainingBytes)/speed) * time.Second
					}

					// Очищаем строку и выводим прогресс
					fmt.Printf("\r📊 Прогресс: %.1f%% | Скорость: %s/s | Прошло: %s | Осталось: %s",
						percentage,
						uploader.FormatFileSize(int64(speed)),
						uploader.FormatDuration(elapsed),
						uploader.FormatDuration(remainingTime))
				}
			case <-ctx.Done():
				fmt.Printf("\n🚫 Загрузка отменена\n")
				return
			}
		}
	}()

	// Выполняем загрузку с контекстом
	result, err := uploadService.UploadFile(ctx, filePath, func(bytesRead int64) {
		progressChan <- bytesRead
	})

	// Закрываем канал прогресса
	close(progressChan)

	if err != nil {
		return fmt.Errorf("ошибка загрузки файла: %w", err)
	}

	// Проверяем, не была ли операция отменена
	if ctx.Err() != nil {
		return fmt.Errorf("операция отменена: %w", ctx.Err())
	}

	fmt.Printf("\n✅ Файл успешно загружен в S3!\n")
	fmt.Printf("   URL: %s\n", result.URL)

	// Обновляем данные приложения
	if err := uploadService.UpdateApplicationData(result); err != nil {
		return fmt.Errorf("ошибка обновления данных приложения: %w", err)
	}

	// Сохраняем данные
	if err := app.SaveData(); err != nil {
		return fmt.Errorf("ошибка сохранения данных: %w", err)
	}

	fmt.Printf("\n📦 Данные трека добавлены в %s\n", defaultDataFilePath)
	return nil
}
