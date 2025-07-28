package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestLoadConfigFromFile(t *testing.T) {
	// Создаем временный файл конфигурации
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	// Создаем тестовую конфигурацию
	testConfig := Config{
		AwsBucketName: "test-bucket",
		AwsAccessKey:  "test-access-key",
		AwsSecretKey:  "test-secret-key",
		AwsRegion:     "us-east-1",
		AwsEndpoint:   "https://s3.amazonaws.com",
		DownloadDir:   "~/test-downloads",
	}

	// Сериализуем конфигурацию в YAML
	data, err := yaml.Marshal(testConfig)
	if err != nil {
		t.Fatalf("Ошибка сериализации конфигурации: %v", err)
	}

	// Записываем в файл
	err = os.WriteFile(configPath, data, 0644)
	if err != nil {
		t.Fatalf("Ошибка записи файла конфигурации: %v", err)
	}

	// Загружаем конфигурацию
	loadedConfig, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Ошибка загрузки конфигурации: %v", err)
	}

	// Проверяем, что конфигурация загружена корректно
	if loadedConfig.AwsBucketName != testConfig.AwsBucketName {
		t.Errorf("Ожидался AwsBucketName: %s, получено: %s", testConfig.AwsBucketName, loadedConfig.AwsBucketName)
	}
	if loadedConfig.AwsAccessKey != testConfig.AwsAccessKey {
		t.Errorf("Ожидался AwsAccessKey: %s, получено: %s", testConfig.AwsAccessKey, loadedConfig.AwsAccessKey)
	}
	if loadedConfig.AwsSecretKey != testConfig.AwsSecretKey {
		t.Errorf("Ожидался AwsSecretKey: %s, получено: %s", testConfig.AwsSecretKey, loadedConfig.AwsSecretKey)
	}
	if loadedConfig.AwsRegion != testConfig.AwsRegion {
		t.Errorf("Ожидался AwsRegion: %s, получено: %s", testConfig.AwsRegion, loadedConfig.AwsRegion)
	}
	if loadedConfig.AwsEndpoint != testConfig.AwsEndpoint {
		t.Errorf("Ожидался AwsEndpoint: %s, получено: %s", testConfig.AwsEndpoint, loadedConfig.AwsEndpoint)
	}

	// Проверяем, что DownloadDir раскрывается с тильдой
	home, _ := os.UserHomeDir()
	expectedDownloadDir := strings.Replace(testConfig.DownloadDir, "~", home, 1)
	if loadedConfig.DownloadDir != expectedDownloadDir {
		t.Errorf("Ожидался DownloadDir: %s, получено: %s", expectedDownloadDir, loadedConfig.DownloadDir)
	}
}

func TestDefaultConfig(t *testing.T) {
	// Создаем временный файл конфигурации с минимальными данными
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "minimal_config.yaml")

	// Создаем минимальную конфигурацию (без DownloadDir)
	minimalConfig := map[string]string{
		"aws_bucket_name": "test-bucket",
		"aws_access_key":  "test-key",
	}

	// Сериализуем в YAML
	data, err := yaml.Marshal(minimalConfig)
	if err != nil {
		t.Fatalf("Ошибка сериализации конфигурации: %v", err)
	}

	// Записываем в файл
	err = os.WriteFile(configPath, data, 0644)
	if err != nil {
		t.Fatalf("Ошибка записи файла конфигурации: %v", err)
	}

	// Загружаем конфигурацию
	loadedConfig, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Ошибка загрузки конфигурации: %v", err)
	}

	// Проверяем, что DownloadDir установлен по умолчанию
	home, _ := os.UserHomeDir()
	expectedDownloadDir := filepath.Join(home, "Downloads")
	if loadedConfig.DownloadDir != expectedDownloadDir {
		t.Errorf("Ожидался DownloadDir по умолчанию: %s, получено: %s", expectedDownloadDir, loadedConfig.DownloadDir)
	}

	// Проверяем, что остальные поля загружены корректно
	if loadedConfig.AwsBucketName != "test-bucket" {
		t.Errorf("Ожидался AwsBucketName: test-bucket, получено: %s", loadedConfig.AwsBucketName)
	}
	if loadedConfig.AwsAccessKey != "test-key" {
		t.Errorf("Ожидался AwsAccessKey: test-key, получено: %s", loadedConfig.AwsAccessKey)
	}
}

