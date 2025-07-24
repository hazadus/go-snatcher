package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
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
	Run: func(_ *cobra.Command, args []string) {
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

	fmt.Printf("🎵 Сейчас играет:\n")
	fmt.Printf("   ID: %d\n", track.ID)
	fmt.Printf("   Исполнитель: %s\n", track.Artist)
	fmt.Printf("   Название: %s\n", track.Title)
	fmt.Printf("   Альбом: %s\n", track.Album)
	fmt.Println()

	// Создаем потоковый ридер с большим буфером для стабильного воспроизведения
	const bufferSize = 256 * 1024 // 256KB буфер для более стабильного потока
	streamReader, err := NewStreamingReader(track.URL, bufferSize)
	if err != nil {
		log.Fatalf("Ошибка создания потокового ридера: %v", err)
	}
	defer streamReader.Close()

	fmt.Printf("🌐 Начинаем потоковое воспроизведение...\n")

	// Декодируем MP3 потоково из нашего буферизованного ридера
	streamer, format, err := mp3.Decode(streamReader)
	if err != nil {
		log.Fatalf("Ошибка декодирования MP3: %v", err)
	}
	defer streamer.Close()

	// Инициализируем speaker с большим буфером для плавного воспроизведения
	err = speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/5)) // 200ms буфер
	if err != nil {
		log.Fatalf("Ошибка инициализации динамиков: %v", err)
	}

	// Получаем длительность трека из локальных данных, если доступна
	var duration time.Duration
	if track.Length > 0 {
		duration = time.Duration(track.Length) * time.Second
		fmt.Printf("   Продолжительность: %s\n", formatDuration(duration))
	} else {
		fmt.Printf("   Продолжительность: определяется в процессе воспроизведения...\n")
	}
	fmt.Printf("   Размер буфера: %d KB\n", bufferSize/1024)
	fmt.Printf("   Качество: %d-bit, %d Hz, %d каналов\n", format.Precision, format.SampleRate, format.NumChannels)
	fmt.Println()

	// Создаем канал для сигнала завершения
	done := make(chan bool)

	// Создаем канал для обработки сигналов прерывания
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	// Запускаем воспроизведение с callback для завершения
	speaker.Play(beep.Seq(streamer, beep.Callback(func() {
		done <- true
	})))

	// Запускаем горутину для отображения прогресса с улучшенной информацией
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		startTime := time.Now()
		lastPosition := int64(0)
		stuckCount := 0

		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				speaker.Lock()
				currentPos := format.SampleRate.D(streamer.Position())
				totalLen := format.SampleRate.D(streamer.Len())
				speaker.Unlock()

				// Проверяем, не застрял ли поток
				currentPosInt := int64(currentPos)
				if currentPosInt == lastPosition {
					stuckCount++
					if stuckCount > 5 { // Если позиция не меняется 5 секунд
						fmt.Printf("\n⚠️  Поток может быть заблокирован. Позиция: %s\n", formatDuration(currentPos))
					}
				} else {
					stuckCount = 0
				}
				lastPosition = currentPosInt

				// Вычисляем скорость воспроизведения для диагностики
				elapsed := time.Since(startTime)
				var speed float64
				if elapsed > 0 {
					speed = float64(currentPos) / float64(elapsed)
				}

				// Определяем процент завершения
				var progress string
				if track.Length > 0 && duration > 0 {
					// Используем локальные данные о длительности
					percent := float64(currentPos) / float64(duration) * 100
					progress = fmt.Sprintf("%.1f%%", percent)
				} else if totalLen > 0 {
					// Используем данные из потока
					percent := float64(currentPos) / float64(totalLen) * 100
					progress = fmt.Sprintf("%.1f%%", percent)
				} else {
					progress = "??%"
				}

				// Показываем детальный прогресс
				if totalLen > 0 || duration > 0 {
					totalDur := duration
					if totalDur == 0 {
						totalDur = totalLen
					}

					statusIcon := "⏱️"
					if stuckCount > 3 {
						statusIcon = "⚠️"
					} else if speed >= 0.98 && speed <= 1.02 {
						statusIcon = "✅"
					}

					fmt.Printf("\r%s  %s | %s / %s | Скорость: %.2fx | Статус: %s",
						statusIcon,
						progress,
						formatDuration(currentPos),
						formatDuration(totalDur),
						speed,
						getStreamStatus(stuckCount))
				} else {
					fmt.Printf("\r⏱️  %s | Скорость: %.2fx | Потоковое воспроизведение",
						formatDuration(currentPos),
						speed)
				}
			}
		}
	}()

	// Ждем завершения воспроизведения или прерывания
	select {
	case <-done:
		fmt.Println("\n✅ Потоковое воспроизведение завершено")
	case <-interrupt:
		fmt.Println("\n⏹️  Воспроизведение остановлено пользователем")
		speaker.Clear() // Останавливаем воспроизведение
	}
}
