// Package s3 предоставляет функционал для загрузки файлов в Amazon S3
package s3

import (
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

// Config содержит настройки для S3
type Config struct {
	Region     string
	AccessKey  string
	SecretKey  string
	Endpoint   string
	BucketName string
}

// Uploader обертка для S3 uploader
type Uploader struct {
	s3Uploader *s3manager.Uploader
	s3Client   *s3.S3
	config     *Config
}

// NewUploader создает новый S3 uploader
func NewUploader(config *Config) (*Uploader, error) {
	awsConfig := &aws.Config{
		Region: aws.String(config.Region),
		Credentials: credentials.NewStaticCredentials(
			config.AccessKey,
			config.SecretKey,
			"",
		),
	}

	// Если указан endpoint, добавляем его
	if config.Endpoint != "" {
		awsConfig.Endpoint = aws.String(config.Endpoint)
		awsConfig.S3ForcePathStyle = aws.Bool(true)
	}

	sess, err := session.NewSession(awsConfig)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания AWS сессии: %w", err)
	}

	return &Uploader{
		s3Uploader: s3manager.NewUploader(sess),
		s3Client:   s3.New(sess),
		config:     config,
	}, nil
}

// UploadFile загружает файл в S3
func (u *Uploader) UploadFile(ctx context.Context, reader io.Reader, key string) (string, error) {
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

// DeleteFile удаляет файл из S3
func (u *Uploader) DeleteFile(ctx context.Context, key string) error {
	_, err := u.s3Client.DeleteObjectWithContext(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(u.config.BucketName),
		Key:    aws.String(key),
	})

	if err != nil {
		return fmt.Errorf("ошибка удаления файла из S3: %w", err)
	}

	return nil
}
