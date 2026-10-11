# Lifecycle State Timing

## Статус документа

- Статус: draft
- Версия: 0.1
- Связанные документы:
  - [`docs/specs/container-lifecycle/states.md`](../specs/container-lifecycle/states.md)
  - [`docs/architecture/exec-acknowledgment.md`](exec-acknowledgment.md)
  - [`docs/architecture/metadata-storage.md`](metadata-storage.md)

## Назначение

Документ определяет, когда lifecycle state записывается в storage и как обрабатывается быстрое завершение target command.

## Основной принцип

Persistent state должен описывать последнюю подтверждённую фазу runtime, а не предположение parent о том, что произошло.

Поэтому:

- `starting` записывается до потенциально долгих privileged operations;
- `running` записывается только после подтверждённого `exec`;
- `stopped` записывается после process wait и cleanup;
- `failed` записывается при ошибке startup или невозможности завершить обязательный cleanup.

## Переходы с точками записи

```text
нет record
    │ validation passed
    ▼
created       ← record persisted
    │ child startup begins
    ▼
starting      ← record persisted
    │ exec confirmed
    ▼
running       ← record persisted
    │ process exit and cleanup
    ▼
stopped       ← final record persisted
```

Ошибка до подтверждения `exec`:

```text
starting → failed
```

Ошибка после подтверждения `exec` относится к runtime/container result и обычно заканчивает lifecycle как `stopped` с exit status, а не как startup failure.

## Почему `starting` сохраняется заранее

Между запуском child и завершением setup могут произойти:

- ошибка mount;
- ошибка cgroup membership;
- timeout protocol;
- авария процесса;
- crash самого parent.

Если state не сохранён как `starting`, после сбоя может остаться record, который ошибочно выглядит как `created` или вообще отсутствует.

## Почему `running` требует exec confirmation

Child process может существовать, но target command ещё не запущена. Наличие PID не доказывает успешный startup.

`running` означает, что:

- child прошёл setup;
- target executable успешно заменил bootstrap image;
- process находится под cgroup policy;
- parent может применять обычную runtime signal policy.

## Быстрое завершение

Команда может завершиться между подтверждением `exec` и записью `running`.

Parent в этом случае использует следующие правила:

1. Если `exec` подтверждён и process exit получен, команда считалась запущенной.
2. Если cleanup успешен, final state — `stopped`.
3. Exit code или signal сохраняется обязательно.
4. Persistent запись `running` может быть пропущена, если между событиями не было безопасной точки записи.
5. Metadata может содержать `started_at`, даже если отдельный state `running` не был виден через `inspect`.

Это не потеря состояния: `stopped` с `started_at` и exit status означает успешный короткий runtime.

## Ошибка записи state

Если runtime operation успешна, но запись следующего state не удалась:

- нельзя молча считать state обновлённым;
- исходная operation result сохраняется в памяти до завершения cleanup;
- повторная запись выполняется, если это безопасно;
- ошибка storage добавляется к diagnostics;
- возможно состояние `unknown` после аварии, если repository недоступен.

`unknown` не добавляется как обычное lifecycle state MVP без отдельной спецификации. Вместо этого используется текущий state с `LastError` и recovery diagnostics.

## Parent crash

Если parent завершился после записи `starting` и до final state:

- record остаётся в `starting`;
- следующая команда обнаруживает stale state по PID и cgroup path;
- автоматический recovery не выполняется до отдельного recovery design;
- `inspect` показывает diagnostics о неполном запуске.

## Stop during startup

Если stop запрошен во время `starting`:

- parent отправляет `ABORT` через protocol, если канал доступен;
- при необходимости завершает child;
- не ждёт `running`;
- выполняет cleanup;
- фиксирует `failed` с причиной `startup aborted` или отдельным stop reason по будущей policy.

## Причины выбора

- состояния отражают подтверждённые runtime events;
- startup failure отличается от target non-zero exit;
- быстрые процессы не теряют exit status;
- parent crash оставляет диагностируемый `starting` record;
- storage semantics остаются явными.

## Критерии готовности

- `starting` сохраняется до запуска child;
- `running` зависит от exec acknowledgment;
- отсутствующая команда не попадает в `running`;
- `/bin/true` оставляет успешный final state и exit code;
- failure записи state не скрывается;
- parent crash оставляет проверяемый stale state;
- integration tests проверяют slow, fast и failed startup.
