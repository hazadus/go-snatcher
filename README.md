# 🔊 go-snatcher

## Ссылки

- [Бакет S3](https://console.yandex.cloud/folders/b1gcjr09b094e4ucdeoq/storage/buckets/snatcher)

## Зависимости

- `github.com/spf13/cobra` - CLI фреймворк
- `github.com/gopxl/beep` - аудио библиотека
- `github.com/dhowden/tag` - чтение метаданных MP3

## Конфигурация

Используется файл конфигурации `~/.snatcher`.

### Для Yandex Cloud Storage:
```yaml
aws_bucket_name: "your-bucket-name"
aws_access_key: "your-access-key"
aws_secret_key: "your-secret-key"
aws_region: "ru-central1"
aws_endpoint: "https://storage.yandexcloud.net"
```
