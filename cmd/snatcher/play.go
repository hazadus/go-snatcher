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

	"github.com/spf13/cobra"

	"github.com/hazadus/go-snatcher/internal/player"
	"github.com/hazadus/go-snatcher/internal/streaming"
	"github.com/hazadus/go-snatcher/internal/utils"
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
	if track.Length > 0 {
		duration := utils.FormatDuration(time.Duration(track.Length) * time.Second)
		fmt.Printf("   Продолжительность: %s\n", duration)
	}
	fmt.Println()

	// Создаем плеер
	p := player.NewPlayer()
	defer p.Close()

	// Запускаем воспроизведение
	err = p.Play(track)
	if err != nil {
		return fmt.Errorf("ошибка запуска воспроизведения: %w", err)
	}

	fmt.Printf("🌐 Начинаем потоковое воспроизведение...\n")
	fmt.Printf("🎮 Управление:\n")
	fmt.Printf("   [Пробел] - пауза/воспроизведение\n")
	fmt.Printf("   [Ctrl+C] - остановить и выйти\n")
	fmt.Println()

	// Включаем raw режим для чтения одиночных клавиш
	enableRawMode()
	defer disableRawMode()

	// Создаем канал для обработки сигналов прерывания
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	// Запускаем горутину для обработки клавиш
	go func() {
		for {
			char, err := readSingleChar()
			if err != nil {
				continue
			}

			// Проверяем на пробел (ASCII 32) или Enter (ASCII 10/13)
			if char == 32 || char == 10 || char == 13 {
				p.Pause()
				// Показываем новое состояние
				fmt.Printf("\r\033[K") // Очищаем текущую строку
				if p.IsPlaying() {
					fmt.Printf("▶️  Воспроизведение\n")
				} else {
					fmt.Printf("⏸️  Пауза\n")
				}
			}
		}
	}()

	// Главный цикл обработки событий
	for {
		select {
		case status := <-p.Progress():
			// Обновляем прогресс
			displayProgress(status)
		case <-p.Done():
			fmt.Println("\n✅ Потоковое воспроизведение завершено")
			return nil
		case <-interrupt:
			fmt.Println("\n⏹️  Воспроизведение остановлено пользователем")
			p.Stop()
			return nil
		case <-ctx.Done():
			fmt.Println("\n🚫 Операция отменена")
			p.Stop()
			return ctx.Err()
		}
	}
}

// displayProgress отображает прогресс воспроизведения
func displayProgress(status player.Status) {
	// Определяем процент завершения
	var progress string
	if status.Total > 0 {
		percent := float64(status.Current) / float64(status.Total) * 100
		progress = fmt.Sprintf("%.1f%%", percent)
	} else {
		progress = "??%"
	}

	// Выбираем иконку статуса
	statusIcon := "⏱️"
	statusText := streaming.GetStreamStatus(status.StuckCount)

	if !status.IsPlaying {
		statusIcon = "⏸️"
		statusText = "На паузе"
	} else if status.StuckCount > 3 {
		statusIcon = "⚠️"
	} else if status.Speed >= 0.98 && status.Speed <= 1.02 {
		statusIcon = "✅"
	}

	// Отображаем прогресс
	if status.Total > 0 {
		if !status.IsPlaying {
			fmt.Printf("\r%s  %s | %s / %s | Статус: %s",
				statusIcon,
				progress,
				utils.FormatDuration(status.Current),
				utils.FormatDuration(status.Total),
				statusText)
		} else {
			fmt.Printf("\r%s  %s | %s / %s | Скорость: %.2fx | Статус: %s",
				statusIcon,
				progress,
				utils.FormatDuration(status.Current),
				utils.FormatDuration(status.Total),
				status.Speed,
				statusText)
		}
	} else {
		if !status.IsPlaying {
			fmt.Printf("\r⏸️  %s | Статус: На паузе | Потоковое воспроизведение",
				utils.FormatDuration(status.Current))
		} else {
			fmt.Printf("\r⏱️  %s | Скорость: %.2fx | Потоковое воспроизведение",
				utils.FormatDuration(status.Current),
				status.Speed)
		}
	}
}