func TestEnvVarOverride(t *testing.T) {
	// Создаем временный файл конфигурации
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	// Создаем базовую конфигурацию
	baseConfig := Config{
		AwsBucketName: "default-bucket",
		AwsAccessKey:  "default-key",
		AwsRegion:     "us-west-1",
		DownloadDir:   "~/default-downloads",
	}

	// Сериализуем в YAML
	data, err := yaml.Marshal(baseConfig)
	if err != nil {
		t.Fatalf("Ошибка сериализации конфигурации: %v", err)
	}

	// Записываем в файл
	err = os.WriteFile(configPath, data, 0644)
	if err != nil {
		t.Fatalf("Ошибка записи файла конфигурации: %v", err)
	}

	// Устанавливаем переменные окружения
	originalBucketName := os.Getenv("AWS_BUCKET_NAME")
	originalAccessKey := os.Getenv("AWS_ACCESS_KEY")
	defer func() {
		if originalBucketName != "" {
			os.Setenv("AWS_BUCKET_NAME", originalBucketName)
		} else {
			os.Unsetenv("AWS_BUCKET_NAME")
		}
		if originalAccessKey != "" {
			os.Setenv("AWS_ACCESS_KEY", originalAccessKey)
		} else {
			os.Unsetenv("AWS_ACCESS_KEY")
		}
	}()

	// Устанавливаем переменные окружения для переопределения
	os.Setenv("AWS_BUCKET_NAME", "env-bucket")
	os.Setenv("AWS_ACCESS_KEY", "env-key")

	// Загружаем конфигурацию
	loadedConfig, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Ошибка загрузки конфигурации: %v", err)
	}

	// Проверяем, что значения из файла загружены корректно
	if loadedConfig.AwsBucketName != "default-bucket" {
		t.Errorf("Ожидался AwsBucketName из файла: default-bucket, получено: %s", loadedConfig.AwsBucketName)
	}
	if loadedConfig.AwsAccessKey != "default-key" {
		t.Errorf("Ожидался AwsAccessKey из файла: default-key, получено: %s", loadedConfig.AwsAccessKey)
	}

	// Примечание: текущая реализация LoadConfig не поддерживает переопределение через переменные окружения
	// Этот тест демонстрирует текущее поведение
}

func TestLoadConfigNonExistentFile(t *testing.T) {
	// Пытаемся загрузить несуществующий файл
	_, err := LoadConfig("/non/existent/config.yaml")

	if err == nil {
		t.Error("Ожидалась ошибка при загрузке несуществующего файла")
	}

	if !strings.Contains(err.Error(), "no such file") && !strings.Contains(err.Error(), "not found") {
		t.Errorf("Неожиданное сообщение об ошибке: %v", err)
	}
}

func TestLoadConfigInvalidYAML(t *testing.T) {
	// Создаем временный файл с некорректным YAML
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "invalid_config.yaml")

	// Записываем некорректный YAML
	invalidYAML := `aws_bucket_name: "test-bucket"
aws_access_key: "test-key"
invalid_field: [unclosed array
`
	err := os.WriteFile(configPath, []byte(invalidYAML), 0644)
	if err != nil {
		t.Fatalf("Ошибка записи файла конфигурации: %v", err)
	}

	// Пытаемся загрузить некорректный файл
	_, err = LoadConfig(configPath)

	if err == nil {
		t.Error("Ожидалась ошибка при загрузке некорректного YAML")
	}

	if !strings.Contains(err.Error(), "yaml") {
		t.Errorf("Неожиданное сообщение об ошибке: %v", err)
	}
}

func TestLoadConfigWithTilde(t *testing.T) {
	// Создаем временный файл конфигурации с тильдой в пути
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	// Создаем конфигурацию с тильдой в DownloadDir
	testConfig := Config{
		AwsBucketName: "test-bucket",
		DownloadDir:   "~/custom-downloads",
	}

	// Сериализуем в YAML
	data, err := yaml.Marshal(testConfig)
	if err != nil {
		t.Fatalf("Ошибка сериализации конфигурации: %v", err)
	}

	// Записываем в файл
	err = os.WriteFile(configPath, data, 0644)
	if err != nil {
		t.Fatalf("Ошибка записи файла конфигурации: %v", err)
	}

	// Загружаем конфигурацию
	loadedConfig, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Ошибка загрузки конфигурации: %v", err)
	}

	// Проверяем, что тильда раскрывается корректно
	home, _ := os.UserHomeDir()
	expectedDownloadDir := filepath.Join(home, "custom-downloads")
	if loadedConfig.DownloadDir != expectedDownloadDir {
		t.Errorf("Ожидался DownloadDir с раскрытой тильдой: %s, получено: %s", expectedDownloadDir, loadedConfig.DownloadDir)
	}
}
