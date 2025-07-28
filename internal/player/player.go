// Package player содержит компоненты для управления воспроизведением аудио
package player

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gopxl/beep"
	"github.com/gopxl/beep/mp3"
	"github.com/gopxl/beep/speaker"

	"github.com/hazadus/go-snatcher/internal/data"
	"github.com/hazadus/go-snatcher/internal/streaming"
)

// Status представляет текущий статус плеера
type Status struct {
	Current    time.Duration // Текущая позиция
	Total      time.Duration // Общая продолжительность
	IsPlaying  bool          // Воспроизводится ли трек
	Speed      float64       // Скорость воспроизведения (для диагностики)
	StuckCount int           // Счетчик зависших состояний
}

// Player управляет воспроизведением треков
type Player struct {
	// Каналы для обратной связи
	progressChan chan Status
	doneChan     chan bool

	// Внутреннее состояние
	ctx           context.Context
	cancel        context.CancelFunc
	mutex         sync.RWMutex
	isInitialized bool
	isPaused      bool
	currentTrack  *data.TrackMetadata

	// Компоненты для воспроизведения
	streamer     beep.StreamSeekCloser
	ctrl         *beep.Ctrl
	streamReader *streaming.Reader
}

// NewPlayer создает новый экземпляр плеера
func NewPlayer() *Player {
	ctx, cancel := context.WithCancel(context.Background())
	return &Player{
		progressChan: make(chan Status, 1),
		doneChan:     make(chan bool, 1),
		ctx:          ctx,
		cancel:       cancel,
	}
}

// Progress возвращает канал для получения обновлений прогресса
func (p *Player) Progress() <-chan Status {
	return p.progressChan
}

// Done возвращает канал, который закрывается при завершении воспроизведения
func (p *Player) Done() <-chan bool {
	return p.doneChan
}

// Play начинает воспроизведение трека
func (p *Player) Play(track *data.TrackMetadata) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// Останавливаем текущее воспроизведение, если есть
	p.stopInternal()

	// Сохраняем информацию о треке
	p.currentTrack = track

	// Создаем потоковый ридер
	const bufferSize = 256 * 1024 // 256KB буфер
	streamReader, err := streaming.NewReader(p.ctx, track.URL, bufferSize)
	if err != nil {
		return fmt.Errorf("ошибка создания потокового ридера: %w", err)
	}
	p.streamReader = streamReader

	// Декодируем MP3
	streamer, format, err := mp3.Decode(streamReader)
	if err != nil {
		streamReader.Close()
		return fmt.Errorf("ошибка декодирования MP3: %w", err)
	}
	p.streamer = streamer

	// Инициализируем speaker (только один раз)
	if !p.isInitialized {
		err = speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/5))
		if err != nil {
			streamer.Close()
			streamReader.Close()
			return fmt.Errorf("ошибка инициализации динамиков: %w", err)
		}
		p.isInitialized = true
	}

	// Создаем контроллер паузы
	p.ctrl = &beep.Ctrl{
		Streamer: streamer,
		Paused:   false,
	}
	p.isPaused = false

	// Запускаем воспроизведение
	speaker.Play(beep.Seq(p.ctrl, beep.Callback(func() {
		// Уведомляем о завершении воспроизведения
		select {
		case p.doneChan <- true:
		default:
		}
	})))

	// Запускаем мониторинг прогресса в отдельной горутине
	go p.monitorProgress(format)

	return nil
}

// Pause приостанавливает или возобновляет воспроизведение
func (p *Player) Pause() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.ctrl != nil {
		speaker.Lock()
		p.isPaused = !p.isPaused
		p.ctrl.Paused = p.isPaused
		speaker.Unlock()
	}
}

// Stop останавливает воспроизведение
func (p *Player) Stop() {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.stopInternal()
}

// stopInternal внутренний метод остановки (должен вызываться под мьютексом)
func (p *Player) stopInternal() {
	if p.ctrl != nil {
		speaker.Clear()
		p.ctrl = nil
	}

	if p.streamer != nil {
		p.streamer.Close()
		p.streamer = nil
	}

	if p.streamReader != nil {
		p.streamReader.Close()
		p.streamReader = nil
	}

	p.currentTrack = nil
	p.isPaused = false
}

// Close закрывает плеер и освобождает ресурсы
func (p *Player) Close() error {
	p.cancel()
	p.Stop()
	close(p.progressChan)
	close(p.doneChan)
	return nil
}

// IsPlaying возвращает true, если трек воспроизводится
func (p *Player) IsPlaying() bool {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.ctrl != nil && !p.isPaused
}

// CurrentTrack возвращает информацию о текущем треке
func (p *Player) CurrentTrack() *data.TrackMetadata {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.currentTrack
}

// monitorProgress мониторит прогресс воспроизведения и отправляет обновления
func (p *Player) monitorProgress(format beep.Format) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	lastPosition := int64(0)
	stuckCount := 0
	startTime := time.Now()
	pausedTime := time.Duration(0)
	lastPausedState := false

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			p.mutex.RLock()

			if p.streamer == nil || p.ctrl == nil {
				p.mutex.RUnlock()
				return
			}

			speaker.Lock()
			currentPos := format.SampleRate.D(p.streamer.Position())
			totalLen := format.SampleRate.D(p.streamer.Len())
			currentPauseState := p.isPaused
			speaker.Unlock()

			// Учитываем время паузы
			if currentPauseState && !lastPausedState {
				pausedTime = time.Since(startTime) - currentPos
			}
			lastPausedState = currentPauseState

			// Проверяем, не застрял ли поток
			currentPosInt := int64(currentPos)
			if !currentPauseState {
				if currentPosInt == lastPosition {
					stuckCount++
				} else {
					stuckCount = 0
				}
			} else {
				stuckCount = 0
			}
			lastPosition = currentPosInt

			// Вычисляем скорость воспроизведения
			elapsed := time.Since(startTime) - pausedTime
			var speed float64
			if elapsed > 0 && !currentPauseState {
				speed = float64(currentPos) / float64(elapsed)
			}

			// Определяем общую продолжительность
			var duration time.Duration
			if p.currentTrack != nil && p.currentTrack.Length > 0 {
				duration = time.Duration(p.currentTrack.Length) * time.Second
			} else if totalLen > 0 {
				duration = totalLen
			}

			p.mutex.RUnlock()

			// Отправляем обновление статуса
			status := Status{
				Current:    currentPos,
				Total:      duration,
				IsPlaying:  !currentPauseState,
				Speed:      speed,
				StuckCount: stuckCount,
			}

			select {
			case p.progressChan <- status:
			default:
				// Если канал заблокирован, пропускаем обновление
			}
		}
	}
}
