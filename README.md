# Cloud File Storage на Go

Подробное описание проекта по [ссылке](https://zhukovsd.github.io/java-backend-learning-course/projects/cloud-file-storage/).

## Конфигурация

Приложение можно запустить как отдельную единицу, либо как один из сервисов docker compose. Для запуска через docker compose перейдите по ссылке.

Env для Dev-стенда: `.env.dev.example` необходимо переименовать в `.env.dev`.

Env для Prod-стенда: `.env.prod.example` необходимо переименовать в`.env.prod`.

## Миграции

В проекте используется база данных `MariaDB`. Необходимо создать базу данных перед миграцией.

Для запуска миграций необходимо установить модуль `goose`:
https://github.com/pressly/goose (Или вручную выполнить sql-запросы из файлов `db/migrations`)

Далее выполнить команду:

`goose up`

## Сборка
Команда для сборки:

`make build`

Или:

`go build -o /cloud_file_storage cmd/main.go`

## Запуск

`./cloud_file_storage --env-file=ENV_PATH`

Используйте флаг `--env-file=.env.dev` или `--env-file=.env.prod` для передачи конфигураций исходя из окружения (dev/prod).

## Swagger
Для генерации документации используется [swaggo/swag](https://github.com/swaggo/swag), необходимо установить библиотеку по инструкции.
Далее выполнить команду, которая отформатирует аннотации и сгенерирует необходимые файлы:

`swag fmt && swag init -g cmd/main.go --dir ./ --parseDependency --parseInternal -q`

После запуска проекта, swagger документация будет доступна по адресу: `/swagger/index.html`
