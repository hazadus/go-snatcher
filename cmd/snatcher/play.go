package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gopxl/beep"
	"github.com/gopxl/beep/mp3"
	"github.com/gopxl/beep/speaker"
	"github.com/spf13/cobra"
)

// createPlayCommand создает команду play с привязкой к экземпляру приложения
func (app *Application) createPlayCommand(ctx context.Context) *cobra.Command {
	return &cobra.Command{
		Use:   "play [trackid]",
		Short: "Play a track by its ID",
		Long:  `Play an mp3 file by its track ID from the app data.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			trackID, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("неверный ID трека: %s", args[0])
			}
			return app.playByID(ctx, trackID)
		},
	}
}

// enableRawMode включает режим raw для терминала (без буферизации и echo)
func enableRawMode() *exec.Cmd {
	cmd := exec.Command("stty", "-echo", "-icanon")
	cmd.Stdin = os.Stdin
	_ = cmd.Run() // Игнорируем ошибку, так как это не критично для работы плеера
	return cmd
}

// disableRawMode восстанавливает нормальный режим терминала
func disableRawMode() {
	cmd := exec.Command("stty", "echo", "icanon")
	cmd.Stdin = os.Stdin
	_ = cmd.Run() // Игнорируем ошибку, так как это не критично для работы плеера
}

// readSingleChar читает одиночный символ без ожидания Enter
func readSingleChar() (byte, error) {
	buffer := make([]byte, 1)
	_, err := os.Stdin.Read(buffer)
	return buffer[0], err
}

func (app *Application) playByID(ctx context.Context, trackID int) error {
	// Находим трек по ID
	track, err := app.Data.TrackByID(trackID)
	if err != nil {
		return fmt.Errorf("ошибка поиска трека: %w", err)
	}

	if track == nil {
		return fmt.Errorf("трек с ID %d не найден", trackID)
	}

	// Проверяем, что у трека есть URL
	if track.URL == "" {
		return fmt.Errorf("у трека с ID %d отсутствует URL", trackID)
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
	streamReader, err := NewStreamingReader(ctx, track.URL, bufferSize)
	if err != nil {
		return fmt.Errorf("ошибка создания потокового ридера: %w", err)
	}
	defer streamReader.Close()

	fmt.Printf("🌐 Начинаем потоковое воспроизведение...\n")

	// Декодируем MP3 потоково из нашего буферизованного ридера
	streamer, format, err := mp3.Decode(streamReader)
	if err != nil {
		return fmt.Errorf("ошибка декодирования MP3: %w", err)
	}
	defer streamer.Close()

	// Инициализируем speaker с большим буфером для плавного воспроизведения
	err = speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/5)) // 200ms буфер
	if err != nil {
		return fmt.Errorf("ошибка инициализации динамиков: %w", err)
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

	// Создаем контроллер паузы с правильной структурой
	ctrl := &beep.Ctrl{
		Streamer: streamer,
		Paused:   false,
	}
	var isPaused bool                                    // Дополнительная переменная для отслеживания состояния
	var currentStreamer beep.StreamSeekCloser = streamer // Сохраняем ссылку на streamer

	// Запускаем воспроизведение с callback для завершения и контролем паузы
	speaker.Play(beep.Seq(ctrl, beep.Callback(func() {
		done <- true
	})))

	// Отображаем инструкции управления
	fmt.Printf("🎮 Управление:\n")
	fmt.Printf("   [Пробел] - пауза/воспроизведение\n")
	fmt.Printf("   [Ctrl+C] - остановить и выйти\n")
	fmt.Println()

	// Включаем raw режим для чтения одиночных клавиш
	enableRawMode()
	defer disableRawMode() // Восстанавливаем нормальный режим при выходе

	// Запускаем горутину для обработки клавиш
	go func() {
		for {
			char, err := readSingleChar()
			if err != nil {
				continue
			}

			// Проверяем на пробел (ASCII 32) или Enter (ASCII 10/13)
			if char == 32 || char == 10 || char == 13 {
				speaker.Lock()
				isPaused = !isPaused
				ctrl.Paused = isPaused
				speaker.Unlock()

				// Очищаем строку и показываем новое состояние
				fmt.Printf("\r\033[K") // Очищаем текущую строку
				if isPaused {
					fmt.Printf("⏸️  Пауза\n")
				} else {
					fmt.Printf("▶️  Воспроизведение\n")
				}
			}
		}
	}()

	// Запускаем горутину для отображения прогресса с улучшенной информацией
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		startTime := time.Now()
		lastPosition := int64(0)
		stuckCount := 0
		pausedTime := time.Duration(0)
		lastPausedState := false

		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				speaker.Lock()
				currentPos := format.SampleRate.D(currentStreamer.Position())
				totalLen := format.SampleRate.D(currentStreamer.Len())
				currentPauseState := isPaused
				speaker.Unlock()

				// Учитываем время паузы
				if currentPauseState && !lastPausedState {
					// Начало паузы
					pausedTime = time.Since(startTime) - currentPos
				}
				lastPausedState = currentPauseState

				// Проверяем, не застрял ли поток (только если не на паузе)
				currentPosInt := int64(currentPos)
				if !currentPauseState {
					if currentPosInt == lastPosition {
						stuckCount++
						if stuckCount > 5 { // Если позиция не меняется 5 секунд
							fmt.Printf("\n⚠️  Поток может быть заблокирован. Позиция: %s\n", formatDuration(currentPos))
						}
					} else {
						stuckCount = 0
					}
				} else {
					stuckCount = 0 // Сбрасываем счетчик при паузе
				}
				lastPosition = currentPosInt

				// Вычисляем скорость воспроизведения для диагностики
				elapsed := time.Since(startTime) - pausedTime
				var speed float64
				if elapsed > 0 && !currentPauseState {
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
					statusText := getStreamStatus(stuckCount)

					if currentPauseState {
						statusIcon = "⏸️"
						statusText = "На паузе"
					} else if stuckCount > 3 {
						statusIcon = "⚠️"
					} else if speed >= 0.98 && speed <= 1.02 {
						statusIcon = "✅"
					}

					if currentPauseState {
						fmt.Printf("\r%s  %s | %s / %s | Статус: %s",
							statusIcon,
							progress,
							formatDuration(currentPos),
							formatDuration(totalDur),
							statusText)
					} else {
						fmt.Printf("\r%s  %s | %s / %s | Скорость: %.2fx | Статус: %s",
							statusIcon,
							progress,
							formatDuration(currentPos),
							formatDuration(totalDur),
							speed,
							statusText)
					}
				} else {
					if currentPauseState {
						fmt.Printf("\r⏸️  %s | Статус: На паузе | Потоковое воспроизведение",
							formatDuration(currentPos))
					} else {
						fmt.Printf("\r⏱️  %s | Скорость: %.2fx | Потоковое воспроизведение",
							formatDuration(currentPos),
							speed)
					}
				}
			}
		}
	}()

	// Ждем завершения воспроизведения, прерывания или отмены контекста
	select {
	case <-done:
		fmt.Println("\n✅ Потоковое воспроизведение завершено")
	case <-interrupt:
		fmt.Println("\n⏹️  Воспроизведение остановлено пользователем")
		speaker.Clear() // Останавливаем воспроизведение
	case <-ctx.Done():
		fmt.Println("\n🚫 Операция отменена")
		speaker.Clear() // Останавливаем воспроизведение
		return ctx.Err()
	}

	return nil
}
