# Design: Базовый process runner

## Контекст

Первая change должна создать исполняемую основу, которую позже можно расширить слоями runtime. Она пока не содержит Linux container setup.

## Учебная справка

### `os/exec`

Стандартный пакет `os/exec` запускает внешние программы и по умолчанию не вызывает shell. `exec.CommandContext` связывает команду с `context.Context`.

### `Cmd`

`exec.Cmd` описывает процесс до запуска и содержит его arguments, environment, working directory и standard streams.

### `Start` и `Wait`

`Start` запускает процесс и возвращает управление после его создания. `Wait` ждёт завершения процесса и освобождает связанные ресурсы.

После успешного `Start` должен быть вызван `Wait`. Иначе lifecycle process resource остаётся неявным и возможна утечка ресурсов.

### Exit status

Процесс может завершиться обычным exit code или быть завершён signal. Go предоставляет детали через `*exec.ExitError` и `ProcessState`, но application layer должен преобразовать их в собственный result type.

### Context cancellation

`CommandContext` связывает отмену context с завершением процесса. Точный signal и поведение потомков будут пересмотрены в change про signals; здесь проверяем только поведение стандартного API.

## Компоненты

```text
cmd/cocker
    ├── creates platform/linux/process.Runner
    └── calls internal/cli
             │
             ▼
    internal/runtime/process.Runner
             ▲
             │
    internal/platform/linux/process
             │
             ▼
           os/exec
```

Первая реализация может использовать меньше пакетов, если границы останутся ясными. Lifecycle process logic не должна находиться в большом `main.go`.

## Предлагаемые interfaces

Абстракция должна описывать потребность application layer, а не копировать каждый параметр `exec.Cmd`:

```go
type ProcessRunner interface {
    Run(ctx context.Context, request ProcessRequest) (ProcessResult, error)
}
```

Точное расположение и форма типов будут уточнены при design review task.

`ProcessRequest` содержит executable, arguments, streams и environment, необходимые этой change. Container-specific поля заранее не добавляются.

`ProcessResult` должен различать normal exit code, signal termination, cancellation, start failure и wait failure.

Для текущей реализации cancellation имеет отдельную семантику: `Result.Canceled` устанавливается в `true`, а наружу возвращается `context.Canceled` или `context.DeadlineExceeded`. Signal, которым был завершён процесс после отмены context, является дополнительной диагностикой и не заменяет context error.

## Семантика аргументов

CLI передаёт command и arguments напрямую в `exec.CommandContext`. Например, `cocker run echo '$HOME'` передаёт `echo` буквальный аргумент `$HOME`. Shell expansion появляется только при явном вызове shell.

## Standard streams

Foreground `run` подключает `os.Stdin`, `os.Stdout` и `os.Stderr` к соответствующим потокам child. Detached mode в эту change не входит.

## Границы ошибок

Linux process layer оборачивает errors с operation context. CLI преобразует их в пользовательское сообщение и exit code.

CLI не должен определять ошибки по совпадению текстов. Используются `errors.Is`, typed results и `ProcessState`.

## Стратегия тестирования

Unit tests используют небольшие детерминированные команды, доступные в test environment, и не зависят от shell behavior.

Обязательные сценарии: успешная команда и exit code `0`, сохранение arguments, ненулевой exit code, command not found, signal или context cancellation, stdout и stderr.

Тесты текущего runner находятся рядом с Linux implementation и используют Linux command `sh`. Переносимые process tests могут появиться позже как отдельный слой, если это потребуется.

## Точки расширения

Process runner должен допускать последующее добавление Linux `SysProcAttr`, namespace flags, inherited protocol descriptors, cgroup PID registration, process group policy и custom rootfs child mode.

Эти возможности не реализуются и не маскируются speculative abstractions в текущей change.
