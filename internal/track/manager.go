// Package track содержит логику управления треками
package track

import (
	"github.com/hazadus/go-snatcher/internal/data"
)

// Manager управляет треками в приложении
type Manager struct {
	appData *data.AppData
}

// NewManager создает новый экземпляр Manager
func NewManager(appData *data.AppData) *Manager {
	return &Manager{
		appData: appData,
	}
}

// ListTracks возвращает список всех треков
func (m *Manager) ListTracks() []data.TrackMetadata {
	return m.appData.Tracks
}
