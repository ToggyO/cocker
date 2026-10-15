# Change 001: Базовый process runner

## Статус

Completed

## Ветка

`feature/task-001-process-runner`

## Цель

Создать минимальный исполняемый срез `cocker`: CLI-команду, которая запускает обычный host process, подключает его стандартные потоки и возвращает результат завершения.

## Зачем это нужно

До добавления namespaces, cgroups и rootfs нужна надёжная основа управления процессом. Эта change проверяет структуру проекта, Go module, границу CLI, context cancellation и обработку exit status.

## Пользовательское поведение

```text
cocker run <command> [args...]
```

Команда запускается без shell. Аргументы передаются без изменений. Standard input, output и error подключаются к текущему CLI-процессу.

## Входит в change

- Go module и binary entrypoint;
- базовая CLI-команда `run`;
- abstraction для запуска процессов;
- context-aware start и wait;
- exit code и signal result;
- unit tests;
- учебные справки по `os/exec`, `Cmd`, `Wait` и exit status.

## Не входит в change

- namespaces;
- cgroups;
- rootfs и `pivot_root`;
- metadata storage;
- JSON configuration;
- parent/child startup protocol;
- privileged Linux operations;
- daemon.

## Краткие критерии приёмки

- `cocker run /bin/echo hello` печатает `hello`;
- аргументы не интерпретируются shell;
- stdin/stdout/stderr подключены;
- exit code сохраняется;
- signal termination представлен в результате;
- некорректный вызов возвращает понятную CLI-ошибку;
- unit tests проходят без root privileges.
