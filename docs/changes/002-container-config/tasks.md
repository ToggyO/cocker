# Tasks: Container Configuration and Domain Model

## Task 1: Domain types and invariants

Создать чистые типы `ContainerSpec`, `ResourceSpec`, `CPUQuota`, `NamespaceSpec`, `EnvVar` и `IOSpec`, а также domain validation errors.

Связанные спецификации: `docs/architecture/container-spec.md`, `docs/architecture/namespace-defaults.md`, `docs/architecture/resource-values.md`.

Приёмка: domain package не импортирует platform или config; структура отражает согласованный `ContainerSpec`; tests проверяют обязательные поля и namespace invariants.

Проверка: `go test ./internal/domain` и `go vet ./...`.

## Task 2: Resource value parsers

Преобразовать memory, CPU и PIDs из пользовательского формата в числовые domain values.

Приёмка: memory принимает bytes, `KiB`, `MiB`, `GiB`; CPU принимает millicores и создаёт fixed period `100000`; PIDs принимает положительное целое; overflow, zero, negative и неизвестные units отклоняются; floating point не используется.

Проверка: valid, invalid и boundary tests; `go test ./...`.

## Task 3: JSON decoding and normalization

Загрузить JSON configuration, применить defaults и получить нормализованный `ContainerSpec`.

Приёмка: неизвестные JSON-поля отклоняются; relative rootfs нормализуется в absolute path; namespaces получают defaults; command arguments сохраняются без shell parsing; invalid fields дают field-oriented errors.

Проверка: `go test ./...` с table-driven JSON fixtures.

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
