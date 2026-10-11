# ContainerSpec

## Статус документа

- Статус: draft
- Версия: 0.1
- Связанные документы:
  - [`docs/adr/0004-container-configuration-format.md`](../adr/0004-container-configuration-format.md)
  - [`docs/architecture/overview.md`](overview.md)
  - [`docs/specs/container-lifecycle/states.md`](../specs/container-lifecycle/states.md)

## Назначение

`ContainerSpec` — внутренняя нормализованная модель желаемого контейнера.

Она описывает намерение пользователя: какой процесс запустить, какое окружение подготовить и какие ограничения применить. Она не описывает фактическое состояние контейнера после запуска. Фактическое состояние хранится в metadata и lifecycle state.

```text
CLI flags ─┐
           ├── decode → merge → normalize → validate → ContainerSpec
JSON file ─┘
```

## Краткая справка: desired state и observed state

В системах управления часто разделяют:

- desired state — что пользователь хочет получить;
- observed state — что реально создано и работает.

`ContainerSpec` является desired state. Например, он может содержать memory limit `256M`, но не содержит фактическое потребление памяти. Usage и PID относятся к observed state и metadata.

Это разделение позволяет повторно проверить желаемую конфигурацию и отдельно диагностировать расхождения с Linux kernel.

## Модель MVP

Предварительная Go-модель:

```go
type ContainerSpec struct {
	RootFS     string
	Command    []string
	Hostname   string
	WorkingDir string
	Env        []EnvVar
	Resources  ResourceSpec
	Namespaces NamespaceSpec
	IO         IOSpec
}

type EnvVar struct {
	Name  string
	Value string
}

type ResourceSpec struct {
	MemoryBytes *uint64
	CPUQuota    *CPUQuota
	PidsLimit   *uint64
}

type CPUQuota struct {
	Quota  uint64
	Period uint64
}

type NamespaceSpec struct {
	PID     bool
	Mount   bool
	UTS     bool
	IPC     bool
	Network bool
}

type IOSpec struct {
	AttachStdin  bool
	AttachStdout bool
	AttachStderr bool
}
```

Имена и конкретные типы могут измениться во время реализации. Важны границы ответственности и отсутствие необязательной привязки к JSON.

## Обязательные поля

### `RootFS`

Путь к подготовленному root filesystem. До запуска runtime проверяет, что путь существует, является каталогом и соответствует ограничениям security policy.

Путь относится к host filesystem до выполнения `pivot_root`. После смены root child работает с тем же ресурсом через новое mount tree.

### `Command`

Непустой список аргументов. Первый элемент — исполняемый файл или путь к нему, остальные элементы — его аргументы.

Команда запускается напрямую, без shell. Если пользователь хочет shell expansion, он явно указывает `/bin/sh -c ...`.

## Необязательные поля и defaults

### `Hostname`

Если не задан, hostname контейнера не меняется или получает значение, определённое runtime policy. Точное правило должно быть выбрано до реализации UTS namespace.

### `WorkingDir`

Если не задан, используется `/`. Путь разрешается внутри rootfs после `pivot_root`.

### `Env`

Представляет переменные окружения как список пар имя-значение, чтобы сохранить порядок и позволить validation duplicate names. Перед `exec` runtime преобразует список в формат `KEY=value`.

### `Resources`

Неуказанный лимит означает отсутствие явного лимита со стороны `cocker`, но это не гарантирует отсутствие лимита со стороны host, systemd или parent cgroup.

### `Namespaces`

Для MVP набор namespaces может иметь безопасные defaults, определённые runtime policy. Пользовательская возможность отключать отдельные namespaces должна быть ограничена, потому что она меняет isolation guarantees.

### `IO`

Для обычного foreground `run` defaults должны подключать stdout и stderr к CLI. Поведение stdin и detached mode будет описано в CLI specification.

## Нормализация

До передачи в application layer конфигурация должна быть нормализована:

- относительный rootfs path превращён в проверенный абсолютный путь;
- memory value преобразован в bytes;
- CPU quota приведён к явным quota и period;
- environment names проверены и дубликаты обработаны;
- отсутствующие значения получили defaults;
- paths очищены без потери смысла;
- command arguments сохранены без shell parsing.

Нормализация не должна создавать Linux resources или менять состояние хоста.

## Валидация

Validation выполняется до создания metadata со статусом `starting` и до запуска child.

Минимальные правила:

- `RootFS` не пустой;
- `Command` не пустой;
- `MemoryBytes`, если задан, больше нуля;
- `PidsLimit`, если задан, больше нуля;
- CPU quota и period больше нуля;
- `WorkingDir` — абсолютный путь внутри контейнера;
- имена environment variables соответствуют Linux process environment rules;
- unsupported namespace combinations отклоняются;
- один параметр не задаётся конфликтующими способами.

Ошибки должны указывать поле, полученное значение и правило, которое нарушено.

## ContainerSpec и JSON

JSON является внешним форматом и может использовать пользовательские представления:

```json
{
  "rootfs": "./rootfs",
  "command": ["/bin/sh"],
  "resources": {
    "memory": "256M",
    "pids": 64
  }
}
```

После decoding строка `"256M"` не должна распространяться в runtime. Runtime получает числовой тип `MemoryBytes`.

JSON tags и strict unknown-field policy будут определены в CLI/config specification.

## Что не входит в ContainerSpec MVP

Не включаем пока:

- image reference;
- image layers;
- registry credentials;
- volumes и bind mounts;
- network interfaces и port mappings;
- capabilities;
- seccomp profile;
- restart policy;
- health checks;
- secrets;
- logging driver.

Будущие поля должны добавляться отдельными блоками, например `NetworkSpec`, `MountSpec` и `SecuritySpec`, чтобы не превращать структуру в плоский список несвязанных параметров.

## Граница с metadata

`ContainerSpec` хранит желаемые параметры:

```text
rootfs, command, env, resource limits, namespace intent
```

Metadata хранит результат и фактические значения:

```text
container ID, PID, timestamps, lifecycle state,
exit status, error, cgroup path, log paths
```

Обе модели могут быть сериализованы, но они не должны использовать одну структуру только ради удобства.

## Критерии готовности модели

- CLI и JSON имеют единый путь преобразования в `ContainerSpec`.
- Runtime не знает о строковых форматах вроде `256M`.
- Validation происходит до запуска процесса и изменения kernel state.
- Невалидный rootfs и пустая команда дают понятные ошибки.
- Domain model не импортирует CLI parser или `encoding/json`.
- Поля для будущих networking, mounts и security не добавлены преждевременно в MVP.
- Unit tests проверяют defaults, normalization, duplicates и invalid values.

## Открытые решения

Перед реализацией config package нужно отдельно выбрать:

1. Точный формат memory и CPU values.
2. Список namespace defaults.
3. Политику duplicate environment variables.
4. Правило default hostname.
5. Семантику отсутствующего или пустого `WorkingDir`.
6. Точный CLI syntax для отделения options от command arguments.
