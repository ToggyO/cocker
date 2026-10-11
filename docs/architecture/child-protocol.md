# Parent/Child Startup Protocol

## Статус документа

- Статус: draft
- Версия: 0.1
- Связанные документы:
  - [`docs/adr/0006-process-and-namespace-creation.md`](../adr/0006-process-and-namespace-creation.md)
  - [`docs/architecture/cgroup-lifecycle.md`](cgroup-lifecycle.md)
  - [`docs/specs/container-lifecycle/states.md`](../specs/container-lifecycle/states.md)

## Назначение

Документ определяет канал и protocol между parent process, который управляет lifecycle контейнера, и child process, который выполняет namespace/rootfs setup и затем запускает target command.

## Краткая справка: `socketpair`

Unix `socketpair` создаёт два связанных file descriptors внутри одного host. Данные, записанные в один descriptor, читаются из другого.

В отличие от обычного pipe, socketpair поддерживает двустороннюю коммуникацию. Это удобно, когда parent должен отправить команду продолжения, а child — сообщить статус или ошибку.

Socketpair не является network socket и не требует TCP, IP или открытого порта.

## Решение

Используем Unix `socketpair` для startup protocol.

Parent создаёт пару descriptors до запуска child, передаёт один descriptor через `cmd.ExtraFiles`, а второй оставляет у себя.

Child получает descriptor с заранее известным номером внутри процесса и закрывает все неиспользуемые descriptors.

## Protocol messages

Для MVP сообщения являются одноразовыми line-oriented frames UTF-8:

```text
READY\n
CONTINUE\n
ABORT\n
ERROR <code> <message>\n
```

Message size ограничивается runtime, чтобы ошибка или повреждённый child не мог заставить parent выделить неограниченную память.

Точный wire format может быть заменён на length-prefixed binary protocol, если line protocol окажется недостаточным. Для локального parent/child взаимодействия MVP line protocol проще диагностировать.

## Состояния протокола

```text
parent creates socketpair
        │
        ▼
child starts in bootstrap mode
        │
        ▼
child completes pre-exec checks
        │
        ├── READY ───────► parent applies cgroup membership
        │                         │
        │                         ├── CONTINUE ─────► child setup/exec
        │                         └── ABORT ─────────► child exits
        │
        └── ERROR ───────► parent performs cleanup
```

## `READY`

Child отправляет `READY`, когда:

- descriptor protocol успешно открыт;
- переданная конфигурация разобрана;
- process identity и необходимые preconditions проверены;
- child готов ждать решения parent.

`READY` не означает, что контейнер уже запущен. Он означает, что parent может безопасно добавить host PID в cgroup и отправить `CONTINUE`.

## `CONTINUE`

Parent отправляет `CONTINUE`, когда:

- cgroup создана;
- resource limits применены;
- child PID добавлен в `cgroup.procs`;
- metadata и lifecycle state позволяют продолжить запуск.

После `CONTINUE` child выполняет mount namespace setup, `pivot_root`, pseudo-filesystem mounts, environment setup и `exec` target command.

## `ABORT`

Parent отправляет `ABORT`, если не смог применить cgroup policy или не может продолжить startup.

Child закрывает resources, не выполняет target command и завершает bootstrap с понятным exit status.

## `ERROR`

Child отправляет `ERROR` при невозможности выполнить bootstrap или setup:

- невалидная inherited configuration;
- невозможность настроить hostname;
- mount error;
- `pivot_root` error;
- невозможность открыть required path;
- невозможность выполнить `exec`.

Сообщение должно содержать stable error code и безопасное диагностическое описание. Нельзя передавать секреты или огромные сырые kernel dumps.

Parent сохраняет структурированную ошибку в metadata и не переводит контейнер в `running`.

## EOF и неожиданные события

Unexpected EOF до `READY` означает startup failure.

Unexpected EOF после `READY` и до `CONTINUE` означает, что child завершился или потерял канал; parent не должен считать контейнер запущенным.

Unexpected EOF после `CONTINUE` обрабатывается отдельно от target process stdout/stderr. Parent должен ориентироваться на wait status процесса, а не только на protocol socket.

Неизвестное сообщение приводит к abort protocol и диагностике.

## File descriptor policy

Parent передаёт child только descriptors, которые нужны для startup и stdio.

Child должен:

- закрыть protocol descriptor после финального сообщения;
- не наследовать случайные descriptors parent;
- не передавать protocol descriptor target command, если это не требуется;
- использовать close-on-exec policy для остальных descriptors.

`ExtraFiles` и descriptor numbering должны быть покрыты integration test, поскольку ошибка в номере descriptor приводит к трудно диагностируемому startup failure.

## Timeout

Parent не ждёт protocol indefinitely.

Для startup используется context timeout. При истечении timeout parent:

1. отправляет `ABORT`, если канал доступен;
2. при необходимости отправляет signal child;
3. ждёт его завершения;
4. выполняет cleanup cgroup и runtime resources;
5. сохраняет timeout как причину failure.

Значение timeout и CLI override будут определены в runtime configuration.

## Security considerations

- child не должен принимать protocol connections от других процессов;
- descriptor передаётся только созданному parent child;
- message length ограничена;
- сообщения не интерпретируются как shell commands;
- fields configuration проходят обычную validation независимо от protocol;
- protocol error не должен приводить к продолжению запуска по умолчанию.

## Причины выбора

- socketpair поддерживает двустороннюю связь;
- не нужен network stack;
- startup status отделён от stdout/stderr;
- parent сохраняет контроль над cgroup ordering;
- protocol легко проверить unit- и integration tests.

## Альтернативы

### Использовать stderr для startup status

Отклонено: stderr является пользовательским потоком и не подходит как надёжный machine-readable protocol.

### Один pipe

Отклонено для MVP: понадобились бы два однонаправленных канала или отдельный механизм для parent-to-child commands.

### Environment variables

Отклонено для control messages: environment подходит для статической configuration, но не для READY/CONTINUE synchronization.

### TCP/Unix listening socket

Отклонено: listening endpoint не нужен для единственного дочернего процесса и расширяет поверхность доступа.

## Критерии готовности

- parent и child используют один явно определённый descriptor;
- `READY` приходит до `CONTINUE`;
- `CONTINUE` отправляется только после cgroup setup;
- `ERROR` не запускает target command;
- startup timeout приводит к cleanup;
- target command не наследует protocol descriptor;
- protocol tests проверяют EOF, unknown message и oversized message.
