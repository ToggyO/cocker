# Exec Acknowledgment

## Статус документа

- Статус: draft
- Версия: 0.1
- Связанные документы:
  - [`docs/architecture/child-protocol.md`](child-protocol.md)
  - [`docs/architecture/cgroup-lifecycle.md`](cgroup-lifecycle.md)
  - [`docs/specs/container-lifecycle/states.md`](../specs/container-lifecycle/states.md)

## Проблема

Parent должен понять, успешно ли child выполнил `exec` target command.

До `exec` child является bootstrap-программой `cocker`. После успешного `exec` тот же process image заменён target command, но PID, namespaces, cgroup membership и open descriptors могут сохраниться согласно их flags.

Если parent переведёт контейнер в `running` слишком рано, ошибка `exec` будет выглядеть как ошибка уже работающего контейнера. Если переведёт слишком поздно, короткоживущая команда может завершиться до того, как состояние будет записано.

## Краткая справка: close-on-exec

File descriptor с флагом close-on-exec автоматически закрывается kernel во время успешного `exec`.

Это позволяет использовать сам факт закрытия descriptor как сигнал: bootstrap process больше не контролирует protocol channel, потому что его image заменён target command.

Если `exec` не удался, старый process image продолжает работать и может отправить parent структурированное сообщение об ошибке.

## Решение

Используем protocol descriptor с close-on-exec.

### Успешный `exec`

```text
child setup
    │
    ├── send STARTING_EXEC
    ├── exec target command
    └── kernel closes protocol fd
             │
             ▼
          parent sees EOF
```

### Ошибка `exec`

```text
child setup
    │
    ├── exec target command
    ├── exec fails
    ├── send ERROR EXEC_FAILED
    └── exit bootstrap process
```

Parent использует protocol result вместе с process wait status. EOF не считается достаточным без проверки того, что процесс действительно продолжал работу или корректно завершился.

## Protocol states

```text
BOOTSTRAP
   │
   ├── READY
   ▼
WAITING_FOR_CONTINUE
   │
   ├── CONTINUE
   ▼
SETTING_UP
   │
   ├── STARTING_EXEC
   ▼
EXEC_ATTEMPTED
   │
   ├── protocol EOF + process alive  → EXEC_SUCCEEDED / running
   ├── protocol EOF + process exited → classify by wait status
   └── ERROR EXEC_FAILED            → failed
```

`STARTING_EXEC` означает, что все действия до `exec` завершены. Он не означает успешный `exec`.

## Быстрое завершение

Команда может успешно выполнить `exec` и почти сразу завершиться:

```text
exec /bin/true
exit 0
```

Parent может получить EOF и process exit почти одновременно. В таком случае:

- если exit code равен `0`, контейнер считается успешно запущенным и завершившимся;
- final state — `stopped`;
- `running` может не попасть в persistent record как отдельная длительная фаза;
- diagnostics должны сохранять, что команда завершилась до наблюдаемого периода работы.

Команда, которая завершилась с ненулевым code после успешного `exec`, также считается выполненной, а не startup failure.

## Ошибка до `exec`

Ошибки setup до `exec` передаются через protocol:

- mount failure;
- `pivot_root` failure;
- invalid working directory;
- environment setup failure;
- security setup failure;
- target executable lookup или `execve` failure.

Последний случай относится к startup failure, поскольку target command не была запущена.

Child должен отправить stable error code до завершения, если protocol descriptor ещё доступен.

## Parent algorithm

Предварительный алгоритм:

1. Запустить child.
2. Дождаться `READY`.
3. Добавить host PID в cgroup.
4. Отправить `CONTINUE`.
5. Читать protocol channel.
6. При `STARTING_EXEC` продолжить ожидание.
7. При `ERROR` классифицировать startup failure.
8. При EOF дождаться или проверить process wait status.
9. Если EOF соответствует успешному exec, сохранить `running` либо сразу final state для быстро завершившегося процесса.
10. После process exit выполнить cleanup.

Для избежания deadlock parent должен одновременно обслуживать protocol и process lifecycle. Конкретная orchestration model может использовать отдельную goroutine для protocol reader с передачей событий в coordinator.

## Timeout

Parent применяет отдельный startup timeout. Если за timeout не получены ожидаемые protocol events:

- отправляется `ABORT` или signal;
- child process завершается;
- cgroup и mounts очищаются;
- state становится `failed`;
- причина фиксируется как `startup timeout`.

Protocol reader не должен блокировать cleanup навсегда после process exit.

## Descriptor inheritance

Protocol fd должен иметь close-on-exec. Target command не должен наследовать его как рабочий descriptor.

Parent должен закрыть свою копию child-side descriptor сразу после запуска. Иначе EOF может не прийти из-за оставшегося open descriptor.

Это важный lifecycle invariant: каждая сторона владеет только своей копией descriptor.

## Альтернативы

### Отправить `EXEC_SUCCEEDED` после exec

Невозможно выполнить обычным способом: после успешного `exec` bootstrap code больше не выполняется. Можно использовать helper process, но это усложняет lifecycle.

### Считать запуск успешным после `STARTING_EXEC`

Отклонено: `exec` может завершиться с ошибкой сразу после этого сообщения.

### Анализировать stdout/stderr

Отклонено: target command может писать любой текст, а отсутствие вывода ничего не доказывает.

### Проверять только process exit code

Отклонено: exit code не различает ошибку bootstrap и быстрый exit target command без дополнительного protocol signal.

## Критерии готовности

- protocol descriptor имеет close-on-exec;
- parent и child закрывают неиспользуемые копии;
- `STARTING_EXEC` не считается успехом сам по себе;
- `ERROR EXEC_FAILED` отличается от target non-zero exit;
- быстрый `/bin/true` классифицируется как успешный запуск и завершение;
- отсутствующая команда классифицируется как startup failure;
- startup timeout не оставляет child или cgroup;
- integration tests покрывают все три сценария.
