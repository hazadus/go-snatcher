package main

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/hazadus/go-snatcher/internal/s3"
)

// createDeleteCommand создает команду delete с привязкой к экземпляру приложения
func (app *Application) createDeleteCommand(ctx context.Context) *cobra.Command {
	return &cobra.Command{
		Use:   "delete [id]",
		Short: "Delete a track by ID",
		Long:  `Delete a track from both S3 storage and local data by its ID.`,
		Args:  cobra.ExactArgs(1),
		Run: func(_ *cobra.Command, args []string) {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				fmt.Printf("❌ Ошибка: неверный ID '%s'. ID должен быть числом.\n", args[0])
				return
			}
			app.deleteTrack(ctx, id)
		},
	}
}

func (app *Application) deleteTrack(ctx context.Context, id int) {
	// Находим трек по ID
	track, err := app.Data.TrackByID(id)
	if err != nil {
		fmt.Printf("❌ Ошибка: %v\n", err)
		return
	}

	fmt.Printf("🗑️  Удаляем трек: %s - %s\n", track.Artist, track.Title)

	// Удаляем файл из S3, если есть URL
	if track.URL != "" {
		if err := app.deleteFromS3(ctx, track.URL); err != nil {
			fmt.Printf("⚠️  Предупреждение: не удалось удалить файл из S3: %v\n", err)
			// Продолжаем выполнение, даже если не удалось удалить из S3
		} else {
			fmt.Println("✅ Файл успешно удален из S3")
		}
	}

	// Удаляем трек из локальных данных
	if err := app.Data.DeleteTrackByID(id); err != nil {
		fmt.Printf("❌ Ошибка удаления трека из данных: %v\n", err)
		return
	}

	// Сохраняем обновленные данные
	if err := app.SaveData(); err != nil {
		fmt.Printf("❌ Ошибка сохранения данных: %v\n", err)
		return
	}

	fmt.Println("✅ Трек успешно удален из библиотеки")
}

func (app *Application) deleteFromS3(ctx context.Context, fileURL string) error {
	// Создаем S3 uploader
	s3Config := &s3.Config{
		Region:     app.Config.AwsRegion,
		AccessKey:  app.Config.AwsAccessKey,
		SecretKey:  app.Config.AwsSecretKey,
		Endpoint:   app.Config.AwsEndpoint,
		BucketName: app.Config.AwsBucketName,
	}

	uploader, err := s3.NewUploader(s3Config)
	if err != nil {
		return fmt.Errorf("ошибка создания S3 клиента: %w", err)
	}

	// Извлекаем ключ из URL
	key, err := extractKeyFromURL(fileURL)
	if err != nil {
		return fmt.Errorf("ошибка извлечения ключа из URL: %w", err)
	}

	// Удаляем файл из S3
	return uploader.DeleteFile(ctx, key)
}

// extractKeyFromURL извлекает ключ файла из URL S3
func extractKeyFromURL(fileURL string) (string, error) {
	parsedURL, err := url.Parse(fileURL)
	if err != nil {
		return "", fmt.Errorf("неверный URL: %w", err)
	}

	// Извлекаем путь без начального слеша и удаляем bucket name
	pathSegments := strings.TrimPrefix(parsedURL.Path, "/")

	// URL обычно имеет формат: endpoint/bucket/key
	// Нам нужно извлечь только key (все после bucket name)
	parts := strings.SplitN(pathSegments, "/", 2)
	if len(parts) < 2 {
		return "", fmt.Errorf("неверный формат URL S3")
	}

	// Возвращаем все части после bucket name
	return parts[1], nil
}
