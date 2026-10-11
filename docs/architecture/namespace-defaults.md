# Namespace Defaults

## Статус документа

- Статус: draft
- Версия: 0.1
- Связанный документ: [`docs/architecture/container-spec.md`](container-spec.md)

## Назначение

Документ определяет namespaces, которые создаются по умолчанию для контейнера MVP.

## Краткая справка: namespace

Linux namespace ограничивает видимость определённого класса ресурсов для процесса.

Namespace не является общим «контейнерным режимом». Каждый тип изолирует отдельную область: процессы, mounts, hostname, IPC или network stack. Поэтому runtime должен явно решить, какие namespaces создавать.

## Default policy

| Namespace | MVP default | Причина |
|---|---:|---|
| PID | enabled | отдельное дерево процессов и PID 1 внутри контейнера |
| Mount | enabled | отдельный rootfs и mount tree |
| UTS | enabled | отдельный hostname |
| IPC | enabled | изоляция System V IPC и POSIX IPC объектов |
| Network | enabled | контейнер не использует host network namespace |
| User | disabled | отдельная задача rootless mode и UID/GID mapping |

## PID namespace

PID namespace даёт процессу отдельное представление дерева процессов.

Первый процесс внутри нового PID namespace получает PID 1. Он отвечает за обработку сигналов и сбор завершившихся дочерних процессов в пределах namespace.

Для `cocker` это означает:

- target command становится основным процессом контейнера;
- внутри контейнера он видит другие процессы относительно нового namespace;
- PID внутри контейнера и PID на хосте отличаются;
- parent хранит host PID для управления процессом.

Нужно учитывать, что создание PID namespace влияет на момент, когда child становится видимым внутри нового namespace. Это будет проверено Linux integration test.

## Mount namespace

Mount namespace создаёт отдельное mount tree. Он обязателен для `pivot_root`, mounts `/proc` и cleanup без изменения host mount namespace.

Перед rootfs setup runtime должен применить private propagation policy. Иначе изменения могут распространяться между связанными namespaces.

## UTS namespace

UTS namespace изолирует hostname и NIS domain name.

Он позволяет задавать hostname контейнера без изменения hostname хоста. Если hostname не задан пользователем, runtime использует default policy, которая будет определена в config specification.

UTS namespace не является security boundary и не влияет на network identity или DNS.

## IPC namespace

IPC namespace изолирует System V IPC objects, POSIX message queues и связанные IPC resources.

Он снижает риск пересечения IPC объектов между контейнером и host processes. Для обычных процессов его создание недорого, поэтому включаем его в default policy MVP.

## Network namespace

Network namespace даёт отдельный network stack: interfaces, routing table, firewall context и sockets, привязанные к namespace.

В первой версии network namespace будет создан без полноценного подключения к host network. Внутри контейнера нужно предусмотреть loopback interface и определить, как будет вести себя команда до появления veth/bridge networking.

Создание network namespace без настройки внешнего интерфейса означает изолированный network stack, а не работающую сеть.

Это важное различие: отсутствие сети — временное ограничение MVP, а использование host network по умолчанию было бы другой isolation policy.

## User namespace

User namespace позволяет создать отдельное пространство UID/GID и capabilities. Он является основой rootless mode, но требует:

- UID/GID mapping;
- проверки `/etc/subuid` и `/etc/subgid`;
- анализа доступности unprivileged user namespaces;
- отдельной cgroup delegation policy;
- повторной проверки mount и network operations.

В MVP user namespace отключён, потому что текущий runtime запускается от root. Добавление user namespace позже должно быть отдельной change с security design.

## Изменение defaults пользователем

В MVP пользователь не может произвольно отключать обязательные PID и mount namespaces.

Возможность отключить UTS, IPC или network namespace пока не включаем, чтобы isolation guarantee была одинаковой для всех обычных запусков. Если появится необходимость в host-network mode, это будет отдельная явно обозначенная и защищённая настройка.

## NamespaceSpec

Предварительная модель:

```go
type NamespaceSpec struct {
	PID     bool
	Mount   bool
	UTS     bool
	IPC     bool
	Network bool
	User    bool
}
```

Defaults применяются при normalization. Runtime получает уже определённую policy и не должен самостоятельно незаметно менять её.

Для MVP default normalized value:

```text
PID=true
Mount=true
UTS=true
IPC=true
Network=true
User=false
```

## Ошибки

Если обязательный namespace создать невозможно:

- startup не переходит в `running`;
- ошибка содержит тип namespace и системную операцию;
- выполняется best-effort cleanup;
- контейнер переводится в `failed`.

Если network namespace создан, но loopback setup не удался, это считается ошибкой подготовки runtime, пока отдельная спецификация не определит допустимый offline режим.

## Причины выбора

- PID и mount нужны для основной модели контейнера;
- UTS и IPC дают ожидаемую изоляцию окружения;
- network namespace не позволяет случайно использовать host network;
- user namespace требует отдельного rootless design;
- фиксированные defaults делают поведение воспроизводимым.

## Критерии готовности

- normalized `NamespaceSpec` имеет определённые defaults;
- integration tests проверяют PID, mount и UTS isolation;
- network namespace не использует host interfaces;
- user namespace не включается случайно;
- невозможность создать обязательный namespace приводит к `failed` и cleanup;
- CLI не может отключить обязательную isolation policy.

## Открытые вопросы

- включать ли network namespace в первый исполняемый runtime до реализации loopback;
- нужен ли отдельный offline mode;
- какая политика сигналов применяется к PID 1;
- будет ли пользовательский параметр для отключения IPC или UTS после MVP.
