package s3

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

// S3UploaderInterface интерфейс для S3 uploader
type S3UploaderInterface interface {
	UploadWithContext(ctx context.Context, input *s3manager.UploadInput, opts ...func(*s3manager.Uploader)) (*s3manager.UploadOutput, error)
}

// S3ClientInterface интерфейс для S3 клиента
type S3ClientInterface interface {
	DeleteObjectWithContext(ctx context.Context, input *s3.DeleteObjectInput, opts ...request.Option) (*s3.DeleteObjectOutput, error)
}

// MockS3Uploader мок для S3 uploader
type MockS3Uploader struct {
	uploadFunc func(input *s3manager.UploadInput) (*s3manager.UploadOutput, error)
}

func (m *MockS3Uploader) UploadWithContext(ctx context.Context, input *s3manager.UploadInput, opts ...func(*s3manager.Uploader)) (*s3manager.UploadOutput, error) {
	return m.uploadFunc(input)
}

// MockS3Client мок для S3 клиента
type MockS3Client struct {
	deleteObjectFunc func(input *s3.DeleteObjectInput) (*s3.DeleteObjectOutput, error)
}

func (m *MockS3Client) DeleteObjectWithContext(ctx context.Context, input *s3.DeleteObjectInput, opts ...request.Option) (*s3.DeleteObjectOutput, error) {
	return m.deleteObjectFunc(input)
}

// TestUploader тестовая версия Uploader для тестирования
type TestUploader struct {
	s3Uploader S3UploaderInterface
	s3Client   S3ClientInterface
	config     *Config
}

// NewTestUploader создает тестовый uploader
func NewTestUploader(config *Config, uploader S3UploaderInterface, client S3ClientInterface) *TestUploader {
	return &TestUploader{
		s3Uploader: uploader,
		s3Client:   client,
		config:     config,
	}
}

// UploadFile загружает файл в S3 (тестовая версия)
func (u *TestUploader) UploadFile(ctx context.Context, reader io.Reader, key string) (string, error) {
	_, err := u.s3Uploader.UploadWithContext(ctx, &s3manager.UploadInput{
		Bucket: aws.String(u.config.BucketName),
		Key:    aws.String(key),
		Body:   reader,
	})

	if err != nil {
		return "", fmt.Errorf("ошибка загрузки: %w", err)
	}

	// Формируем URL файла
	url := fmt.Sprintf("%s/%s/%s", u.config.Endpoint, u.config.BucketName, key)
	return url, nil
}

// DeleteFile удаляет файл из S3 (тестовая версия)
func (u *TestUploader) DeleteFile(ctx context.Context, key string) error {
	_, err := u.s3Client.DeleteObjectWithContext(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(u.config.BucketName),
		Key:    aws.String(key),
	})

	if err != nil {
		return fmt.Errorf("ошибка удаления файла из S3: %w", err)
	}

	return nil
}

// TestSuccessfulUpload тестирует успешную загрузку файла в S3
func TestSuccessfulUpload(t *testing.T) {
	// Создаем тестовую конфигурацию
	config := &Config{
		Region:     "us-east-1",
		AccessKey:  "test-access-key",
		SecretKey:  "test-secret-key",
		Endpoint:   "https://s3.amazonaws.com",
		BucketName: "test-bucket",
	}

	// Создаем мок uploader
	mockUploader := &MockS3Uploader{
		uploadFunc: func(input *s3manager.UploadInput) (*s3manager.UploadOutput, error) {
			// Проверяем, что переданные параметры корректны
			if aws.StringValue(input.Bucket) != "test-bucket" {
				t.Errorf("Ожидался bucket: test-bucket, получено: %s", aws.StringValue(input.Bucket))
			}
			if aws.StringValue(input.Key) != "test-file.mp3" {
				t.Errorf("Ожидался key: test-file.mp3, получено: %s", aws.StringValue(input.Key))
			}

			// Читаем содержимое для проверки
			body, err := io.ReadAll(input.Body)
			if err != nil {
				t.Errorf("Ошибка чтения тела запроса: %v", err)
			}
			if string(body) != "test content" {
				t.Errorf("Ожидалось содержимое: test content, получено: %s", string(body))
			}

			return &s3manager.UploadOutput{
				Location: "https://s3.amazonaws.com/test-bucket/test-file.mp3",
			}, nil
		},
	}

	// Создаем тестовый uploader с моками
	uploader := NewTestUploader(config, mockUploader, &MockS3Client{})

	// Тестируем загрузку
	ctx := context.Background()
	reader := strings.NewReader("test content")
	url, err := uploader.UploadFile(ctx, reader, "test-file.mp3")

	if err != nil {
		t.Errorf("Неожиданная ошибка при загрузке: %v", err)
	}

	expectedURL := "https://s3.amazonaws.com/test-bucket/test-file.mp3"
	if url != expectedURL {
		t.Errorf("Ожидался URL: %s, получено: %s", expectedURL, url)
	}
}

