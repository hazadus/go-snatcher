package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/gopxl/beep"
	"github.com/gopxl/beep/mp3"
	"github.com/gopxl/beep/speaker"
	"github.com/spf13/cobra"
)

var playCmd = &cobra.Command{
	Use:   "play [trackid]",
	Short: "Play a track by its ID",
	Long:  `Play an mp3 file by its track ID from the app data.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		trackID, err := strconv.Atoi(args[0])
		if err != nil {
			log.Fatalf("Неверный ID трека: %s", args[0])
		}
		playByID(trackID)
	},
}

func playByID(trackID int) {
	// Находим трек по ID
	track, err := appData.TrackByID(trackID)
	if err != nil {
		log.Fatalf("Ошибка поиска трека: %v", err)
	}

	if track == nil {
		log.Fatalf("Трек с ID %d не найден", trackID)
	}

	// Проверяем, что у трека есть URL
	if track.URL == "" {
		log.Fatalf("У трека с ID %d отсутствует URL", trackID)
	}

	fmt.Printf("🎵 Воспроизводим трек ID %d: %s - %s\n", trackID, track.Artist, track.Title)

	// Загружаем файл по URL
	reader, err := downloadFromURL(track.URL)
	if err != nil {
		log.Fatalf("Ошибка загрузки файла: %v", err)
	}
	defer reader.Close()

	// Создаем временный файл для чтения метаданных
	tempFile, err := os.CreateTemp("", "snatcher-*.mp3")
	if err != nil {
		log.Fatalf("Ошибка создания временного файла: %v", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	// Копируем данные в временный файл
	_, err = io.Copy(tempFile, reader)
	if err != nil {
		log.Fatalf("Ошибка копирования данных: %v", err)
	}

	// Сбрасываем позицию в файле
	if _, err := tempFile.Seek(0, 0); err != nil {
		log.Fatalf("Ошибка позиционирования в файле: %v", err)
	}

	// Декодируем MP3
	streamer, format, err := mp3.Decode(tempFile)
	if err != nil {
		log.Fatalf("Ошибка декодирования MP3: %v", err)
	}
	defer streamer.Close()

	err = speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))
	if err != nil {
		log.Fatalf("Ошибка инициализации динамиков: %v", err)
	}

	// Получаем длительность трека
	duration := format.SampleRate.D(streamer.Len())

	// Выводим информацию о треке
	fmt.Printf("🎵 Сейчас играет:\n")
	fmt.Printf("   ID: %d\n", track.ID)
	fmt.Printf("   Исполнитель: %s\n", track.Artist)
	fmt.Printf("   Название: %s\n", track.Title)
	fmt.Printf("   Альбом: %s\n", track.Album)
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
