// Package data содержит структуры и методы для работы с данными приложения
package data

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// TrackMetadata содержит метаданные музыкального трека
type TrackMetadata struct {
	ID               int    `yaml:"id"`
	Artist           string `yaml:"artist"`
	Title            string `yaml:"title"`
	Album            string `yaml:"album"`
	Length           int    `yaml:"length"`            // Длина трека в секундах
	FileSize         int64  `yaml:"file_size"`         // Размер файла в байтах
	URL              string `yaml:"url"`               // URL трека в хранилище S3
	SourceURL        string `yaml:"source_url"`        // URL источника, откуда скачан материал
	PlaybackPosition int    `yaml:"playback_position"` // Сохраненная позиция воспроизведения в секундах
}

// AppData содержит все данные приложения
type AppData struct {
	Tracks []TrackMetadata `yaml:"tracks"`
}

// NewAppData создает новую структуру AppData
func NewAppData() *AppData {
	return &AppData{
		Tracks: make([]TrackMetadata, 0),
	}
}

// LoadData загружает данные из файла
func (d *AppData) LoadData(filePath string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	path := strings.Replace(filePath, "~", home, 1)

	data, err := os.ReadFile(path)
	if err != nil {
		// Если файл не найден, инициализируем пустыми данными
		if os.IsNotExist(err) {
			*d = *NewAppData()
			return nil
		}
		return fmt.Errorf("ошибка чтения файла данных: %w", err)
	}
	if len(data) == 0 {
		*d = *NewAppData()
		return nil
	}
	if err := yaml.Unmarshal(data, d); err != nil {
		return fmt.Errorf("ошибка разбора данных: %w", err)
	}
	return nil
}

// AddTrack добавляет новый трек в AppData
func (d *AppData) AddTrack(track TrackMetadata) {
	// Найдем максимальный ID и присваиваем новый треку
	if len(d.Tracks) > 0 {
		maxID := d.Tracks[0].ID
		for _, t := range d.Tracks {
			if t.ID > maxID {
				maxID = t.ID
			}
		}
		track.ID = maxID + 1
	} else {
		track.ID = 1 // Если треков нет, начинаем с 1
	}
	// Добавляем трек в список
	d.Tracks = append(d.Tracks, track)
}

// SaveData сохраняет данные в файл
func (d *AppData) SaveData(filePath string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	path := strings.Replace(filePath, "~", home, 1)

	data, err := yaml.Marshal(d)
	if err != nil {
		return fmt.Errorf("ошибка сериализации данных: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("ошибка записи файла данных: %w", err)
	}
	return nil
}

// TrackByID возвращает трек по ID
func (d *AppData) TrackByID(id int) (*TrackMetadata, error) {
	for i := range d.Tracks {
		if d.Tracks[i].ID == id {
			return &d.Tracks[i], nil
		}
	}
	return nil, fmt.Errorf("трека с ID %d не найдено", id)
}

// DeleteTrackByID удаляет трек по ID
func (d *AppData) DeleteTrackByID(id int) error {
	for i, track := range d.Tracks {
		if track.ID == id {
			// Удаляем элемент из слайса
			d.Tracks = append(d.Tracks[:i], d.Tracks[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("трека с ID %d не найдено", id)
}

// UpdateTrack обновляет существующий трек
func (d *AppData) UpdateTrack(updatedTrack TrackMetadata) error {
	for i := range d.Tracks {
		if d.Tracks[i].ID == updatedTrack.ID {
			d.Tracks[i] = updatedTrack
			return nil
		}
	}
	return fmt.Errorf("трека с ID %d не найдено", updatedTrack.ID)
}

// UpdateTrackPlaybackPosition обновляет позицию воспроизведения трека
func (d *AppData) UpdateTrackPlaybackPosition(trackID int, position int) error {
	for i := range d.Tracks {
		if d.Tracks[i].ID == trackID {
			d.Tracks[i].PlaybackPosition = position
			return nil
		}
	}
	return fmt.Errorf("трека с ID %d не найдено", trackID)
}
