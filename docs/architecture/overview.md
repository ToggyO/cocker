# Cocker: Architecture Overview

## Статус документа

- Статус: draft
- Версия: 0.1
- Связанная продуктовая спецификация: [`docs/product/vision.md`](../product/vision.md)

## Архитектурная цель

Архитектура должна отделять пользовательские сценарии, модель контейнера, управление runtime, хранение состояния и Linux-specific операции.

Это нужно по двум причинам:

1. Системные вызовы и детали ядра не должны распространяться по всему коду.
2. После MVP можно добавлять networking, images, daemon и rootless mode без переписывания CLI и domain model.

## Границы системы

`cocker` управляет жизненным циклом локальных Linux-процессов и ресурсов, которые нужны для их изоляции.

Внешние границы:

- пользовательский CLI;
- Linux kernel API;
- локальная файловая система;
- cgroup v2 filesystem;
- будущий локальный daemon/API.

В MVP система не отвечает за orchestration, registry, удалённые узлы и распределённое хранение.

## Логическая схема

```text
CLI
 │
 ▼
Application layer
 │
 ├── Domain model
 ├── Runtime coordinator
 └── State repository
        │
        ├── Metadata storage
        └── Linux runtime
               │
               ├── Process management
               ├── Namespaces
               ├── Mounts and rootfs
               ├── Cgroups
               └── Signals and cleanup
                      │
                      ▼
                  Linux kernel
```

## Компоненты

### CLI layer

Отвечает за команды, флаги, аргументы, формат вывода и перевод ошибок приложения в exit code CLI.

CLI не должен вызывать Linux system calls напрямую и не должен самостоятельно менять состояние контейнера.

Предполагаемые команды MVP:

- `run`;
- `list`;
- `inspect`;
- `stop`.

### Application layer

Содержит use cases, которые описывают действия пользователя:

- создать и запустить контейнер;
- получить список контейнеров;
- получить описание контейнера;
- остановить контейнер.

Этот слой координирует domain model, runtime и storage. Он не должен зависеть от конкретного формата JSON или Linux-системного вызова.

### Domain model

Содержит понятия, правила и инварианты, не привязанные к Linux API:

- `Container`;
- `ContainerID`;
- состояние контейнера;
- `ContainerSpec`;
- параметры ресурсов;
- exit status;
- допустимые переходы состояний.

Domain model не знает, как именно создаётся namespace или записывается файл в cgroup.

### Runtime coordinator

Координирует технический запуск контейнера:

1. подготавливает runtime resources;
2. создаёт процесс;
3. настраивает namespaces;
4. подготавливает rootfs и mounts;
5. применяет cgroup limits;
6. подключает stdio;
7. ожидает завершение;
8. выполняет cleanup.

Coordinator должен работать через интерфейсы отдельных подсистем, чтобы lifecycle не превратился в один неразделимый системный вызов.

### Linux runtime

Содержит адаптеры к Linux kernel API и filesystem interfaces:

- process creation and waiting;
- namespace operations;
- mount and rootfs operations;
- cgroup v2 operations;
- signal delivery;
- cleanup.

Здесь допустимы `golang.org/x/sys/unix`, Linux-specific constants, errno и операции с `/proc` и `/sys/fs/cgroup`.

### Storage

Хранит metadata и состояние контейнеров. Начальная реализация может использовать JSON-файлы в локальном каталоге, но application layer должен зависеть от repository interface.

Storage не отвечает за состояние Linux kernel resources. Оно хранит описание и результаты управления ими.

## Предварительная структура пакетов

```text
cmd/cocker/
    main.go

internal/
    cli/
    app/
    domain/
    runtime/
    storage/
    platform/linux/
        process/
        namespace/
        mount/
        cgroup/

docs/
    architecture/
    product/
    specs/
    changes/
    adr/
```

Названия пакетов пока являются рабочими. Перед началом реализации их можно изменить, если это улучшит зависимости и читаемость.

## Направление зависимостей

```text
cli -> app -> domain
         │
         ├── runtime interfaces -> Linux runtime
         └── storage interfaces -> file storage
```

Правила зависимостей:

- `domain` не зависит от `cli`, `storage` и `platform`;
- `cli` не знает о Linux system calls;
- `app` зависит от интерфейсов, а не от конкретного JSON storage;
- `platform/linux` изолирует OS-specific код;
- `cmd/cocker` только собирает зависимости и запускает приложение.

## Модель runtime

Внутри MVP контейнер — это управляемый процесс и набор связанных ресурсов:

```text
Container
 ├── Process
 ├── Namespaces
 ├── Root filesystem and mounts
 ├── Cgroup
 ├── Metadata
 └── Lifecycle state
```

Идентификатор контейнера связывает metadata, PID процесса, cgroup path, временные mounts и журналы. При cleanup все созданные ресурсы должны быть найдены по этому идентификатору и обработаны идемпотентно.

## Архитектурные решения, которые нужно принять дальше

Следующие вопросы будут рассмотрены в отдельных ADR:

1. Поддерживаем только cgroup v2 или обе версии.
2. Запускаем первую версию только от root или сразу проектируем rootless mode.
3. Используем ли `clone`/`clone3` через `golang.org/x/sys/unix` или стандартные механизмы `os/exec` с Linux-specific настройками.
4. Используем ли `pivot_root`, `chroot` или комбинацию mount operations для rootfs.
5. Как хранить metadata и как восстанавливать состояние после аварийного завершения.
6. Как разделить foreground CLI и будущий daemon.

## Требования к расширяемости

Будущие функции должны подключаться через новые реализации или подсистемы:

- network backend не должен менять domain model контейнера;
- image store не должен менять процессный lifecycle;
- daemon должен переиспользовать application layer;
- storage backend можно заменить без изменений Linux runtime;
- security policy должна быть отдельной частью runtime configuration.

## Нерешённые вопросы

Этот документ намеренно не выбирает конкретные варианты для перечисленных архитектурных решений. Они станут задачами следующего этапа и будут объяснены вместе с необходимыми Linux concepts.
