# Future Daemon Boundary

## Статус документа

- Статус: draft
- Версия: 0.1
- Связанные документы:
  - [`docs/architecture/overview.md`](overview.md)
  - [`docs/architecture/package-structure.md`](package-structure.md)
  - [`docs/architecture/metadata-storage.md`](metadata-storage.md)

## Назначение

Документ определяет, что останется в CLI, что перейдёт в будущий `cockerd` и какие границы нужно сохранить уже в MVP.

## Краткая справка: CLI и daemon

CLI — короткоживущий процесс, который принимает команду пользователя, вызывает use case и завершается.

Daemon — долгоживущий процесс, который принимает запросы через локальный IPC endpoint, хранит или восстанавливает runtime state и управляет контейнерами после завершения клиентского CLI.

Daemon нужен, когда:

- контейнеры должны продолжать работать после закрытия CLI;
- несколькими клиентами нужно управлять одним runtime;
- требуется единая concurrency policy;
- нужен background event handling и recovery.

## MVP boundary

В MVP:

- `cocker` — единственный пользовательский process;
- CLI напрямую вызывает application use cases;
- runtime управляется в рамках процесса CLI;
- контейнерный state сохраняется на диске;
- отдельного `cockerd` нет.

Это упрощает первую реализацию и позволяет сначала стабилизировать lifecycle.

## Будущая схема

```text
cocker CLI ── Unix socket ──► cockerd
                                │
                                ▼
                         Application layer
                                │
                    ┌───────────┴───────────┐
                    ▼                       ▼
                 Storage                 Runtime
                    │                       │
                    └───────────┬───────────┘
                                ▼
                            Linux kernel
```

CLI станет thin client: он будет разбирать пользовательский ввод, отправлять structured request и форматировать structured response.

Daemon будет владеть runtime lifecycle и длительными процессами.

## Что переиспользуется

Daemon должен переиспользовать:

- `domain` types;
- `config` validation и normalization;
- `app` use cases;
- `storage` repository;
- `runtime` coordinator;
- platform/linux adapters;
- error model и lifecycle state machine.

Daemon не должен требовать отдельной версии container lifecycle.

## Что относится к daemon

Будущий daemon отвечает за:

- Unix socket listener;
- authentication/authorization local clients;
- request concurrency;
- process supervision;
- восстановление stale state;
- background cleanup;
- event delivery;
- lock ownership;
- graceful shutdown самого daemon.

## IPC

Предварительно используем Unix domain socket.

Unix socket не открывает network port и позволяет использовать filesystem permissions для базового local access control.

Protocol должен передавать structured request/response:

```text
request:  command, ContainerSpec, options
response: result, state, error, diagnostics
```

CLI protocol не должен зависеть от stderr text parsing.

Точная serialization и versioning policy будут отдельной API specification.

## Storage ownership

В daemon mode один process должен быть владельцем изменений container records. Это уменьшает риск concurrent updates от нескольких CLI processes.

CLI direct mode MVP остаётся single-controller model. При появлении daemon нужно определить migration и совместимость:

- запрещать direct mode, если daemon запущен;
- использовать lock;
- или иметь отдельные namespaces state.

Нельзя позволять одновременно двум контроллерам независимо менять один lifecycle record.

## Process supervision

Daemon не должен полагаться на то, что CLI остаётся parent process target command.

Он должен:

- владеть process handles или воспроизводимо находить process set;
- обрабатывать SIGCHLD и завершение дочерних процессов;
- сохранять state transitions;
- выполнять cleanup после client disconnect;
- восстанавливать диагностику после своего restart.

Это одна из причин, по которой daemon не включён в MVP: supervision и recovery требуют более зрелой storage и signal model.

## Security boundary

Unix socket должен иметь:

- ограниченные filesystem permissions;
- определённого владельца и группу;
- проверку peer credentials, если она нужна policy;
- ограниченный набор разрешённых operations.

Наличие доступа к socket фактически даёт возможность управлять container runtime, поэтому socket нельзя создавать с unrestricted permissions.

## Версионирование API

Daemon API должен иметь version field. Изменение request/response schema не должно молча ломать старые CLI clients.

Внутренний `ContainerSpec` и внешний daemon API не обязаны быть одной Go struct: API contract может иметь compatibility fields, а application layer получает нормализованную модель.

## Причины решения

- MVP остаётся проще;
- application layer уже становится transport-independent;
- daemon не дублирует lifecycle logic;
- Unix socket подходит для локального системного инструмента;
- state ownership и concurrency заранее учитываются.

## Критерии готовности будущей daemon change

- CLI может работать как thin client;
- application use cases не зависят от CLI;
- daemon владеет lifecycle и state updates;
- Unix socket имеет permission и peer policy;
- client disconnect не останавливает контейнер;
- daemon restart имеет recovery strategy;
- API имеет versioning;
- direct mode и daemon mode не создают concurrent state corruption.

## Не входит в текущую реализацию

- daemon binary;
- Unix socket server;
- API authentication;
- background supervision;
- daemon restart recovery;
- remote TCP API.
