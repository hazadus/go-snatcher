package s3

import (
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
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
		config:     config,
	}, nil
}

// UploadFile загружает файл в S3
func (u *Uploader) UploadFile(reader io.Reader, key string) (string, error) {
	_, err := u.s3Uploader.Upload(&s3manager.UploadInput{
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
