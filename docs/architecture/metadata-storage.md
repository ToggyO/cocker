# Metadata and Storage Repository

## Статус документа

- Статус: draft
- Версия: 0.1
- Связанные документы:
  - [`docs/adr/0005-runtime-state-directory.md`](../adr/0005-runtime-state-directory.md)
  - [`docs/specs/container-lifecycle/states.md`](../specs/container-lifecycle/states.md)
  - [`docs/architecture/container-spec.md`](container-spec.md)

## Назначение

Документ разделяет желаемую конфигурацию контейнера, фактическую metadata и lifecycle state, а также определяет предварительный контракт storage repository.

## Три модели данных

### `ContainerSpec`

Описывает desired state:

- rootfs;
- command и arguments;
- environment;
- resource limits;
- namespace policy;
- IO policy.

Эта модель создаётся из CLI или JSON и проходит validation до запуска.

### `ContainerMetadata`

Описывает идентичность и созданные ресурсы:

```go
type ContainerMetadata struct {
	ID            string
	Spec          ContainerSpec
	HostPID       int
	CgroupPath    string
	StateDir      string
	StdoutLogPath string
	StderrLogPath string
	CreatedAt     time.Time
	StartedAt     *time.Time
	FinishedAt    *time.Time
}
```

Metadata отвечает на вопрос: какой контейнер мы создали и где находятся связанные с ним ресурсы.

### `ContainerState`

Описывает lifecycle и результат:

```go
type ContainerState struct {
	Status        Status
	ExitCode      *int
	Signal        *int
	StopReason    string
	LastError     *StoredError
	CleanupError  *StoredError
	UpdatedAt     time.Time
}
```

State отвечает на вопрос: что с контейнером произошло.

Эти модели могут храниться в одном JSON-документе для простоты MVP, но логически остаются разными.

## Краткая справка: repository pattern

Repository — интерфейс доступа к постоянным данным, скрывающий способ хранения.

Application layer вызывает `Create`, `Get`, `UpdateState` и `List`, но не знает, находятся ли данные в JSON-файлах, SQLite или database service.

Это полезно для проекта по двум причинам:

- storage можно тестировать отдельно от runtime;
- позже можно заменить формат хранения без переписывания lifecycle use cases.

Repository не должен скрывать Linux runtime resources. Он хранит их описание и результаты управления ими, но не заменяет cgroup или process manager.

## Предварительный интерфейс

```go
type ContainerRepository interface {
	Create(ctx context.Context, metadata ContainerMetadata) error
	Get(ctx context.Context, id ContainerID) (ContainerRecord, error)
	List(ctx context.Context) ([]ContainerRecord, error)
	Update(ctx context.Context, record ContainerRecord) error
	Delete(ctx context.Context, id ContainerID) error
}

type ContainerRecord struct {
	Metadata ContainerMetadata
	State    ContainerState
}
```

Точный интерфейс может быть уменьшен, если application use cases покажут, что `Update` слишком широкий. Предпочтительно не добавлять методы «на всякий случай».

## Файловая структура

На MVP используется отдельная директория контейнера:

```text
/var/lib/cocker/containers/<id>/
    record.json
    logs/
        stdout.log
        stderr.log
```

`record.json` содержит metadata и state. Отдельные файлы можно добавить позже, если concurrent update или recovery потребуют разделения.

Каталог контейнера создаётся только после validation `ContainerID`. ID не может содержать `/`, `..` или platform-specific path separators.

## Запись record

Обновление record выполняется атомарно:

1. сериализовать полный новый record в память;
2. создать temporary file внутри каталога контейнера;
3. записать bytes;
4. flush и close file;
5. заменить `record.json` через rename;
6. при необходимости синхронизировать directory.

Partial JSON не должен становиться текущим record после сбоя записи.

## Versioning

Record должен иметь schema version:

```json
{
  "schema_version": 1,
  "metadata": {},
  "state": {}
}
```

При изменении структуры storage увеличиваем version или добавляем migration. Неизвестная версия не должна молча интерпретироваться как текущая.

## Concurrency

MVP начинается с предположения, что один `cocker` управляет конкретным контейнером в один момент времени.

Storage должен защищать операции записи внутри процесса. Межпроцессная блокировка и daemon concurrency будут определены при проектировании `cockerd`.

Повторный `stop` должен быть безопасным с точки зрения repository и lifecycle state. Нельзя допускать, чтобы два update затёрли более новое состояние старым record без явной policy.

## Recovery

При чтении record возможны ситуации:

- контейнерная директория отсутствует;
- record отсутствует;
- JSON повреждён;
- schema version неизвестна;
- state указывает на PID, которого нет;
- cgroup path отсутствует;
- metadata ссылается на rootfs, которого больше нет.

В MVP:

- повреждённый record возвращает явную ошибку;
- отсутствующий PID диагностируется как stale runtime state;
- runtime не запускает процесс на основании неполного record;
- repair и автоматическое восстановление будут отдельной change.

## Lifecycle и storage

Предварительные записи:

```text
create record:       created
start preparation:   starting
child ready/exec:     running
stop requested:      stopping
process exited:      stopped
startup failure:     failed
```

Сначала обновляется state в памяти, затем новый record атомарно сохраняется. Если запись состояния не удалась, application layer должен остановить дальнейшее изменение ресурсов и сохранить диагностическую ошибку доступным способом.

## List и Inspect

### `list`

Repository перечисляет директории `containers`, пытается прочитать каждый record и возвращает отсортированный результат. Порядок должен быть определён, например по `CreatedAt`, а не зависеть от порядка filesystem enumeration.

Повреждённая запись не должна молча исчезать из списка. Она представляется как diagnostic entry или приводит к явной ошибке согласно CLI specification.

### `inspect`

`inspect` читает один record по ID. Он не должен автоматически запускать cleanup или менять state, если это не будет отдельной recovery command.

## Причины выбора

- JSON-файл прост для чтения и диагностики;
- repository отделяет use cases от filesystem details;
- schema version оставляет путь к миграциям;
- атомарная запись снижает риск partial state;
- раздельные desired и observed models предотвращают путаницу.

## Критерии готовности

- `ContainerSpec`, metadata и state представлены разными типами;
- repository interface не содержит Linux system calls;
- record имеет schema version;
- запись выполняется атомарно;
- corrupted и stale state различаются;
- list имеет детерминированный порядок;
- storage tests используют `t.TempDir()`;
- lifecycle tests проверяют записи для успешного и ошибочного запуска.

## Открытые вопросы

- хранить ли `ContainerSpec` внутри record полностью или только ссылку на config;
- нужен ли отдельный history переходов состояний;
- должна ли metadata хранить host PID после `stopped`;
- какая policy для corrupted record в `list`;
- когда понадобится межпроцессная lock.
