# Offline Network Namespace

## Статус документа

- Статус: draft
- Версия: 0.1
- Связанный документ: [`docs/architecture/namespace-defaults.md`](namespace-defaults.md)

## Решение

В MVP network namespace создаётся всегда, но внешний network connectivity не настраивается.

Внутри namespace runtime поднимает только loopback interface `lo`.

Итоговая модель:

```text
container network namespace
    └── lo: UP, 127.0.0.1

host interfaces: invisible
veth/bridge/NAT: absent
external network: unavailable
```

## Краткая справка: network namespace

Network namespace изолирует network stack процесса:

- network interfaces;
- routing table;
- local addresses;
- firewall context;
- network-related namespace state.

Процесс в отдельном network namespace не видит host `eth0`, `wlan0` и другие interfaces автоматически.

Создание namespace само по себе не создаёт работающую сеть. Даже loopback interface может существовать в состоянии DOWN, пока runtime явно не поднимет его.

## Краткая справка: loopback

Loopback interface — виртуальный интерфейс для соединений процесса с самим собой. Обычно используется адрес `127.0.0.1` и hostname `localhost`.

Многие приложения ожидают наличие loopback даже тогда, когда им не нужен внешний network access. Поэтому offline namespace с поднятым `lo` даёт более предсказуемое поведение.

## Порядок setup

Network setup выполняется после создания network namespace и до `exec` target command:

1. войти в child network namespace;
2. найти interface `lo`;
3. перевести `lo` в состояние UP;
4. при необходимости назначить loopback address согласно kernel defaults;
5. не создавать veth или bridge;
6. не добавлять default route;
7. продолжить rootfs setup и `exec`.

Если loopback setup не удался, child отправляет startup `ERROR`, а контейнер не переходит в `running`.

## Почему не оставляем host network

Host network по умолчанию означает, что контейнер может видеть host interfaces, routes и локальные network services. Это противоречит принятой isolation policy.

Даже отсутствие полноценной сети безопаснее и понятнее как baseline, чем неявное использование host network.

## Поведение приложений

В MVP:

- `127.0.0.1` доступен внутри контейнера;
- host IP addresses не доступны через container interfaces;
- DNS не настраивается автоматически;
- внешний TCP/UDP access не предполагается;
- port publishing отсутствует;
- network-dependent commands могут завершаться ошибкой.

Это должно быть явно отражено в документации и acceptance tests.

## Security policy

Target command не получает network administration privileges после setup без отдельного решения.

Runtime выполняет privileged network setup до `exec`. Capabilities policy должна определить, какие capabilities остаются после настройки.

Host network configuration и host interfaces не передаются через bind mount или shared namespace.

## Будущее расширение

Полноценная сеть будет отдельной change:

```text
host/bridge
    │
    └── veth pair
             │
             ▼
    container network namespace
```

Она потребует:

- veth pair;
- bridge;
- IP address allocation;
- routes;
- DNS strategy;
- NAT или routing;
- port mapping;
- cleanup при остановке;
- `CAP_NET_ADMIN` policy;
- concurrent network allocation.

Эти требования не должны проникать в offline MVP.

## Альтернативы

### Не создавать network namespace

Отклонено: контейнер использовал бы host network stack и нарушал default isolation policy.

### Создавать namespace, но не поднимать loopback

Отклонено: многие локальные приложения будут получать неочевидные ошибки из-за отсутствия `localhost`.

### Сразу создавать veth и bridge

Отклонено: networking добавляет отдельный большой набор kernel concepts и не нужен для проверки базового process/rootfs runtime.

### Предоставить host networking как default

Отклонено из-за security и воспроизводимости.

## Критерии готовности

- target command не видит host interfaces;
- `lo` существует и находится в состоянии UP;
- внешний network access отсутствует;
- startup fails, если обязательный loopback setup невозможен;
- veth, bridge и NAT не создаются;
- integration test проверяет hostname/network namespace separation;
- cleanup не оставляет network resources.
