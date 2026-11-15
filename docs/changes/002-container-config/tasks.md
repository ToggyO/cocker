# Tasks: Container Configuration and Domain Model

## Task 1: Domain types and invariants — completed

Создать чистые типы `ContainerSpec`, `ResourceSpec`, `CPUQuota`, `NamespaceSpec`, `EnvVar` и `IOSpec`, а также domain validation errors.

Связанные спецификации: `docs/architecture/container-spec.md`, `docs/architecture/namespace-defaults.md`, `docs/architecture/resource-values.md`.

Приёмка: domain package не импортирует platform или config; структура отражает согласованный `ContainerSpec`; tests проверяют обязательные поля и namespace invariants.

Проверка: `go test ./internal/domain`, `go test ./...` и `go vet ./...` — успешно.

## Task 2: Resource value parsers — completed

Преобразовать memory, CPU и PIDs из пользовательского формата в числовые domain values.

Приёмка: memory принимает bytes, `KiB`, `MiB`, `GiB`; CPU принимает millicores и создаёт fixed period `100000`; PIDs принимает положительное целое; overflow, zero, negative и неизвестные units отклоняются; floating point не используется.

Проверка: valid, invalid и boundary tests; `go test ./...`.

Результат приёмки: исправлены потеря последней цифры memory без суффикса и промежуточное переполнение CPU quota. Регрессионные тесты проверяют многозначные bytes, границы uint64, максимальную допустимую CPU quota и PIDs limit. `gofmt -l internal/config`, `go test ./...`, `go vet ./...` и `go build -o /tmp/cocker-task2 ./cmd/cocker` — успешно. Privileged integration tests не требуются: задача содержит только parsing без kernel operations. Контракт ошибок пока сохраняет `strconv.ErrRange` через `ParseError` для memory/CPU и использует `ParsingErrorOverflow` для арифметического overflow и PIDs.

## Task 3: JSON decoding and normalization — completed

Загрузить JSON configuration, применить defaults и получить нормализованный `ContainerSpec`.

### Цель

Реализовать config-layer decode и normalization без запуска процессов, создания
kernel resources или зависимости domain package от `encoding/json`.

### JSON-контракт

Внешняя JSON-модель отделена от domain-модели. Для environment используется
массив объектов, чтобы сохранить порядок и обнаруживать duplicate names:

```json
{
  "rootfs": "./rootfs",
  "command": ["/bin/sh", "-c", "echo hello"],
  "hostname": "cocker-dev",
  "workingDir": "/",
  "env": [
    {"name": "APP_ENV", "value": "dev"}
  ],
  "resources": {
    "memory": "256MiB",
    "cpus": "500m",
    "pids": 64
  }
}
```

JSON decoder должен использовать strict unknown-field policy. Resource strings
преобразуются при normalization через `ParseMemoryBytes` и `ParseCPUQuota`.
JSON `pids` декодируется в `*uint64` и копируется в domain resource limit;
ноль отклоняется последующей domain validation. `ParsePidsLimit` остаётся
для строкового ввода CLI.

Decoder принимает ровно один объект, отклоняет пустой ввод и верхнеуровневый
`null`. Вложенные optional `null` означают отсутствие значения. Указатели и
nil slices отличают отсутствие поля от явно пустого значения. Пустой объект
допустим до merge и validation. `namespaces` и `io` не входят в JSON schema.

### Normalization

- непустой relative `rootfs` преобразуется через `filepath.Abs` относительно
  текущего каталога CLI; пустой остаётся пустым, symlinks не разрешаются;
- отсутствующий или пустой `workingDir` получает `/`; непустой очищается через
  `filepath.Clean`, относительный путь остаётся относительным;
- отсутствующие namespaces получают `domain.DefaultNamespaceSpec()`;
- `IO` получает `AttachStdin=true`, `AttachStdout=true`, `AttachStderr=true`:
  foreground run использует потоки вызывающего CLI без выделения PTY;
- `IO` отсутствует в `Input` и JSON; пользовательских IO overrides в MVP нет;
- отсутствующие resource limits остаются `nil`;
- command и arguments сохраняются без shell parsing;
- hostname остаётся пустым, если не задан;
- normalization не вызывает validation; вызывающий код выполняет
  `ContainerSpec.Validate()` после merge и normalization при интеграции Task 5;
- проверка существования и типа rootfs остаётся runtime/platform responsibility.

### Предлагаемые компоненты

Реализованные функции:

```go
func DecodeJSON(r io.Reader) (Input, error)
func Normalize(input Input) (domain.ContainerSpec, error)
```

Normalization возвращает первую ошибку преобразования как `FieldError`,
сохраняя причину через `Unwrap`. Decoder добавляет контекст `decode config`
через `%w` и возвращает пустой `Input` при ошибке.

### Тестовые сценарии

Table-driven tests должны покрывать полный JSON, минимальный JSON, defaults,
relative rootfs, command arguments, resource conversion, unknown fields,
invalid types, invalid resource values, duplicate/invalid environment names,
пустые обязательные поля и trailing JSON values.

Тест normalization должен явно проверять все три `IO.Attach* = true`, в том
числе для минимального ввода. JSON с полем `io` отклоняется как unknown field.

Приёмка: неизвестные JSON-поля отклоняются; relative rootfs нормализуется в absolute path; namespaces получают defaults; command arguments сохраняются без shell parsing; invalid fields дают field-oriented errors.

Проверка: `go test ./...` с table-driven JSON fixtures.

Результат review: реализация и тесты приняты. Покрыты optional values,
strict decoding, trailing input, defaults, paths, resource conversion,
сохранение arguments/env, независимость результата от Input и граница с
domain validation. `go test ./...`, `go vet ./...`, build, gofmt и
`git diff --check` проходят. Privileged integration tests не требуются.

## Task 4: CLI overrides and merge

Представить optional CLI values и объединить их с JSON с приоритетом явно заданных CLI flags.

Приёмка: отсутствующий flag не затирает JSON; явно переданный flag заменяет JSON; command arguments однозначно отделены от options; conflicts отклоняются до runtime.

Проверка: unit tests merge precedence и CLI tests.

## Task 5: Integrate configuration into `run`

Подключить `decode → merge → normalize → validate` к команде `cocker run`.

Приёмка: `run` передаёт runtime только валидированный `ContainerSpec`; invalid configuration не запускает process runner; CLI и JSON дают одинаковую domain model; существующие process runner tests проходят.

Проверка: `gofmt`, `go test ./...`, `go vet ./...`.

## Block completion review

После Task 5 выполнить отдельное milestone review по `docs/development/feature-review.md`, обновить связанные architecture documents и `docs/session-context.md`, затем зафиксировать результат блока.
