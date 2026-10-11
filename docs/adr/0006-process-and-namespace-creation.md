# ADR-0006: Создание процесса через `os/exec` и внутренний child mode

## Статус

Accepted for MVP

## Контекст

`cocker` должен создать процесс контейнера, настроить для него Linux namespaces, подключить stdio и затем управлять его сигналами и exit status.

В Linux это обычно строится вокруг fork/clone и exec. Go-программа не должна произвольно вызывать fork из многопоточного runtime: после fork до exec безопасен ограниченный набор действий, а состояние Go runtime и goroutines делает ручную схему сложной.

## Краткая справка: fork, clone и exec

`fork` создаёт новый процесс как копию текущего процесса.

`clone` — Linux-механизм создания процесса или потока с настраиваемым разделением ресурсов. Флаги `CLONE_NEWPID`, `CLONE_NEWNS`, `CLONE_NEWUTS` и другие создают новые namespaces.

`execve` заменяет программу внутри текущего процесса другой программой, сохраняя PID процесса.

Практически запуск контейнера обычно состоит из создания child process с нужными namespace flags, выполнения setup внутри child и затем `exec` целевой команды.

## Краткая справка: re-exec

При re-exec родитель запускает тот же бинарник ещё раз со специальным внутренним аргументом, например `__cocker_child`.

Родитель выполняет пользовательскую команду:

```text
cocker run ... /bin/sh
```

А child mode получает конфигурацию через аргументы, environment или открытые file descriptors, выполняет setup и затем запускает `/bin/sh`.

Такой подход позволяет разделить внешний CLI и privileged setup, не дублируя бинарники и не вызывая пользовательскую команду напрямую из parent orchestration.

## Решение

Для MVP используем:

1. `os/exec.Cmd` для управления дочерним процессом.
2. Linux-specific `syscall.SysProcAttr` для параметров создания процесса и namespace flags.
3. Внутренний child mode того же бинарника для namespace/rootfs setup.
4. `golang.org/x/sys/unix` для операций, которые не покрываются безопасным уровнем `os/exec` и `SysProcAttr`.

Внешний CLI не запускает пользовательскую команду напрямую. Он запускает child mode, а child mode после setup выполняет пользовательскую команду через `syscall.Exec` или эквивалентную операцию.

## Логическая схема

```text
Parent cocker
    │
    ├── create metadata
    ├── prepare state
    ├── create child via os/exec
    │       └── namespace flags
    │
    ▼
Child cocker (__cocker_child)
    │
    ├── validate inherited configuration
    ├── configure hostname
    ├── configure mounts and rootfs
    ├── configure stdio and environment
    ├── perform privilege/security setup
    └── exec target command
```

## Ответственность parent

Parent отвечает за:

- разбор и validation `ContainerSpec`;
- создание container metadata;
- создание cgroup и применение resource policy в согласованной точке lifecycle;
- запуск child mode;
- передачу stdin/stdout/stderr;
- ожидание завершения;
- обработку stop timeout;
- сохранение exit status;
- cleanup.

Parent не должен выполнять mount и rootfs setup внутри своего собственного namespace.

## Ответственность child

Child отвечает за операции, которые должны выполняться уже в namespace контейнера:

- hostname;
- mount namespace setup;
- rootfs preparation;
- mounts `/proc`, `/dev` и других необходимых pseudo-filesystems;
- рабочая директория;
- environment;
- будущая смена UID/GID и capabilities;
- `exec` целевой команды.

Child должен сообщить parent об ошибке setup до выполнения target command. Канал передачи startup error будет определён в implementation design; предпочтительно использовать отдельный pipe/file descriptor, а не разбор stderr.

## Порядок действий

Предварительный порядок для `run`:

1. Parse и validate configuration.
2. Создать metadata в состоянии `created`.
3. Подготовить state directory.
4. Создать cgroup или подготовить resource policy согласно отдельному runtime design.
5. Запустить child mode с namespace flags.
6. Child выполняет mount/rootfs/security setup.
7. Child сообщает об успешной подготовке.
8. Child выполняет `exec` целевой команды.
9. Parent переводит контейнер в `running`.
10. Parent ждёт завершения и сохраняет exit status.
11. Parent выполняет cleanup и переводит контейнер в `stopped`.

Если child сообщает ошибку до `exec`, parent не должен считать контейнер running.

## Почему не ручной fork/clone orchestration

Ручной вызов низкоуровневых process creation APIs из Go усложняет:

- взаимодействие с goroutines и Go runtime;
- передачу ошибок между parent и child;
- настройку close-on-exec descriptors;
- обработку signals;
- ожидание процесса;
- переносимость даже внутри Linux variants.

Мы всё равно изучим fork/clone/exec и соответствующие kernel concepts, но используем стандартный process abstraction там, где она не мешает контролю над Linux behavior.

## Последствия

### Положительные

- parent и child имеют ясные границы;
- `os/exec` предоставляет привычный lifecycle API;
- namespace setup происходит в правильном процессе;
- системные вызовы изолированы в Linux platform layer;
- удобнее тестировать process management отдельно от rootfs setup.

### Отрицательные

- нужно поддерживать внутренний child command;
- нужно аккуратно передавать конфигурацию и startup errors;
- часть поведения `os/exec` зависит от Linux `SysProcAttr`;
- порядок setup и момент перехода в `running` требуют integration tests.

## Требования к безопасности

- child mode не должен быть доступен как полноценная публичная пользовательская команда;
- конфигурация child должна приходить из контролируемого parent или защищённого state path;
- file descriptors должны иметь явную lifecycle policy;
- пользовательские аргументы нельзя склеивать в shell command;
- target command запускается без shell, если shell явно не указан пользователем;
- ошибка setup должна закрывать или очищать созданные ресурсы.

## Альтернативы

### Напрямую запускать target command с `os/exec`

Отклонено: parent тогда должен выполнять setup в неудобной точке или передавать сложную bootstrap logic внешнему скрипту.

### Использовать готовый container runtime library

Отклонено в рамках учебных целей и требования изучить Linux mechanisms.

### Вызывать `unix.Clone` напрямую из Go

Отклонено для первого process layer: ручное управление clone/fork взаимодействует с Go runtime сложнее. Такой API можно рассмотреть в отдельной экспериментальной ветке или после появления конкретного требования.

## Проверка решения

Первая integration test implementation должна проверить:

- parent получает exit status child;
- target command запускается после child setup;
- startup error не переводит контейнер в `running`;
- stdout и stderr не смешиваются без явного решения;
- сигнал stop доходит до управляемого процесса;
- target arguments передаются без shell interpolation.

## Связанные документы

- [`docs/adr/0001-cgroup-v2.md`](0001-cgroup-v2.md)
- [`docs/adr/0002-root-mode-for-mvp.md`](0002-root-mode-for-mvp.md)
- [`docs/specs/container-lifecycle/states.md`](../specs/container-lifecycle/states.md)
