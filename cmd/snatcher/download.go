package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/kkdai/youtube/v2"
	"github.com/spf13/cobra"
)

// createDownloadCommand создает команду download с привязкой к экземпляру приложения
func (app *Application) createDownloadCommand(ctx context.Context) *cobra.Command {
	return &cobra.Command{
		Use:   "download [YouTube URL]",
		Short: "Download audio from YouTube video as MP3",
		Long:  `Download audio from YouTube video and save it as MP3 file to the configured download directory.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			// Создаем контекст с таймаутом для скачивания (15 минут)
			downloadCtx, cancel := context.WithTimeout(ctx, 15*time.Minute)
			defer cancel()
			return app.downloadYouTubeAudio(downloadCtx, args[0])
		},
	}
}

// downloadYouTubeAudio скачивает аудио из YouTube видео
func (app *Application) downloadYouTubeAudio(ctx context.Context, url string) error {
	// Извлекаем ID видео из URL
	videoID, err := extractVideoID(url)
	if err != nil {
		return fmt.Errorf("ошибка извлечения ID видео: %w", err)
	}

	fmt.Printf("Скачиваем аудио для видео ID: %s\n", videoID)

	// Создаем YouTube client
	client := youtube.Client{}

	// Получаем информацию о видео с контекстом
	video, err := client.GetVideoContext(ctx, videoID)
	if err != nil {
		return fmt.Errorf("ошибка получения информации о видео: %w", err)
	}

	fmt.Printf("Название: %s\n", video.Title)
	fmt.Printf("Автор: %s\n", video.Author)

	// Находим лучший аудио формат
	audioFormat := findBestAudioFormat(video.Formats)
	if audioFormat == nil {
		return fmt.Errorf("аудио формат не найден")
	}

	fmt.Printf("Используем формат: itag=%d, качество=%s\n", audioFormat.ItagNo, audioFormat.Quality)

	// Получаем поток для скачивания с контекстом
	stream, _, err := client.GetStreamContext(ctx, video, audioFormat)
	if err != nil {
		return fmt.Errorf("ошибка получения потока: %w", err)
	}
	defer stream.Close()

	// Создаем имя файла
	fileName := sanitizeFileName(video.Title) + ".mp3"
	filePath := filepath.Join(app.Config.DownloadDir, fileName)

	// Создаем директорию если она не существует
	if err := os.MkdirAll(app.Config.DownloadDir, 0755); err != nil {
		return fmt.Errorf("ошибка создания директории: %w", err)
	}

	// Создаем файл
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("ошибка создания файла: %w", err)
	}
	defer file.Close()

	// Копируем данные из потока в файл
	fmt.Printf("Скачиваем в файл: %s\n", filePath)

	// Используем io.Copy с проверкой контекста
	done := make(chan error, 1)
	go func() {
		_, err := io.Copy(file, stream)
		done <- err
	}()

	select {
	case err := <-done:
		if err != nil {
			return fmt.Errorf("ошибка скачивания: %w", err)
		}
	case <-ctx.Done():
		return fmt.Errorf("скачивание отменено: %w", ctx.Err())
	}

	fmt.Printf("Аудио успешно скачано: %s\n", filePath)
	return nil
}

// extractVideoID извлекает ID видео из различных форматов YouTube URL
func extractVideoID(url string) (string, error) {
	// Паттерны для различных форматов YouTube URL
	patterns := []string{
		`(?:youtube\.com/watch\?v=|youtu\.be/)([a-zA-Z0-9_-]{11})`,
		`(?:youtube\.com/embed/)([a-zA-Z0-9_-]{11})`,
		`(?:youtube\.com/v/)([a-zA-Z0-9_-]{11})`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(url)
		if len(matches) > 1 {
			return matches[1], nil
		}
	}

	// Если это просто ID видео (11 символов)
	if len(url) == 11 && regexp.MustCompile(`^[a-zA-Z0-9_-]+$`).MatchString(url) {
		return url, nil
	}

	return "", fmt.Errorf("не удалось извлечь ID видео из URL: %s", url)
}

// findBestAudioFormat находит лучший аудио формат для скачивания
func findBestAudioFormat(formats youtube.FormatList) *youtube.Format {
	// Сначала ищем форматы только с аудио
	audioFormats := formats.WithAudioChannels()

	if len(audioFormats) == 0 {
		// Если нет только аудио форматов, ищем видео+аудио форматы
		videoAudioFormats := formats.Type("video")
		for i := range videoAudioFormats {
			if videoAudioFormats[i].AudioChannels > 0 {
				return &videoAudioFormats[i]
			}
		}
		return nil
	}

	// Ищем форматы с лучшим качеством аудио
	bestFormat := &audioFormats[0]

	for i := range audioFormats {
		format := &audioFormats[i]

		// Предпочитаем форматы с более высоким битрейтом
		if format.Bitrate > bestFormat.Bitrate {
			bestFormat = format
		}

		// Предпочитаем MP4/M4A форматы для лучшей совместимости
		if strings.Contains(format.MimeType, "mp4") || strings.Contains(format.MimeType, "m4a") {
			if !strings.Contains(bestFormat.MimeType, "mp4") && !strings.Contains(bestFormat.MimeType, "m4a") {
				bestFormat = format
			}
		}
	}

	return bestFormat
}

// sanitizeFileName очищает имя файла от недопустимых символов
func sanitizeFileName(name string) string {
	// Заменяем недопустимые символы
	re := regexp.MustCompile(`[<>:"/\\|?*]`)
	name = re.ReplaceAllString(name, "_")

	// Убираем лишние пробелы по краям
	name = strings.TrimSpace(name)

	// Заменяем все пробелы на подчеркивания
	name = strings.ReplaceAll(name, " ", "_")

	// Ограничиваем длину имени файла
	if len(name) > 200 {
		name = name[:200]
	}

	return name
}
