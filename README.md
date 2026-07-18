# Shreelance 💼

Платформа-биржа для фрилансеров, где роли заказчика и исполнителя объединены в одном аккаунте. Разделение контекста происходит на уровне интерфейса личного кабинета. Авторизация строго через GitHub OAuth.

## 🛠 Технологический стек
- **Backend:** Go (Golang 1.22+) + Router Chi v5
- **Frontend:** HTMX v2 + gomponents (UI на чистом Go) + Tailwind CSS (через native CLI)
- **Клиентский интерактив:** Alpine.js (подключается через CDN)
- **База данных:** PostgreSQL + GORM
- **Сессии:** alexedwards/scs/v2 + alexedwards/scs/redisstore
- **Кэш/Хранилище сессий/Чат:** Valkey (через go-redis/v9 клиент)
- **Безопасность:** gorilla/csrf (защита от CSRF)
- **Dev-инструменты:** Air (live reload для Go)

---

## 🚀 Быстрый старт

### 1. Переменные окружения
Создайте файл `.env` на основе примера `.env.example`:
```bash
cp .env.example .env
```
Заполните параметры подключения к БД, Valkey, а также `GITHUB_CLIENT_ID` и `GITHUB_CLIENT_SECRET` из настроек OAuth App в GitHub.

### 2. Запуск инфраструктуры
Запустите PostgreSQL и Valkey через Docker Compose:
```bash
make up
# или: docker compose up -d
```

### 3. Запуск веб-сервера в режиме разработки (с Live Reload)
Для сборки Tailwind CSS и запуска Air выполните:
```bash
# В терминале 1 (сборка стилей Tailwind):
make tailwind-watch

# В терминале 2 (запуск Go с Air):
make dev
```
Приложение будет доступно по адресу [http://localhost:8080](http://localhost:8080).

---

## 🔒 Безопасность и архитектура
- **Контроль доступа (IDOR)**: Доступ к деталям заказов, принятию откликов, отмене и чату разграничен строгими проверками сопоставления ID создателей и исполнителей.
- **CSRF**: Все формы используют токены безопасности `csrf_token`, генерируемые `gorilla/csrf`.
- **Чат на Valkey Streams**: При принятии исполнителя открывается чат. Сообщения публикуются в Valkey Stream (`chat:order:<id>`) и доставляются с помощью HTMX-запросов.
