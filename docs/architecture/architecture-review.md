# Architecture Review

## Статус документа

- Статус: accepted baseline for change planning
- Версия: 0.1
- Дата: 2026-10-11

## Проверенная цель

`cocker` — Linux-only CLI container runtime для запуска локального процесса с namespaces, подготовленным rootfs, cgroup v2 limits, lifecycle state и cleanup.

## Принятые решения

| Область | Решение |
|---|---|
| Platform | Ubuntu 24.04 LTS baseline, фактические kernel features проверяются отдельно |
| Go | minimum Go 1.26 |
| Privileges | MVP от root, подготовка к rootless mode |
| cgroups | только cgroup v2 |
| Configuration | CLI flags + JSON, единый `ContainerSpec` |
| State | `/var/lib/cocker`, configurable `--state-dir` |
| Process | `os/exec`, internal child mode |
| IPC | Unix `socketpair` |
| Namespaces | PID, mount, UTS, IPC, network enabled; user disabled |
| Rootfs | mount namespace + private propagation + `pivot_root` |
| Network | offline namespace с поднятым loopback |
| Signals | SIGTERM, timeout, SIGKILL; cgroup process set |
| Cleanup | reverse order mount ledger, cgroup cleanup |
| Daemon | после MVP, application layer reusable |

## Сквозные invariants

1. Невалидная конфигурация не создаёт kernel resources.
2. `ContainerSpec` не смешивается с observed metadata.
3. Resource limits применяются до выполнения target command.
4. Target command не запускается до `CONTINUE`.
5. `running` фиксируется только после подтверждённого `exec`.
6. Startup failure отличается от non-zero exit уже запущенной команды.
7. Все созданные runtime resources имеют ownership и cleanup path.
8. Parent и child не используют stdout/stderr как control protocol.
9. Linux-specific API изолирован в `platform/linux`.
10. Unit tests не требуют root; kernel behavior проверяется integration tests.

## Согласованный run flow

```text
CLI input
  → decode/merge/normalize/validate
  → create metadata(created)
  → create cgroup and write limits
  → start child with namespace flags
  → child sends READY
  → parent writes PID to cgroup.procs
  → parent sends CONTINUE
  → child configures namespaces, rootfs, loopback and security
  → child sends STARTING_EXEC
  → child execs target command
  → close-on-exec EOF confirms successful exec
  → state running, or stopped for fast exit
  → stop/wait and collect exit result
  → reverse cleanup mounts and cgroup
  → persist final state
```

## Известные границы MVP

- rootless mode не реализован;
- target command запускается как PID 1 без отдельного init process;
- внешняя сеть отсутствует;
- arbitrary bind mounts отсутствуют;
- полный capabilities dropping требует отдельной реализации;
- seccomp не является обязательным механизмом MVP;
- recovery после crash parent ограничен stale-state diagnostics;
- daemon отсутствует;
- image import и overlayfs отсутствуют;
- полноценная OCI compatibility отсутствует.

## Риски

### Privileged Linux operations

Ошибка в mount, cgroup или capability setup может изменить host state. Поэтому реализация и integration tests выполняются в disposable VM.

### Startup race

Ошибки в protocol ordering могут запустить target command до применения limits. Это проверяется отдельными integration tests.

### Cleanup failure

Авария parent может оставить stale cgroup или mount. Persistent recovery ledger не входит в MVP, поэтому diagnostics и cleanup tests обязательны.

### PID 1 behavior

Некоторые программы плохо обрабатывают сигналы или не reaping descendants как PID 1. Это ограничение документируется до появления init process.

### Security overclaim

MVP не является sandbox для недоверенного кода. Product documentation должна сохранять это ограничение до реализации security hardening.

## Проверка архитектурной целостности

- application layer может работать поверх direct CLI и будущего daemon;
- storage не зависит от Linux runtime;
- runtime использует platform abstractions;
- `ContainerSpec` расширяется отдельными blocks для network, mounts и security;
- cgroup, namespace и mount cleanup связаны одним container ID;
- lifecycle states согласуются с process protocol;
- integration tests могут проверять весь run flow в Linux VM.

## Решения, которые можно отложить

Эти темы не блокируют первую change:

- точная JSON schema beyond MVP fields;
- history transitions;
- persistent cleanup ledger;
- cgroup usage metrics;
- default hostname value;
- duplicate environment variable policy;
- final CLI formatting;
- daemon API schema.

Они будут решаться перед соответствующими implementation phases.

## Итог

Архитектура готова к декомпозиции MVP на changes. Первая change не должна создавать namespaces или cgroups. Она должна подготовить Go module, domain types, базовый process runner и проверяемый запуск обычной дочерней команды.
