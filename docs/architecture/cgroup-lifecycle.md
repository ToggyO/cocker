# Cgroup Lifecycle

## Статус документа

- Статус: draft
- Версия: 0.1
- Связанные документы:
  - [`docs/adr/0001-cgroup-v2.md`](../adr/0001-cgroup-v2.md)
  - [`docs/specs/container-lifecycle/states.md`](../specs/container-lifecycle/states.md)
  - [`docs/architecture/namespace-defaults.md`](namespace-defaults.md)

## Назначение

Документ определяет порядок создания cgroup v2, применения resource limits, добавления процесса и cleanup.

## Краткая справка: cgroup tree

Cgroup v2 организует процессы в едином дереве групп. Каждая cgroup представлена каталогом в cgroup filesystem.

В каталоге находятся control files, например:

```text
memory.max
memory.current
cpu.max
pids.max
cgroup.procs
```

Запись значения в control file передаёт настройку Linux kernel. `cgroup.procs` содержит PID процессов, принадлежащих группе, и позволяет переместить процесс записью его PID.

## Краткая справка: почему важен порядок

Если сначала запустить пользовательскую команду, а потом настроить cgroup, процесс успеет выполнить работу без ограничений. Это создаёт race condition между запуском процесса и применением policy.

Race condition — ситуация, в которой результат зависит от того, какая из конкурентных операций произойдёт первой.

Для runtime нужно создать controlled startup point: child должен ждать разрешения parent перед выполнением target command.

## Модель cgroup

Для каждого контейнера создаётся отдельная cgroup:

```text
<cgroup-root>/cocker/<container-id>
```

Точный parent path зависит от host configuration и delegation policy. Runtime не должен предполагать, что `/sys/fs/cgroup/cocker` уже существует без проверки или controlled setup.

Container ID используется как имя дочерней cgroup, но должен пройти validation и не содержать path separators.

## Предварительный lifecycle

```text
create cgroup directory
        │
        ▼
write resource limits
        │
        ▼
start child in startup-wait mode
        │
        ▼
write child host PID to cgroup.procs
        │
        ▼
send continue signal through startup pipe
        │
        ▼
child performs setup and execs target
        │
        ▼
wait for child
        │
        ▼
read optional usage
        │
        ▼
remove cgroup
```

## Детальный порядок

### 1. Проверить cgroup v2

Runtime проверяет filesystem type, наличие unified hierarchy и доступность required controllers.

Если environment не соответствует ADR-0001, запуск завершается до создания контейнера или переводится в `failed` согласно lifecycle policy.

### 2. Создать cgroup

Создаётся каталог, имя которого связано с container ID. Операция должна быть безопасной при повторном cleanup и не должна использовать непроверенный пользовательский путь.

### 3. Применить limits

До запуска child runtime записывает:

- memory limit в `memory.max`;
- CPU quota и period в `cpu.max`;
- process limit в `pids.max`.

Если запись одного из limits не удалась, cgroup удаляется, а процесс ещё не создаётся.

### 4. Запустить child в startup-wait mode

Parent запускает внутренний child mode через `os/exec`. Child получает namespaces и file descriptor startup pipe.

Child не выполняет rootfs setup и target command до получения разрешения или до обнаружения ошибки. Конкретное место ожидания должно быть выбрано так, чтобы PID уже существовал и его можно было добавить в cgroup, а пользовательская команда ещё не выполнялась.

### 5. Добавить child в `cgroup.procs`

Parent получает host PID созданного child и записывает его в `cgroup.procs`.

После успешной записи процессы child и его будущие потомки контролируются cgroup policy. Нужно отдельно проверить, как дочерние процессы наследуют cgroup membership в выбранной модели запуска.

### 6. Продолжить startup

Parent отправляет child разрешение через pipe. Child выполняет namespace-specific setup, rootfs setup и `exec` target command.

Если child сообщает startup error, parent не переводит контейнер в `running` и начинает cleanup.

### 7. Завершить lifecycle

После exit основного процесса parent:

1. сохраняет exit status;
2. при необходимости читает resource usage;
3. закрывает startup и stdio descriptors;
4. удаляет cgroup;
5. сохраняет результат cleanup;
6. переводит контейнер в `stopped` или `failed`.

## Startup synchronization

Для синхронизации parent и child используем отдельный pipe или socketpair, переданный через `ExtraFiles`.

Минимальный protocol может иметь сообщения:

```text
READY
CONTINUE
ERROR <serialized startup error>
```

Названия и формат protocol будут определены в process runtime change.

Stderr не используется как protocol: это пользовательский диагностический поток, который нельзя надёжно парсить.

## Failure cases

### Ошибка создания cgroup

Процесс не создаётся. Пользователь получает ошибку с path и operation.

### Ошибка применения limit

Child не запускается. Созданная cgroup удаляется best-effort.

### Ошибка добавления PID

Child не получает `CONTINUE`. Parent отправляет stop signal при необходимости, ждёт завершения и удаляет cgroup.

### Child завершился до добавления в cgroup

Parent должен обнаружить ошибку записи PID или закрытие startup pipe. Контейнер не считается running.

### Cleanup cgroup не удался

Основной exit status сохраняется отдельно от cleanup error. Metadata должна позволить диагностировать orphan cgroup.

## Cleanup и вложенные процессы

Удалить cgroup v2 можно после того, как в ней не осталось процессов. Если target command оставила дочерние процессы, runtime должен определить policy:

- дождаться всех процессов;
- отправить им signals;
- использовать cgroup.kill, если он доступен и разрешён policy.

Точная process tree policy будет частью signal and shutdown design.

## Resource semantics

Не заданный limit не записывается искусственно как `max`, если это не требуется runtime policy.

Usage считывается после завершения до удаления cgroup, если соответствующие control files доступны.

Ошибки resource usage не должны маскировать exit status процесса, но должны попадать в diagnostics.

## Причины выбора

- лимиты применяются до запуска пользовательской команды;
- startup synchronization устраняет race condition;
- parent сохраняет управление lifecycle;
- cgroup path связан с container ID;
- protocol отделён от stdout/stderr;
- cleanup можно проверять независимо от exit status.

## Критерии готовности

- cgroup создаётся до child process;
- limits записываются до исполнения target command;
- PID добавляется через `cgroup.procs`;
- child ждёт явного разрешения;
- ошибка startup не приводит к состоянию `running`;
- cgroup удаляется после завершения процесса;
- orphan cgroup диагностируется;
- integration test проверяет memory/pids limit и порядок запуска.

## Открытые вопросы

- использовать ли `cgroup.kill` для остановки всей группы;
- как находить и подключать parent cgroup;
- какие controllers обязательны для MVP;
- нужно ли читать usage в MVP или только limits;
- как тестировать cgroups без изменения host hierarchy.
