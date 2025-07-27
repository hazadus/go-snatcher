# 🔊 go-snatcher

CLI tool для ведения коллекции DJ-миксов и их прослушивания.

Программа заточена под мои очень специфичные нужды в собирательстве и прослушивании миксов. Её очертания вырисовываются постепенно по мере разработки и использования. Позже, по мере готовности, добавлю подробное описание – что, как, и зачем.

## Ссылки

- [Бакет S3](https://console.yandex.cloud/folders/b1gcjr09b094e4ucdeoq/storage/buckets/snatcher)

## Зависимости

- `github.com/spf13/cobra` - CLI фреймворк
- `github.com/gopxl/beep` - аудио библиотека
- `github.com/dhowden/tag` - чтение метаданных MP3
- `github.com/kkdai/youtube` - загрузка с YouTube

## Конфигурация

Используется файл конфигурации `~/.snatcher`.

### Параметры конфигурации:

| Параметр | Описание | Значение по умолчанию | Обязательный |
|----------|----------|----------------------|--------------|
| `aws_bucket_name` | Имя бакета в Yandex Cloud Storage | - | Да |
| `aws_access_key` | Ключ доступа для S3 API | - | Да |
| `aws_secret_key` | Секретный ключ для S3 API | - | Да |
| `aws_region` | Регион хранилища | - | Да |
| `aws_endpoint` | Эндпоинт Yandex Cloud Storage | - | Да |
| `download_dir` | Папка для загрузки аудиофайлов | `~/Downloads` | Нет |

### Пример конфигурации для Yandex Cloud Storage:
```yaml
aws_bucket_name: "your-bucket-name"
aws_access_key: "your-access-key"
aws_secret_key: "your-secret-key"
aws_region: "ru-central1"
aws_endpoint: "https://storage.yandexcloud.net"
download_dir: "~/Music/snatcher"
```

## Команды

### `snatcher add`

Загружает MP3-файл в облачное хранилище S3 и добавляет его в библиотеку треков.

**Синтаксис:**
```bash
snatcher add [путь к файлу]
```

**Примеры:**
```bash
# Загрузить файл из текущей директории
snatcher add "my_mix.mp3"

# Загрузить файл по абсолютному пути
snatcher add "/Users/username/Music/Deep_House_Mix.mp3"

# Загрузить файл из другой папки
snatcher add "~/Downloads/Techno_Set_2024.mp3"
```

**Что происходит:**
- Проверка существования файла
- Извлечение метаданных (исполнитель, название, альбом, длительность)
- Загрузка в S3 с отображением прогресса
- Сохранение информации о треке в локальной базе данных

---

### `snatcher list`

Отображает список всех треков в библиотеке с подробной информацией.

**Синтаксис:**
```bash
snatcher list
```

**Пример вывода:**
```
📚 Найдено треков: 3

ID   Исполнитель                    Название                       Альбом               Длительность Размер
------------------------------------------------------------------------------------------------------------------------
1    Ben Kaczor                     Inverted Audio In-Store        Various Artists      45:23        64.2 MB
2    Hazadus                        Deep Dark Mix 08.01.12         Personal Collection  67:45        92.8 MB
```

**Отображаемая информация:**
- ID трека (для использования с командой `play`)
- Исполнитель
- Название трека
- Альбом
- Продолжительность
- Размер файла

---

### `snatcher play`

Воспроизводит трек по его ID с интерактивным управлением.

**Синтаксис:**
```bash
snatcher play [ID трека]
```

**Примеры:**
```bash
# Воспроизвести трек с ID 1
snatcher play 1

# Воспроизвести трек с ID 5
snatcher play 5
```

**Управление во время воспроизведения:**
- `[Пробел]` - пауза/возобновление
- `[Ctrl+C]` - остановить и выйти

**Пример вывода:**
```
🎵 Воспроизводится: Ben Kaczor - Inverted Audio In-Store
   Продолжительность: 45:23
   Размер буфера: 64 KB
   Качество: 16-bit, 44100 Hz, 2 каналов

🎮 Управление:
   [Пробел] - пауза/воспроизведение
   [Ctrl+C] - остановить и выйти
```

---

### `snatcher download`

Скачивает аудио из YouTube видео и сохраняет как MP3-файл в папку загрузок.

**Синтаксис:**
```bash
snatcher download [URL YouTube видео]
```

**Примеры:**
```bash
# Скачать аудио из YouTube видео
snatcher download "https://www.youtube.com/watch?v=dQw4w9WgXcQ"

# Сокращенный URL также работает
snatcher download "https://youtu.be/dQw4w9WgXcQ"
```

**Процесс загрузки:**
1. Извлечение ID видео из URL
2. Получение информации о видео (название, автор)
3. Поиск лучшего аудио формата
4. Загрузка аудио потока
5. Сохранение как MP3 в папку `download_dir` из конфигурации

**Пример вывода:**
```
Скачиваем аудио для видео ID: dQw4w9WgXcQ
Название: Rick Astley - Never Gonna Give You Up
Автор: RickAstleyVEVO
Загружаем аудио поток...
Файл сохранен: ~/Music/snatcher/Rick_Astley_Never_Gonna_Give_You_Up.mp3
```

**Примечание:** После загрузки используйте команду `snatcher add` для добавления файла в библиотеку.

---

## Типичный рабочий процесс

1. **Загрузка аудио с YouTube:**
   ```bash
   snatcher download "https://www.youtube.com/watch?v=example"
   ```

2. **Добавление в библиотеку:**
   ```bash
   snatcher add "~/Music/snatcher/downloaded_track.mp3"
   ```

3. **Просмотр библиотеки:**
   ```bash
   snatcher list
   ```

4. **Воспроизведение трека:**
   ```bash
   snatcher play 1
   ```