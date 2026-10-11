# ADR-0005: Каталог runtime state — `/var/lib/cocker`

## Статус

Accepted for MVP

## Контекст

`cocker` должен сохранять данные, необходимые для управления контейнерами после завершения команды `run`:

- container ID;
- lifecycle state;
- PID;
- command and arguments;
- rootfs;
- resource limits;
- timestamps;
- exit status;
- error diagnostics;
- paths to logs and runtime resources.

Эти данные должны отличаться от временных данных процесса и находиться в месте, которое понятно системному администратору.

## Краткая справка: Linux filesystem hierarchy

`/var/lib` обычно используется приложениями для хранения изменяемого постоянного состояния. Это подходящее место для metadata, локальных баз данных и runtime state, которые переживают отдельный запуск процесса.

`/run` обычно используется для данных текущего boot или активных процессов. Там можно хранить PID files и временные sockets, но полное состояние контейнера после перезагрузки там хранить неудобно.

`/tmp` предназначен для временных файлов и не подходит как основной каталог управления контейнерами.

## Решение

Production default state directory — `/var/lib/cocker`.

Путь можно переопределить через глобальный CLI-флаг:

```text
cocker --state-dir /custom/path run ...
```

Внутри каталога используется структура:

```text
/var/lib/cocker/
    containers/
        <container-id>/
            metadata.json
            state.json
            logs/
                stdout.log
                stderr.log
            runtime/
                mounts/
                cgroup-path
```

Точные имена файлов могут измениться при разработке storage package, но container ID должен оставаться основным ключом изоляции данных.

## Разделение постоянного и временного состояния

Metadata, lifecycle state, exit status и диагностические данные считаются persistent runtime state.

Mount points, pipes, sockets и другие объекты, существующие только во время работы процесса, считаются runtime resources. Они могут иметь ссылки или описания в state directory, но их фактическая жизнь определяется Linux kernel и cleanup logic.

После остановки контейнера state directory может сохраняться для `inspect` и диагностики. Автоматическое удаление завершённых контейнеров в MVP не выполняется без явной команды удаления, которая появится позже.

## Права доступа

Для MVP, работающего от root:

- `/var/lib/cocker` создаётся с владельцем и группой, выбранными deployment policy;
- metadata не должна быть доступна произвольным пользователям на запись;
- logs могут содержать данные процесса и должны иметь ограниченные permissions;
- storage package не должен следовать за symlink за пределы state directory без явного решения.

Конкретные mode bits и ownership будут зафиксированы в security design.

## Атомарность записи

Запись metadata не должна оставлять частично записанный JSON при аварийном завершении.

Предварительная стратегия:

1. записать содержимое во временный файл внутри того же каталога;
2. выполнить flush и закрыть файл;
3. атомарно заменить целевой файл через rename;
4. при необходимости синхронизировать каталог.

Требования к durability для каждой записи будут определены при реализации storage package. Не все поля требуют одинаковой стоимости fsync.

## Обнаружение повреждённого состояния

Если `metadata.json` или `state.json` отсутствует либо не разбирается:

- `cocker list` не должен молча игнорировать каталог;
- пользователь получает диагностическую запись с container ID;
- runtime не должен запускать процесс на основании неполной metadata;
- repair или quarantine policy будет отдельной функцией.

Если state указывает на PID, которого уже нет, контейнер считается stale. Точное изменение lifecycle state после обнаружения stale state будет определено в recovery design.

## Тестовый режим

Тесты не должны использовать постоянный `/var/lib/cocker`.

Storage tests используют временный каталог, созданный через `t.TempDir()`. Integration tests могут получать state directory через явный параметр или environment setup.

## Причины выбора

- путь соответствует привычной Linux filesystem hierarchy;
- state переживает завершение процесса и reboot лучше, чем `/run`;
- состояние контейнера отделено от исходного кода и rootfs;
- `--state-dir` позволяет тестировать storage без privileged system path;
- отдельные директории контейнеров упрощают cleanup, inspect и будущую миграцию формата.

## Последствия

### Положительные

- предсказуемое расположение данных;
- возможность сохранять завершённые контейнеры для диагностики;
- изоляция данных разных контейнеров;
- удобный temporary state для тестов;
- подготовка к будущему daemon.

### Отрицательные

- первый запуск обычно требует создать системный каталог;
- нужно определить permissions и ownership;
- авария во время записи может оставить orphan directory;
- понадобится recovery и garbage collection policy.

## Альтернативы

### `/run/cocker`

Отклонено как основной каталог: `/run` подходит для transient state и активных PID/socket files, но не для истории завершённых контейнеров.

### `/tmp/cocker`

Отклонено: данные могут быть удалены системой и не должны использоваться для управления контейнерами.

### Только in-memory state

Отклонено: после завершения CLI невозможно выполнить `list`, `inspect` и корректно восстановить диагностику.

### SQLite с первого этапа

Отклонено для MVP: JSON-файлы проще изучить и проверить, а repository interface оставляет возможность перейти на SQLite при росте требований к concurrency и recovery.

## Требования к реализации

Storage package должен:

1. Принимать state directory через dependency injection или явную конфигурацию.
2. Не содержать hardcoded `/var/lib/cocker` внутри отдельных операций.
3. Создавать каталоги безопасно и проверять их тип.
4. Записывать metadata атомарно.
5. Уметь перечислять контейнеры без зависимости от порядка файловой системы.
6. Возвращать различимые ошибки для missing, corrupt и permission denied state.
7. Иметь unit tests на временном каталоге.

## Связанные документы

- [`docs/adr/0002-root-mode-for-mvp.md`](0002-root-mode-for-mvp.md)
- [`docs/adr/0004-container-configuration-format.md`](0004-container-configuration-format.md)
- [`docs/specs/container-lifecycle/states.md`](../specs/container-lifecycle/states.md)
