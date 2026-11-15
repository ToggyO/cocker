# Design: Container Configuration and Domain Model

## Границы компонентов

`internal/domain` содержит чистые типы желаемого состояния и правила предметной области. Он не импортирует `encoding/json`, CLI parser, `os/exec` или `golang.org/x/sys/unix`.

`internal/config` отвечает за внешний ввод и преобразование его в domain model:

```text
JSON / CLI input
    → decode
    → merge explicit CLI overrides
    → normalize defaults and paths
    → validate ContainerSpec
```

После этого application/runtime получает только нормализованный `ContainerSpec`.

`DecodeJSON(io.Reader) (Input, error)` выполняет только decoding.
`Normalize(Input) (domain.ContainerSpec, error)` применяется после merge,
возвращает первую ошибку преобразования и не вызывает `Validate`.
Финальную domain validation выполняет вызывающий pipeline в Task 5.
Вложенный optional `null` означает отсутствие значения; верхнеуровневый
`null`, пустой ввод и trailing JSON values запрещены. При ошибке decoding
возвращается пустой `Input`. `pids` в JSON — число `uint64`, положительность
проверяется domain validation. Environment — упорядоченный массив name/value;
дубликаты сохраняются до validation. Namespace policy фиксирована и не
представлена в `Input`. Relative rootfs разрешается от cwd CLI без проверки
существования и разрешения symlinks. WorkingDir очищается, но не превращается
из относительного в абсолютный; пустое значение получает `/`.

## Domain model

Базовая модель включает `ContainerSpec` с полями rootfs, command, hostname, working directory, environment, resources, namespaces и IO policy. Resource values хранятся числовыми типами; pointer-поля отличают отсутствие ограничения от запрещённого `0`.

```go
type ResourceSpec struct {
	MemoryBytes *uint64
	CPUQuota    *CPUQuota
	PidsLimit   *uint64
}

type CPUQuota struct {
	Quota  uint64
	Period uint64
}
```

## Defaults

После normalization `WorkingDir` равен `/`, если не задан; PID, Mount, UTS, IPC и Network включены; User namespace не включён; неуказанные resource limits остаются `nil`; stdio defaults соответствуют foreground CLI run.

Foreground IO policy фиксирована: `IO.AttachStdin`, `IO.AttachStdout` и
`IO.AttachStderr` равны `true`. Runtime получает потоки вызывающего CLI;
выделение PTY этим не подразумевается. `IO` не входит в промежуточный
`config.Input` и JSON schema, CLI overrides для него в этом change отсутствуют.
Значение `IOSpec` формируется на этапе normalization. Detached mode остаётся
за пределами change.

Rootfs path нормализуется в абсолютный path. Проверка существования rootfs остаётся runtime/platform responsibility, если она требует filesystem policy; config validation проверяет непустое значение и корректную форму.

## Внешние форматы

JSON использует пользовательские значения:

```json
{
  "rootfs": "./rootfs",
  "command": ["/bin/sh"],
  "resources": {
    "memory": "256MiB",
    "cpus": "500m",
    "pids": 64
  }
}
```

Memory поддерживает bytes без суффикса и binary units `KiB`, `MiB`, `GiB`. CPU принимает millicores с суффиксом `m`; period фиксирован как `100000` микросекунд. PIDs — положительное целое.

JSON decoder использует strict unknown-field policy. Внешняя JSON-модель отделена от domain model, чтобы формат файла не распространялся в runtime.

## Merge semantics

JSON сначала заполняет базовую конфигурацию. Явно переданные CLI flags заменяют соответствующие JSON values. Отсутствующий CLI flag не должен затирать значение из JSON.

Команда и её аргументы должны быть отделены от опций `cocker`; ambiguous syntax является ошибкой до запуска.

## Validation

Validation проверяет непустые RootFS и Command, положительные limits, отсутствие overflow, абсолютный WorkingDir, корректные environment names, отсутствие duplicate names, namespace invariants и отсутствие конфликтующих способов задания одного параметра.

Ошибки должны содержать имя поля и нарушенное правило. Ошибки применения уже корректной конфигурации к kernel относятся к runtime/platform layer.

## Направление зависимостей

```text
domain
  ▲
  │
config ── CLI adapter / JSON decoder
  ▲
  │
application and CLI
```

`config` может использовать domain types. Domain не знает о config. Ни один пакет этого change не создаёт process, namespace, cgroup или runtime state.

## Проверки

- `gofmt -w` для новых Go-файлов;
- `go test ./...`;
- `go vet ./...`;
- unit tests config и domain без root;
- ручная проверка CLI merge после подключения config к `run`.
