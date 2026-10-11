# Go Package Structure

## Статус документа

- Статус: draft
- Версия: 0.1
- Связанные документы:
  - [`docs/architecture/overview.md`](overview.md)
  - [`docs/architecture/container-spec.md`](container-spec.md)
  - [`docs/architecture/metadata-storage.md`](metadata-storage.md)

## Цель

Структура пакетов должна:

- отделять CLI от use cases;
- отделять domain rules от Linux APIs;
- локализовать `golang.org/x/sys/unix`;
- позволять тестировать config, lifecycle и storage без root;
- подготовить application layer к будущему daemon.

## Предварительная структура

```text
cmd/
    cocker/
        main.go

internal/
    cli/
    app/
    domain/
    config/
    runtime/
    storage/
    platform/
        linux/
            process/
            namespace/
            mount/
            cgroup/
            security/
    observability/

docs/
```

## Краткая справка: `cmd` и `internal`

`cmd/cocker` содержит entrypoint конкретного бинарника. Его задача — собрать зависимости и вызвать application layer.

Каталог `internal` ограничивает импорт пакетов кодом внутри этого module tree. Это полезно для учебного runtime: внутренние детали не объявляются публичным API без необходимости.

## `cmd/cocker`

Содержит `main.go`.

Responsibilities:

- создать root context;
- создать concrete implementations storage, runtime и logger;
- вызвать CLI application;
- перевести итоговую ошибку в process exit code.

`main.go` не должен содержать parsing, namespace setup или lifecycle logic.

## `internal/cli`

Отвечает за внешний интерфейс командной строки:

- parse commands and flags;
- загрузить JSON config;
- показать help;
- вызвать application use case;
- отформатировать result и error.

CLI может зависеть от `config` и `app`, но не должен импортировать `platform/linux`.

## `internal/config`

Отвечает за:

- JSON decoding;
- CLI input mapping;
- merge values;
- defaults;
- parsing resource values;
- normalization;
- validation `ContainerSpec`.

Config package не создаёт процессы, namespaces, cgroups или файлы runtime state.

## `internal/domain`

Содержит чистые типы и правила:

- `ContainerID`;
- `ContainerSpec`;
- resource types;
- `NamespaceSpec`;
- lifecycle statuses;
- `ContainerMetadata`;
- `ContainerState`;
- valid state transitions;
- domain errors.

Domain package не должен импортировать `os/exec`, `golang.org/x/sys/unix`, JSON decoder или CLI parser.

## `internal/app`

Содержит application use cases:

- `RunContainer`;
- `ListContainers`;
- `InspectContainer`;
- `StopContainer`.

Application layer координирует domain, storage и runtime. Он принимает context, проверяет use case rules и управляет lifecycle transitions.

Он не должен знать, какой именно Linux syscall реализует namespace creation.

## `internal/runtime`

Содержит orchestration runtime:

- подготовка запуска;
- parent/child protocol;
- startup synchronization;
- ожидание процесса;
- signal lifecycle;
- cleanup order.

Runtime зависит от abstractions platform и не должен размещать конкретные `unix.Mount` или запись в `memory.max` прямо в orchestration code.

## `internal/storage`

Содержит repository implementation:

- state directory resolution;
- record serialization;
- atomic writes;
- list/get/update/delete;
- schema versioning.

Storage не должен импортировать Linux platform packages.

## `internal/platform/linux`

Содержит OS-specific adapters:

### `process`

- `os/exec` integration;
- `SysProcAttr`;
- wait status;
- descriptor setup.

### `namespace`

- namespace flags;
- namespace detection;
- hostname and namespace-related operations.

### `mount`

- mount propagation;
- `mount`;
- `pivot_root`;
- `umount2`;
- rootfs setup.

### `cgroup`

- cgroup v2 detection;
- controller discovery;
- cgroup creation;
- limits;
- `cgroup.procs`;
- cleanup and usage.

### `security`

- capabilities;
- credential checks;
- future seccomp;
- security policy enforcement.

Все эти пакеты могут импортировать `golang.org/x/sys/unix`. Верхние слои — нет.

## `internal/observability`

Содержит logging и diagnostic primitives, если стандартного `log/slog` станет недостаточно.

Observability не должна менять lifecycle state и не должна быть единственным источником machine-readable ошибок.

## Направление зависимостей

```text
cmd/cocker
    │
    ▼
cli ───────────────► app
 │                    │
 ▼                    ├──► domain
config                ├──► storage abstraction
                     └──► runtime abstraction
                              │
                              ▼
                       platform/linux
```

Практические правила:

- `domain` не зависит от остальных внутренних пакетов;
- `config` зависит от domain types, но не от runtime;
- `app` зависит от interfaces, которые нужны его use cases;
- concrete Linux implementations собираются в `cmd/cocker` или composition package;
- platform packages не импортируют `cli`;
- циклические зависимости запрещены.

## Интерфейсы

Интерфейс создаём на стороне потребителя, когда application layer должен тестироваться без реального Linux runtime.

Примеры возможных consumer-owned interfaces:

```go
type Runtime interface {
	Run(ctx context.Context, spec domain.ContainerSpec) (domain.ContainerResult, error)
	Stop(ctx context.Context, id domain.ContainerID, timeout time.Duration) error
}
```

```go
type ContainerRepository interface {
	Create(ctx context.Context, record domain.ContainerRecord) error
	Get(ctx context.Context, id domain.ContainerID) (domain.ContainerRecord, error)
	List(ctx context.Context) ([]domain.ContainerRecord, error)
	Update(ctx context.Context, record domain.ContainerRecord) error
}
```

Не нужно создавать интерфейсы для каждой маленькой структуры или функции. На первом этапе конкретная JSON storage implementation может быть единственной, а интерфейс появится там, где он улучшает тестирование use case.

## Error boundaries

Нижний слой должен возвращать errors с operation context и wrapped cause. Верхний слой переводит их в domain/application error без потери причины.

CLI решает только, как показать ошибку и какой exit code вернуть. Он не анализирует errno через string matching.

## Test boundaries

- `domain`: обычные unit tests без Linux privileges;
- `config`: table-driven unit tests;
- `storage`: filesystem tests с `t.TempDir()`;
- `app`: tests с fake repository и fake runtime;
- `platform/linux`: Linux integration tests;
- `runtime`: integration tests для process protocol и cleanup.

## Причины выбора

- зависимости направлены внутрь к правилам и use cases;
- Linux-specific code легко найти и проверить;
- application layer можно запускать в unit tests;
- будущий daemon сможет переиспользовать `app`;
- интерфейсы не размножаются без потребности.

## Критерии готовности

- `main.go` остаётся composition root;
- domain не знает о Linux и JSON;
- CLI не вызывает system calls;
- `unix` импортируется только в platform layer;
- storage не зависит от runtime;
- unit и integration tests имеют ясные границы;
- нет циклических imports.
