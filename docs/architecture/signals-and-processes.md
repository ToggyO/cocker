# Signals and Process Groups

## Статус документа

- Статус: draft
- Версия: 0.1
- Связанные документы:
  - [`docs/specs/container-lifecycle/states.md`](../specs/container-lifecycle/states.md)
  - [`docs/architecture/cgroup-lifecycle.md`](cgroup-lifecycle.md)
  - [`docs/adr/0006-process-and-namespace-creation.md`](../adr/0006-process-and-namespace-creation.md)

## Назначение

Документ определяет, как `cocker` останавливает контейнер, передаёт сигналы и работает с основным процессом и его потомками.

## Краткая справка: signal

Signal — асинхронное уведомление процесса от kernel или другого процесса.

`SIGTERM` обычно означает просьбу корректно завершиться. Процесс может перехватить его и выполнить cleanup.

`SIGKILL` немедленно завершает процесс. Его нельзя перехватить или проигнорировать, поэтому он используется после истечения timeout.

Signal не является сообщением с произвольными данными и не гарантирует, что target process обработает его в желаемое время.

## Краткая справка: PID 1

Первый процесс в PID namespace получает PID 1 внутри этого namespace.

PID 1 имеет особое поведение в Linux: необработанные сигналы и orphan child processes требуют специального внимания.

В нашем runtime target command становится основным процессом контейнера, поэтому не каждая команда корректно ведёт себя как PID 1. Позже можно добавить tiny init process, но это не входит в первый implementation step.

## Process boundaries

Контейнер может содержать:

- основной target process;
- дочерние процессы target command;
- процессы, созданные shell или application runtime.

Остановка только host PID основного процесса недостаточна: потомки могут продолжить работу или удерживать mounts и cgroup.

## Основная политика остановки

```text
running
   │ stop requested
   ▼
stopping
   │ SIGTERM to container process set
   │ wait timeout
   │ SIGKILL to remaining processes
   │ wait and cleanup
   ▼
stopped
```

Порядок:

1. Перевести state в `stopping`.
2. Определить process set контейнера.
3. Отправить SIGTERM.
4. Ждать завершения до configured timeout.
5. Если процессы остались, отправить SIGKILL.
6. Дождаться завершения и выполнить cleanup.
7. Сохранить stop reason и exit status.

## Process set

### Cgroup v2

Основной способ определить process set — cgroup контейнера. Она включает основной процесс и процессы, которые унаследовали membership.

Если на поддерживаемой системе доступен `cgroup.kill`, runtime может использовать его для завершения всех процессов cgroup. Это уменьшает окно между перечислением процессов и отправкой сигналов.

### Process group

Process group является дополнительной границей сигналов. Parent может отправить сигнал всей группе через отрицательный process group ID.

Process group и cgroup решают разные задачи:

- process group группирует процессы для signal delivery;
- cgroup группирует процессы для resource control и lifecycle cleanup.

Они не должны считаться взаимозаменяемыми.

## Решение MVP

В MVP:

- cgroup является authoritative process set для resource lifecycle;
- runtime использует SIGTERM → timeout → SIGKILL;
- `cgroup.kill` используется, если доступен и успешен;
- process group может использоваться как дополнительный fallback;
- после signal delivery parent всегда ждёт завершения процессов и проверяет cleanup.

Точная минимальная поддерживаемая kernel version для обязательного `cgroup.kill` будет решена в compatibility design. До этого runtime не должен безусловно предполагать его наличие.

## Signal forwarding

В foreground режиме сигналы, полученные CLI, должны быть осмысленно переданы контейнеру.

Например, SIGINT от terminal может быть передан основному процессу, если текущая команда выполняется в attached mode. SIGTERM от `cocker stop` всегда следует lifecycle policy.

Нельзя безусловно передавать любой signal из parent в child: некоторые сигналы относятся к самому CLI или terminal session. Список forwarding policy будет частью CLI specification.

## Stop timeout

Timeout должен быть:

- конечным;
- заданным конфигурацией с безопасным default;
- доступным для явного override;
- применённым отдельно к graceful и forced phase, если это понадобится.

После timeout отсутствие завершения считается основанием для SIGKILL, но не меняет исходный exit status без сохранения stop reason.

## Race conditions

Runtime должен учитывать:

- процесс завершился до отправки SIGTERM;
- процесс завершился между проверкой и отправкой signal;
- новый потомок появился во время graceful shutdown;
- PID был переиспользован после завершения процесса;
- cgroup уже удалена другим cleanup path.

Поэтому операции сигналов должны проверять принадлежность процессного ресурса контейнеру и корректно обрабатывать `ESRCH`, уже завершённый process и отсутствие cgroup.

## Exit status и stop reason

Нужно хранить отдельно:

- как завершился основной process;
- был ли вызван `cocker stop`;
- применялся ли SIGKILL;
- возникла ли ошибка signal delivery;
- завершился ли cleanup.

Например, процесс может завершиться с code `0` после полученного SIGTERM. Это отличается от принудительного SIGKILL и должно быть видно в inspect output.

## PID 1 policy

Для MVP target command является PID 1 внутри PID namespace.

Runtime должен документировать ограничения этого режима:

- target command может не обрабатывать SIGTERM;
- target command может не выполнять reaping дочерних процессов;
- shell может менять signal behavior.

Tiny init process и subreaper mode будут рассматриваться после стабилизации базового lifecycle.

## Cleanup

После завершения process set runtime:

1. проверяет, что в cgroup не остались процессы;
2. закрывает stdio и protocol descriptors;
3. выполняет rootfs/mount cleanup;
4. удаляет cgroup;
5. записывает final state.

Если процессы остались, контейнер не должен быть помечен как полностью cleaned только на основании завершения основного PID.

## Причины выбора

- graceful shutdown даёт приложению возможность закрыть ресурсы;
- SIGKILL ограничивает время зависшего shutdown;
- cgroup отражает resource ownership лучше одиночного PID;
- process group сохраняет fallback для систем без нужного cgroup operation;
- отдельные stop reason и exit status улучшают диагностику.

## Критерии готовности

- `stop` переводит контейнер в `stopping`;
- сначала отправляется SIGTERM;
- после timeout применяется SIGKILL;
- учитываются потомки основного процесса;
- повторный `stop` безопасен;
- `ESRCH` обрабатывается как возможное нормальное завершение;
- exit status и stop reason сохраняются раздельно;
- integration tests проверяют graceful и forced shutdown.

## Открытые вопросы

- сделать ли `cgroup.kill` обязательным для MVP;
- нужен ли process group как обязательная fallback policy;
- какой default timeout выбрать;
- нужен ли отдельный init process;
- какие terminal signals forwarding разрешить в attached mode.
