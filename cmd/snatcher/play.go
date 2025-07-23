package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

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
		if _, err := seeker.Seek(0, 0); err != nil {
			log.Printf("Seek error: %v", err)
			return // корректный выход из функции, если это main handler
		}
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
