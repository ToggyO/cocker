# Linux Development Environment

## Статус документа

- Статус: draft
- Версия: 0.1
- Связанные документы:
  - [`docs/adr/0001-cgroup-v2.md`](../adr/0001-cgroup-v2.md)
  - [`docs/adr/0003-minimum-go-version.md`](../adr/0003-minimum-go-version.md)
  - [`docs/architecture/security-policy.md`](security-policy.md)

## Цель

Определить минимальную среду, в которой можно разрабатывать и проверять `cocker`, не смешивая требования Go tooling, Linux kernel и прав доступа.

## Supported baseline

Основной baseline MVP:

- Ubuntu 24.04 LTS;
- amd64 или arm64;
- Linux kernel с доступным cgroup v2;
- systemd-hosted или эквивалентная unified cgroup hierarchy;
- Go 1.26 или новее;
- root privileges для container integration tests MVP;
- отдельная VM или disposable host для mount и namespace tests.

Ubuntu 24.04 LTS выбрана как долгоживущая база с поддержкой standard security maintenance до мая 2029 года. [Официальный release cycle Ubuntu](https://ubuntu.com/about/release-cycle)

Ubuntu 24.04 LTS обычно использует современный kernel baseline, но `cocker` должен проверять фактические kernel features, а не полагаться только на номер дистрибутива. Release notes доступны в [официальной документации Ubuntu](https://documentation.ubuntu.com/release-notes/24.04/).

## Краткая справка: distribution и kernel

Ubuntu — это distribution: userspace, package manager, init system, libraries и kernel package.

Linux kernel предоставляет namespaces, cgroups, mounts и system calls. Две машины с одинаковой Ubuntu version могут иметь разные kernel versions и configuration.

Поэтому compatibility должна проверять capabilities окружения:

- тип cgroup filesystem;
- доступные controllers;
- доступность namespace operations;
- наличие нужных mount features;
- права текущего пользователя.

## Проверка среды

До integration tests нужно диагностировать:

```text
uname -a
go version
id -u
stat -fc %T /sys/fs/cgroup
cat /sys/fs/cgroup/cgroup.controllers
cat /sys/fs/cgroup/cgroup.subtree_control
```

Ожидается cgroup v2 filesystem type `cgroup2fs`. Список controllers может зависеть от host policy и parent cgroup.

Runtime должен выдавать понятную диагностику, если feature или controller недоступен.

## Права

MVP требует root для операций, связанных с:

- созданием некоторых namespaces;
- mount и `pivot_root`;
- настройкой cgroup hierarchy;
- capabilities setup;
- signal management чужих процессов.

Unit tests config, domain, storage и application fakes запускаются без root.

Integration tests, меняющие mount namespace или cgroup state, запускаются только в контролируемой Linux-среде.

## Disposable test environment

Предпочтительная среда integration tests — отдельная Ubuntu VM.

Причины:

- ошибки mount cleanup не затрагивают рабочую систему;
- cgroup policy не конфликтует с desktop/session manager;
- можно воспроизводить kernel и filesystem setup;
- тесты могут иметь root privileges без использования основного host.

Docker container не считается автоматически достаточной средой для всех тестов `cocker`: nested namespaces, mounts и cgroup delegation могут быть ограничены runtime, который запускает тестовый контейнер.

## Test classes

### Unit tests

Запускаются на любой поддерживаемой Go environment:

- domain transitions;
- config parsing;
- resource values;
- repository с `t.TempDir()`;
- application use cases с fake runtime.

### Linux integration tests

Запускаются только на Linux:

- process creation;
- PID и mount namespaces;
- rootfs setup;
- cgroup limits;
- signals и cleanup;
- parent/child protocol.

Тесты должны иметь build tags или runtime checks, чтобы не падать бессодержательно на macOS.

### Security tests

После появления security implementation проверяют:

- отсутствие host rootfs;
- отсутствие host network;
- mount propagation;
- capabilities policy;
- read-only rootfs;
- seccomp policy.

## CI baseline

Минимальный CI pipeline должен включать:

1. formatting check;
2. `go vet`;
3. unit tests;
4. race detector для применимых пакетов;
5. Linux integration job с root-capable isolated runner;
6. сборку бинарника;
7. проверку документации и repository state.

Реальный privileged integration runner нельзя симулировать одним unit test. Если CI не имеет нужных прав, job должна быть явно помечена как skipped with reason, а не как успешная проверка runtime behavior.

## Compatibility policy

Project не обещает работу на любом Linux kernel только по факту наличия Linux.

Перед запуском runtime выполняются feature checks. Минимальный kernel version будет определён только после реализации всех обязательных operations и integration tests.

Такой подход лучше фиксированного числа kernel version, потому что важны конкретные features, их configuration и delegation policy.

## Local development commands

Предварительный набор команд:

```text
go test ./...
go vet ./...
go test -race ./...
go build ./cmd/cocker
```

Privileged tests будут запускаться отдельной командой, например:

```text
go test -tags=integration ./...
```

Точный workflow появится после создания Go module и первой тестовой структуры.

## Известные ограничения

- macOS и Windows не являются runtime targets;
- macOS может использоваться для редактирования и unit tests;
- Linux VM должна поддерживать nested operations, необходимые тесту;
- systemd и host cgroup delegation могут менять доступные controllers;
- `cgroup.kill` и некоторые security features проверяются capability detection, а не только version string.

## Критерии готовности

- есть документированная Ubuntu baseline;
- есть команда диагностики окружения;
- unit tests отделены от privileged integration tests;
- integration tests не запускаются без явной Linux capability check;
- CI проверяет Go baseline;
- kernel feature failure содержит actionable error;
- тестовая среда не требует изменять рабочую ОС пользователя.
