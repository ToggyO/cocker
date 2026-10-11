# Security Policy

## Статус документа

- Статус: draft
- Версия: 0.1
- Связанные документы:
  - [`docs/adr/0002-root-mode-for-mvp.md`](../adr/0002-root-mode-for-mvp.md)
  - [`docs/adr/0007-root-filesystem-strategy.md`](../adr/0007-root-filesystem-strategy.md)
  - [`docs/architecture/namespace-defaults.md`](namespace-defaults.md)

## Назначение

Документ определяет минимальную security policy для MVP и границы того, что `cocker` может обещать пользователю.

Container isolation — это комбинация механизмов, а не один системный вызов:

- namespaces ограничивают видимость;
- cgroups ограничивают ресурсы;
- filesystem policy ограничивает доступ к файлам;
- capabilities ограничивают привилегированные операции;
- seccomp ограничивает доступные system calls;
- user namespace меняет модель идентичности.

Ни один из этих механизмов отдельно не превращает произвольный root process в безопасную sandbox.

## Краткая справка: capabilities

Исторически root ассоциировался с широким набором привилегий. Linux capabilities разделяют часть этих полномочий на отдельные флаги, например `CAP_SYS_ADMIN`, `CAP_NET_ADMIN` и `CAP_KILL`.

Runtime может выполнить privileged setup, а затем удалить ненужные capabilities перед `exec` target command.

Важно различать capabilities parent runtime и capabilities процесса внутри контейнера. Parent может требовать привилегии для создания container resources, но это не означает, что target command должна их наследовать.

## Краткая справка: seccomp

Seccomp ограничивает system calls, которые процесс может выполнять. Обычно используется seccomp-BPF policy.

Seccomp требует хорошо понимать нужные system calls и поведение runtime, поэтому сложный allowlist не включаем в первую реализацию. Неправильная policy может сломать обычные программы или создать ложное ощущение безопасности.

## Security assumptions MVP

MVP:

- запускается от root;
- работает на доверенном Linux host;
- не предназначен для запуска полностью недоверенного кода;
- не обещает escape-proof sandbox;
- требует disposable environment для экспериментов с mount и cgroup operations.

Это ограничение продукта должно быть явно указано в документации и тестовых сценариях.

## Filesystem policy

### Rootfs

Target command видит только подготовленный rootfs после `pivot_root`.

Host root `/` не пробрасывается внутрь контейнера как обычный mount.

Rootfs path проверяется до запуска:

- существует;
- является каталогом;
- не является неожиданным symlink path;
- доступен для необходимых операций;
- не пересекается с runtime state directory без явного решения.

### Bind mounts

Произвольные host bind mounts не входят в MVP. Это уменьшает риск открыть контейнеру чувствительные файлы и упрощает cleanup.

Будущие mounts должны описывать source, target, read-only policy, propagation и допустимые filesystem types. Каждый mount должен быть явно разрешён policy.

### Rootfs read-only

Базовая security policy предпочитает read-only rootfs. Однако некоторые программы требуют writable directories.

Для MVP можно использовать:

- read-only rootfs как default;
- отдельный writable `/tmp` через tmpfs, если это будет нужно;
- явную ошибку для программ, которым нужен writable rootfs, до добавления корректной layer/volume model.

Точный выбор между read-only default и временным read-write режимом должен быть согласован с rootfs integration tests. Read-write rootfs проще для первого прототипа, но слабее как security baseline.

### Pseudo-filesystems

- `/proc` монтируется для process view контейнера;
- `/dev` предоставляется в ограниченном варианте;
- `/sys` не монтируется по умолчанию;
- `/sys/fs/cgroup` не пробрасывается внутрь target command без отдельной policy;
- host `/run` не пробрасывается автоматически.

## `/dev` policy

Полный host `/dev` нельзя bind-mount в контейнер.

Минимальный набор устройств и механизм его создания будет отдельной rootfs task. Предварительно нужны только устройства и pseudo-filesystems, необходимые для базовых команд и stdio.

Доступ к block devices, raw devices и `/dev/kmsg` запрещён без отдельного security decision.

## Capabilities policy

Для MVP parent runtime может работать с root privileges, необходимыми для setup.

Target command должна получить минимально возможный набор capabilities. Если capability dropping ещё не реализован в первом executable slice, это должно быть явно отражено в статусе безопасности и запрещать использование недоверенных workloads.

Будущая policy должна:

1. определить initial capabilities;
2. определить permitted, effective, inheritable, bounding и ambient sets;
3. удалить capabilities, не нужные target command;
4. проверить результат перед `exec`;
5. покрыть policy integration tests.

## Host identity и user namespace

В MVP user namespace отключён. Поэтому UID 0 внутри контейнера соответствует привилегированной модели host credentials.

Это одна из главных причин, по которой MVP нельзя считать безопасной sandbox для недоверенного кода.

Rootless mode и UID/GID mapping должны быть отдельной security change.

## Network policy

Network namespace создаётся отдельно от host network namespace.

До реализации veth/bridge:

- host interfaces не видны контейнеру;
- исходящий network access не предполагается;
- loopback policy должна быть определена;
- `CAP_NET_ADMIN` не должен оставаться у target command без необходимости.

Port mapping и host networking не входят в MVP.

## Seccomp policy

Seccomp не обязателен для первой версии запуска процесса, но должен иметь архитектурную точку подключения перед `exec`.

До появления проверенного профиля runtime не должен заявлять, что system call surface ограничен.

Первый seccomp design должен определить:

- baseline allowlist;
- architecture handling;
- errno policy для запрещённых вызовов;
- совместимость с Go runtime и динамическим linker;
- отдельные integration tests.

## Secrets и sensitive data

MVP не имеет secrets subsystem.

Environment variables могут содержать чувствительные значения, поэтому:

- они не должны автоматически попадать в обычные logs;
- `inspect` должен иметь policy для скрытия sensitive values;
- metadata permissions должны ограничивать доступ;
- JSON config с secrets нельзя считать безопасным secret store.

## Error handling

Security-related error должен содержать операцию и безопасную диагностику, но не раскрывать секреты и полный host filesystem layout без необходимости.

Недостаточные permissions, недоступные capabilities и запрещённые mounts должны приводить к отказу запуска до `running`.

## Причины выбранной policy

- минимизация случайного доступа к host filesystem;
- явное отделение runtime privileges от target command privileges;
- отказ от host network и `/sys` по умолчанию;
- возможность постепенно добавлять capabilities и seccomp;
- честное описание ограничений root-based MVP.

## Критерии готовности

- host rootfs не доступен через стандартный container root;
- произвольные host bind mounts запрещены;
- `/sys` и host `/dev` не пробрасываются автоматически;
- security policy явно документирует root requirement;
- target command не получает непроверенные дополнительные capabilities;
- ошибки permissions приводят к startup failure и cleanup;
- integration tests проверяют filesystem и namespace boundaries.

## Обязательные будущие security changes

- capabilities dropping;
- read-only rootfs и writable tmp policy;
- minimal `/dev` setup;
- seccomp profile;
- user namespace и rootless mode;
- mount allowlist;
- security-focused integration tests;
- threat model для untrusted workloads.
