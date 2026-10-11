# Local Development and CI Workflow

## Статус документа

- Статус: draft
- Версия: 0.1
- Связанный документ: [`docs/architecture/linux-environment.md`](../architecture/linux-environment.md)

## Цель

Каждая implementation task должна иметь воспроизводимую проверку. Быстрые проверки выполняются локально без root, а Linux kernel behavior проверяется отдельным integration workflow.

## Локальный цикл разработки

Перед отправкой задачи на review:

1. прочитать связанные spec и task;
2. реализовать минимальное изменение;
3. отформатировать код;
4. запустить unit tests;
5. запустить static checks;
6. при необходимости запустить Linux integration tests;
7. проверить acceptance criteria;
8. обновить task result и `docs/session-context.md`.

## Базовые команды

Предварительно проект должен поддерживать:

```text
gofmt -w .
go test ./...
go vet ./...
go build ./cmd/cocker
```

После появления race-safe concurrent code:

```text
go test -race ./...
```

Для integration tests:

```text
go test -tags=integration ./...
```

Точная команда может быть обёрнута в `Makefile`, но Makefile не должен скрывать, какие проверки реально выполняются.

## Уровни проверок

### Formatting

`gofmt` является обязательным. Code review не должен тратить время на ручные форматные расхождения.

### Unit tests

Проверяют чистую логику:

- domain transitions;
- resource parsers;
- configuration normalization;
- validation;
- storage repository с `t.TempDir()`;
- application use cases через fake runtime и fake repository.

Unit tests не требуют Linux root privileges и не должны запускать реальный mount или cgroup setup.

### `go vet`

Проверяет распространённые ошибки, которые компилятор не обязан считать ошибками. Его результат является обязательной частью CI.

### Race detector

`go test -race` используется для кода с shared state, lifecycle coordination, storage locks и concurrent event handling.

Он не заменяет Linux integration tests и может требовать больше CPU и памяти.

### Linux integration tests

Проверяют поведение настоящего kernel:

- child process;
- parent/child protocol;
- PID, mount, UTS и network namespaces;
- rootfs и `pivot_root`;
- cgroup v2 limits;
- signals и cleanup.

Они запускаются в Linux disposable environment и могут требовать root.

## Test naming and tags

Unit tests по умолчанию участвуют в `go test ./...`.

Privileged tests получают build tag `integration` или другой явно выбранный tag. Test должен также проверять runtime prerequisites и возвращать понятный skip, если среда не подходит.

Тест нельзя считать пройденным, если он просто пропущен из-за отсутствующего root privilege. В отчёте CI должно быть видно, что integration suite не запускалась.

## GitHub Actions baseline

Предварительный pipeline состоит из jobs:

```text
format-and-static
        │
        ├── unit-tests
        └── build

linux-integration
```

`format-and-static` проверяет gofmt и `go vet`.

`unit-tests` запускает обычные unit tests и, после стабилизации concurrency, race detector.

`build` собирает `cmd/cocker` на Linux.

`linux-integration` использует runner или VM с необходимыми privileges и проверяет реальные kernel mechanisms.

Privileged runner нельзя имитировать обычным unprivileged CI job. Если инфраструктура для него ещё не подготовлена, это фиксируется как незавершённая проверка, а не маскируется зелёным unit pipeline.

## Dependency policy

В CI зависимости должны загружаться через `go mod download` или обычные Go commands после появления `go.mod`.

В репозитории фиксируются:

- `go.mod`;
- `go.sum`;
- версия Go baseline;
- версии внешних модулей.

Нельзя использовать floating shell installer для production checks без фиксации версии.

## Security of CI

Integration tests должны:

- работать в disposable VM или изолированном runner;
- использовать тестовые rootfs;
- не монтировать пользовательские host directories без необходимости;
- удалять cgroups и mounts после теста;
- иметь timeout;
- сохранять diagnostics при failure.

CI runner с root privileges считается доверенной инфраструктурой и не должен запускать непроверенные arbitrary test inputs.

## Code review evidence

Каждая задача сопровождается:

- списком выполненных команд;
- кратким результатом каждой проверки;
- указанием, какие integration tests были недоступны;
- ссылкой на изменённую спецификацию;
- описанием известных ограничений.

Это особенно важно для задач, где unit tests не могут подтвердить kernel behavior.

## Критерии готовности

- локальные unit и static checks документированы;
- privileged integration tests отделены;
- CI явно показывает skipped или executed integration suite;
- build и tests используют Go baseline проекта;
- каждая implementation task содержит evidence;
- cleanup и timeout являются обязательными требованиями integration tests.
