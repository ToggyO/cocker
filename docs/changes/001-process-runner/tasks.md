# Tasks: Базовый process runner

## TASK-001: Создать Go module и entrypoint — completed

### Цель

Создать Go module и минимальный binary в `cmd/cocker`.

### Учебная справка

`go.mod` определяет module path и Go language baseline. `main.go` является entrypoint процесса, но должен только собрать зависимости и запустить CLI.

### Критерии приёмки

- module объявляет Go 1.26;
- `go build ./cmd/cocker` завершается успешно;
- запуск binary возвращает контролируемый результат;
- runtime logic не помещена в большой `main`.

## TASK-002: Определить process request и result — completed

### Цель

Определить собственные типы проекта для входных данных и результата процесса.

### Учебная справка

Собственный result type не позволяет деталям `os/exec` распространиться по слоям. Он делает normal exit, signal termination и start failure явными.

### Критерии приёмки

- command и arguments представлены раздельно;
- normal exit и signal termination различимы;
- исходные errors доступны для wrapping и inspection;
- типы не зависят от CLI parser.

## TASK-003: Реализовать process runner — completed

### Цель

Запускать и ожидать процесс через `os/exec`.

### Учебная справка

Runner владеет парой `Start`/`Wait`. Это первый lifecycle invariant runtime: тот, кто запускает процесс, отвечает за ожидание и освобождение process resources.

### Критерии приёмки

- command запускается без shell interpolation;
- context передаётся в process creation;
- после успешного `Start` всегда обрабатывается `Wait`;
- errors содержат operation context;
- stdin/stdout/stderr настраиваются вызывающим кодом.

## TASK-004: Добавить CLI-команду `run` — completed

### Цель

Предоставить process runner через `cocker run <command> [args...]`.

### Учебная справка

CLI является adapter: он переводит пользовательский input в application request и result в output. CLI не должен знать, как внутри работают `fork`, `exec` или wait status.

### Критерии приёмки

- отсутствие command даёт понятную usage error;
- arguments передаются без изменений;
- foreground streams подключены;
- CLI возвращает документированный exit result.

## TASK-005: Добавить unit tests — completed

### Цель

Проверить process runner и CLI без root privileges.

### Учебная справка

Subprocess tests проверяют наблюдаемое поведение. Для этой change они полезнее тестов, которые только подтверждают вызов mock: цель — корректное взаимодействие с реальным process API.

### Критерии приёмки

- success;
- non-zero exit;
- command not found;
- argument preservation;
- cancellation или signal result;
- tests детерминированы и не вызывают shell неявно.

## TASK-006: Обновить project context — completed

### Цель

Зафиксировать статус реализации и evidence для следующей сессии.

### Критерии приёмки

- task statuses обновлены;
- commands и результаты записаны;
- `docs/session-context.md` указывает active change и remaining work;
- ссылки на specification действительны.
