// Package config содержит функции для загрузки конфигурации приложения
package config

import (
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config структура для хранения конфигурации приложения
type Config struct {
	AwsBucketName string `yaml:"aws_bucket_name"`
	AwsAccessKey  string `yaml:"aws_access_key"`
	AwsSecretKey  string `yaml:"aws_secret_key"`
	AwsRegion     string `yaml:"aws_region"`
	AwsEndpoint   string `yaml:"aws_endpoint"`
	DownloadDir   string `yaml:"download_dir"`
}

// LoadConfig загружает конфигурацию приложения из указанного файла
func LoadConfig(filePath string) (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	path := strings.Replace(filePath, "~", home, 1)

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	config := &Config{}
	err = yaml.Unmarshal(data, config)
	if err != nil {
		return nil, err
	}
	
	// Устанавливаем значения по умолчанию, если они не заданы
	if config.DownloadDir == "" {
		config.DownloadDir = "~/Downloads"
	}
	
	// Раскрываем тильду в пути загрузки
	config.DownloadDir = strings.Replace(config.DownloadDir, "~", home, 1)
	
	return config, nil
}
