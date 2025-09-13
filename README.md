# Nize Echo

Цифровой голос, звучащий сквозь шум информационного мира.

## Локальный запуск Hugo сервера

### Установка зависимостей

#### macOS

1. **Установка Hugo** (через Homebrew):
   ```bash
   brew install hugo
   ```

2. **Альтернативные способы установки**:
   - Через MacPorts: `sudo port install hugo`
   - Скачать бинарный файл с [официального сайта](https://github.com/gohugoio/hugo/releases)

#### Linux

1. **Ubuntu/Debian**:
   ```bash
   sudo apt update
   sudo apt install hugo
   ```

2. **CentOS/RHEL/Fedora**:
   ```bash
   # CentOS/RHEL
   sudo yum install hugo
   
   # Fedora
   sudo dnf install hugo
   ```

3. **Arch Linux**:
   ```bash
   sudo pacman -S hugo
   ```

4. **Скачать бинарный файл**:
   ```bash
   wget https://github.com/gohugoio/hugo/releases/download/v0.121.2/hugo_extended_0.121.2_linux-amd64.deb
   sudo dpkg -i hugo_extended_0.121.2_linux-amd64.deb
   ```

### Запуск локального сервера

1. **Клонирование репозитория**:
   ```bash
   git clone <repository-url>
   cd nize-echo
   ```

2. **Инициализация git submodules** (обязательно!):
   ```bash
   git submodule update --init --recursive
   ```

3. **Запуск сервера разработки**:
   ```bash
   hugo server
   ```

4. **Запуск с дополнительными опциями**:
   ```bash
   # С автоматическим обновлением при изменениях
   hugo server --watch
   
   # С привязкой к внешнему IP (для доступа с других устройств)
   hugo server --bind 0.0.0.0
   
   # С отключением кэширования
   hugo server --disableFastRender
   ```

5. **Сборка для продакшена**:
   ```bash
   hugo
   ```

Сайт будет доступен по адресу: `http://localhost:1313`

## Структура контента

### 1. Правила оформления и хранения постов

#### Структура директорий
```
content/
├── en/
│   └── posts/
│       ├── _index.md          # Главная страница постов
│       └── post-slug/         # Папка для каждого поста
│           └── index.md       # Содержимое поста
└── ru/
    └── posts/
        ├── _index.md          # Главная страница постов
        └── post-slug/         # Папка для каждого поста
            └── index.md       # Содержимое поста
```

#### Front Matter для постов
```yaml
---
author: Nikita Zinovev
title: "Заголовок поста"
date: 2025-08-15 09:15:00 +0300
description: "Краткое описание поста для SEO"
---
```

#### Правила именования
- **Slug поста**: используйте kebab-case (например: `spring-di-and-ioc`)
- **Папка**: создавайте отдельную папку для каждого поста с именем slug
- **Файл**: всегда называйте `index.md`

#### Пример структуры поста
```
content/en/posts/my-awesome-post/
└── index.md
```

### 2. Правила оформления и хранения курсов

#### Структура директорий
```
content/
├── en/
│   └── courses/
│       ├── _index.md                    # Главная страница курсов
│       └── course-slug/                 # Папка для каждого курса
│           ├── _index.md                # Описание курса
│           └── lesson-N/                # Папки для уроков
│               └── index.md             # Содержимое урока
└── ru/
    └── courses/
        ├── _index.md                    # Главная страница курсов
        └── course-slug/                 # Папка для каждого курса
            ├── _index.md                # Описание курса
            └── lesson-N/                # Папки для уроков
                └── index.md             # Содержимое урока
```

#### Front Matter для курсов

**Файл курса** (`course-slug/_index.md`):
```yaml
---
title: "Название курса"
description: "Описание курса"
date: 2025-09-12
author: "Nikita Zinovev"
base: true
course:
  level: "Beginner"  # Beginner, Intermediate, Advanced
  lessons: 5         # Количество уроков
---
```

**Файл урока** (`course-slug/lesson-N/index.md`):
```yaml
---
title: "Название урока"
description: "Описание урока"
date: 2025-09-12
author: "Nikita Zinovev"
lesson:
  number: 1                    # Номер урока
  course: "course-slug"        # Slug курса
---
```

#### Правила именования
- **Slug курса**: используйте kebab-case (например: `ai-introductory-course`)
- **Папка курса**: создавайте отдельную папку для каждого курса
- **Уроки**: именуйте папки как `lesson-N` (например: `lesson-1`, `lesson-2`)
- **Файлы**: всегда называйте `index.md`

#### Пример структуры курса
```
content/en/courses/ai-introductory-course/
├── _index.md              # Описание курса
├── lesson-1/
│   └── index.md          # Урок 1
├── lesson-2/
│   └── index.md          # Урок 2
└── lesson-3/
    └── index.md          # Урок 3
```

#### Специальные поля
- **`base: true`** - указывает, что это основная страница курса
- **`course.level`** - уровень сложности курса
- **`course.lessons`** - общее количество уроков в курсе
- **`lesson.number`** - номер урока
- **`lesson.course`** - ссылка на slug курса

## Многоязычность

Сайт поддерживает два языка:
- **Русский** (ru-ru) - основной язык
- **Английский** (en-us) - дополнительный язык

Каждый пост и курс должен быть создан в обеих языковых версиях в соответствующих директориях `content/ru/` и `content/en/`.

## Полезные команды Hugo

```bash
# Создание нового поста
hugo new posts/my-new-post/index.md

# Создание нового курса
hugo new courses/my-new-course/_index.md

# Создание нового урока
hugo new courses/my-new-course/lesson-1/index.md

# Предварительный просмотр с автоматическим обновлением
hugo server -D

# Сборка без черновиков
hugo --minify

# Проверка конфигурации
hugo config
```

## Развертывание

Сайт настроен для автоматического развертывания на Netlify. При пуше в основную ветку происходит автоматическая сборка и деплой.