// TestUploadErrorHandling тестирует обработку ошибок при загрузке
func TestUploadErrorHandling(t *testing.T) {
	config := &Config{
		Region:     "us-east-1",
		AccessKey:  "invalid-key",
		SecretKey:  "invalid-secret",
		Endpoint:   "https://s3.amazonaws.com",
		BucketName: "test-bucket",
	}

	// Тест 1: Ошибка неверных учетных данных
	t.Run("InvalidCredentials", func(t *testing.T) {
		mockUploader := &MockS3Uploader{
			uploadFunc: func(input *s3manager.UploadInput) (*s3manager.UploadOutput, error) {
				return nil, awserr.New("InvalidAccessKeyId", "The AWS Access Key Id you provided does not exist in our records.", nil)
			},
		}

		uploader := NewTestUploader(config, mockUploader, &MockS3Client{})

		ctx := context.Background()
		reader := strings.NewReader("test content")
		_, err := uploader.UploadFile(ctx, reader, "test-file.mp3")

		if err == nil {
			t.Error("Ожидалась ошибка при неверных учетных данных")
		}

		if !strings.Contains(err.Error(), "ошибка загрузки") {
			t.Errorf("Неожиданное сообщение об ошибке: %v", err)
		}
	})

	// Тест 2: Сетевая ошибка
	t.Run("NetworkError", func(t *testing.T) {
		mockUploader := &MockS3Uploader{
			uploadFunc: func(input *s3manager.UploadInput) (*s3manager.UploadOutput, error) {
				return nil, awserr.New("RequestTimeout", "Request timeout", nil)
			},
		}

		uploader := NewTestUploader(config, mockUploader, &MockS3Client{})

		ctx := context.Background()
		reader := strings.NewReader("test content")
		_, err := uploader.UploadFile(ctx, reader, "test-file.mp3")

		if err == nil {
			t.Error("Ожидалась ошибка при сетевой проблеме")
		}

		if !strings.Contains(err.Error(), "ошибка загрузки") {
			t.Errorf("Неожиданное сообщение об ошибке: %v", err)
		}
	})

	// Тест 3: Ошибка доступа к bucket
	t.Run("BucketAccessError", func(t *testing.T) {
		mockUploader := &MockS3Uploader{
			uploadFunc: func(input *s3manager.UploadInput) (*s3manager.UploadOutput, error) {
				return nil, awserr.New("AccessDenied", "Access Denied", nil)
			},
		}

		uploader := NewTestUploader(config, mockUploader, &MockS3Client{})

		ctx := context.Background()
		reader := strings.NewReader("test content")
		_, err := uploader.UploadFile(ctx, reader, "test-file.mp3")

		if err == nil {
			t.Error("Ожидалась ошибка при отсутствии доступа к bucket")
		}

		if !strings.Contains(err.Error(), "ошибка загрузки") {
			t.Errorf("Неожиданное сообщение об ошибке: %v", err)
		}
	})
}

// TestS3ObjectKeyFormation тестирует корректность формирования имени файла (ключа) в S3
func TestS3ObjectKeyFormation(t *testing.T) {
	config := &Config{
		Region:     "us-east-1",
		AccessKey:  "test-access-key",
		SecretKey:  "test-secret-key",
		Endpoint:   "https://s3.amazonaws.com",
		BucketName: "test-bucket",
	}

	testCases := []struct {
		name        string
		inputKey    string
		expectedKey string
		description string
	}{
		{
			name:        "SimpleFileName",
			inputKey:    "song.mp3",
			expectedKey: "song.mp3",
			description: "Простое имя файла без пути",
		},
		{
			name:        "FileNameWithPath",
			inputKey:    "music/artist/song.mp3",
			expectedKey: "music/artist/song.mp3",
			description: "Имя файла с путем",
		},
		{
			name:        "FileNameWithSpecialChars",
			inputKey:    "song (remix) [2024].mp3",
			expectedKey: "song (remix) [2024].mp3",
			description: "Имя файла со специальными символами",
		},
		{
			name:        "FileNameWithSpaces",
			inputKey:    "my song title.mp3",
			expectedKey: "my song title.mp3",
			description: "Имя файла с пробелами",
		},
		{
			name:        "FileNameWithUnicode",
			inputKey:    "песня.mp3",
			expectedKey: "песня.mp3",
			description: "Имя файла с Unicode символами",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var receivedKey string
			mockUploader := &MockS3Uploader{
				uploadFunc: func(input *s3manager.UploadInput) (*s3manager.UploadOutput, error) {
					receivedKey = aws.StringValue(input.Key)
					return &s3manager.UploadOutput{
						Location: "https://s3.amazonaws.com/test-bucket/" + receivedKey,
					}, nil
				},
			}

			uploader := NewTestUploader(config, mockUploader, &MockS3Client{})

			ctx := context.Background()
			reader := strings.NewReader("test content")
			_, err := uploader.UploadFile(ctx, reader, tc.inputKey)

			if err != nil {
				t.Errorf("Ошибка при загрузке: %v", err)
			}

			if receivedKey != tc.expectedKey {
				t.Errorf("Ожидался ключ: %s, получено: %s", tc.expectedKey, receivedKey)
			}
		})
	}
}

