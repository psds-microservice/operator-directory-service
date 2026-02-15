# operator-directory-service

Реестр операторов для UI: список, профиль, создание, обновление. REST API на Gin; контракт описан в proto для единообразия с остальными Go-сервисами.

## Proto

- **Источник:** `pkg/operator_directory_service/operator_directory.proto` — сервис `OperatorDirectoryService` (ListOperators, GetOperator, CreateOperator, UpdateOperator) с аннотациями `google.api.http`.
- **Генерация:** `make proto-generate` (локальный protoc + плагины) или `make proto` (сборка образа protoc при наличии `infra/protoc-go.Dockerfile` и генерация). На Windows при отсутствии make: установите protoc, `protoc-gen-go`, `protoc-gen-go-grpc`, `protoc-gen-grpc-gateway` и выполните команду protoc из `Makefile` (цель `proto-generate-local`).
- **Сгенерированный код:** `pkg/gen/operator_directory_service/` (`.pb.go`, `_grpc.pb.go`, `.pb.gw.go`). Сейчас REST по-прежнему обслуживается Gin; при необходимости можно перевести маршруты на grpc-gateway.
- **OpenAPI из proto:** `make proto-openapi` → `api/openapi.json`.

## Запуск

- `make run-dev` — режим разработки.
- Порт по умолчанию: 8095. Health: `/health`, Swagger UI: `/swagger/index.html`.