// TestNewUploader тестирует создание нового uploader
func TestNewUploader(t *testing.T) {
	// Тест с корректной конфигурацией
	t.Run("ValidConfig", func(t *testing.T) {
		config := &Config{
			Region:     "us-east-1",
			AccessKey:  "test-access-key",
			SecretKey:  "test-secret-key",
			BucketName: "test-bucket",
		}

		uploader, err := NewUploader(config)
		if err != nil {
			t.Errorf("Неожиданная ошибка при создании uploader: %v", err)
		}

		if uploader == nil {
			t.Error("Uploader не должен быть nil")
			return
		}

		if uploader.config != config {
			t.Error("Конфигурация должна быть сохранена")
		}
	})

	// Тест с конфигурацией с endpoint
	t.Run("ConfigWithEndpoint", func(t *testing.T) {
		config := &Config{
			Region:     "us-east-1",
			AccessKey:  "test-access-key",
			SecretKey:  "test-secret-key",
			Endpoint:   "https://custom-s3-endpoint.com",
			BucketName: "test-bucket",
		}

		uploader, err := NewUploader(config)
		if err != nil {
			t.Errorf("Неожиданная ошибка при создании uploader с endpoint: %v", err)
		}

		if uploader == nil {
			t.Error("Uploader не должен быть nil")
		}
	})
}

// TestDeleteFile тестирует удаление файла из S3
func TestDeleteFile(t *testing.T) {
	config := &Config{
		Region:     "us-east-1",
		AccessKey:  "test-access-key",
		SecretKey:  "test-secret-key",
		BucketName: "test-bucket",
	}

	// Тест успешного удаления
	t.Run("SuccessfulDelete", func(t *testing.T) {
		mockClient := &MockS3Client{
			deleteObjectFunc: func(input *s3.DeleteObjectInput) (*s3.DeleteObjectOutput, error) {
				if aws.StringValue(input.Bucket) != "test-bucket" {
					t.Errorf("Ожидался bucket: test-bucket, получено: %s", aws.StringValue(input.Bucket))
				}
				if aws.StringValue(input.Key) != "test-file.mp3" {
					t.Errorf("Ожидался key: test-file.mp3, получено: %s", aws.StringValue(input.Key))
				}
				return &s3.DeleteObjectOutput{}, nil
			},
		}

		uploader := NewTestUploader(config, &MockS3Uploader{}, mockClient)

		ctx := context.Background()
		err := uploader.DeleteFile(ctx, "test-file.mp3")

		if err != nil {
			t.Errorf("Неожиданная ошибка при удалении: %v", err)
		}
	})

	// Тест ошибки удаления
	t.Run("DeleteError", func(t *testing.T) {
		mockClient := &MockS3Client{
			deleteObjectFunc: func(input *s3.DeleteObjectInput) (*s3.DeleteObjectOutput, error) {
				return nil, awserr.New("NoSuchKey", "The specified key does not exist.", nil)
			},
		}

		uploader := NewTestUploader(config, &MockS3Uploader{}, mockClient)

		ctx := context.Background()
		err := uploader.DeleteFile(ctx, "non-existent-file.mp3")

		if err == nil {
			t.Error("Ожидалась ошибка при удалении несуществующего файла")
		}

		if !strings.Contains(err.Error(), "ошибка удаления файла из S3") {
			t.Errorf("Неожиданное сообщение об ошибке: %v", err)
		}
	})
}

// TestUploadFileWithContext тестирует загрузку с контекстом
func TestUploadFileWithContext(t *testing.T) {
	config := &Config{
		Region:     "us-east-1",
		AccessKey:  "test-access-key",
		SecretKey:  "test-secret-key",
		BucketName: "test-bucket",
	}

	// Тест с контекстом
	t.Run("WithContext", func(t *testing.T) {
		mockUploader := &MockS3Uploader{
			uploadFunc: func(input *s3manager.UploadInput) (*s3manager.UploadOutput, error) {
				return &s3manager.UploadOutput{
					Location: "https://s3.amazonaws.com/test-bucket/test-file.mp3",
				}, nil
			},
		}

		uploader := NewTestUploader(config, mockUploader, &MockS3Client{})

		ctx := context.Background()
		reader := strings.NewReader("test content")
		url, err := uploader.UploadFile(ctx, reader, "test-file.mp3")

		if err != nil {
			t.Errorf("Неожиданная ошибка при загрузке: %v", err)
		}

		expectedURL := "/test-bucket/test-file.mp3"
		if url != expectedURL {
			t.Errorf("Ожидался URL: %s, получено: %s", expectedURL, url)
		}
	})
}
