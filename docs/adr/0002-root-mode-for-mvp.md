# ADR-0002: Запуск MVP от root с подготовкой к rootless mode

## Статус

Accepted for MVP

## Контекст

Linux ограничивает операции с namespaces, mount points, cgroups и capabilities через модель прав.

`root` — это привилегированный пользователь с широкими правами в системе. В учебной первой версии запуск от root упрощает работу с kernel interfaces, но контейнер, запущенный от root, нельзя автоматически считать безопасным: процесс внутри может получить опасные полномочия, если runtime неправильно настроит namespaces, capabilities и filesystem.

`Rootless mode` — режим, в котором runtime и контейнер запускаются от обычного пользователя без постоянных root privileges. Для него обычно нужны user namespace, UID/GID mapping, корректная обработка subordinate IDs, ограничения на mount operations и специальная модель доступа к cgroups.

## Краткая справка: Linux credentials

Каждый процесс имеет credentials, среди которых есть UID, GID и набор capabilities.

UID определяет пользователя процесса. GID определяет его основную группу и влияет на доступ к файлам.

Capabilities разделяют традиционные полномочия root на отдельные права. Например, `CAP_SYS_ADMIN` затрагивает множество системных операций, включая часть mount-related действий.

Namespaces изолируют видимость ресурсов, но сами по себе не превращают привилегированный процесс в безопасный. Права процесса и доступ к объектам нужно рассматривать отдельно от того, что процесс видит.

## Краткая справка: user namespace

User namespace создаёт отдельное пространство идентификаторов пользователей и capabilities. Процесс может быть UID 0 внутри user namespace, оставаясь обычным пользователем снаружи.

Это основа rootless подхода, но она требует явного mapping между внутренними и внешними UID/GID. Также не каждая операция внутри user namespace автоматически разрешена: kernel проверяет реальные права и ограничения конкретного типа ресурса.

## Решение

MVP запускается от root на поддерживаемой Ubuntu-системе.

При этом архитектура должна изолировать проверку прав, user identity, namespace configuration и resource setup так, чтобы rootless mode можно было добавить отдельной реализацией или policy layer после MVP.

## Причины выбора

### Уменьшение первоначальной сложности

Rootless mode добавляет несколько самостоятельных тем:

- user namespace;
- UID/GID mapping;
- `/etc/subuid` и `/etc/subgid`;
- ограничения unprivileged user namespaces;
- rootless filesystem and networking;
- cgroup delegation;
- cleanup ресурсов, созданных не владельцем системы.

Если включить всё это в первый runtime, станет сложнее понять, какая ошибка относится к container lifecycle, а какая — к модели привилегий.

### Последовательное обучение

Сначала разбираем базовые kernel mechanisms и lifecycle контейнера. Затем добавляем security hardening и rootless mode как отдельную возможность с понятными предпосылками.

### Полный MVP всё равно требует security work

Запуск от root не отменяет необходимость изучить capabilities, ограничения rootfs, обработку сигналов и очистку ресурсов. Эти темы остаются обязательными перед тем, как считать runtime безопасным.

## Последствия

### Положительные

- проще создавать namespaces и mounts;
- проще настраивать cgroup v2;
- меньше ранних ошибок, связанных с permission model;
- проще писать первые Linux integration tests;
- легче локализовать ошибки системных вызовов.

### Отрицательные

- запуск MVP требует root privileges;
- ошибку в runtime нельзя считать безопасной только потому, что код учебный;
- проект первое время не будет пригоден для обычного непривилегированного пользователя;
- придётся явно документировать опасные операции и тестировать их на disposable Linux environment.

## Границы решения

Это решение не означает, что:

- контейнер внутри MVP получает неограниченные capabilities;
- rootfs можно считать доверенным;
- проект готов к запуску недоверенного кода;
- rootless mode не будет поддерживаться;
- security model можно отложить до самого конца.

До выполнения недоверенных сценариев нужно определить capabilities policy, доступные mounts, read-only режимы, seccomp и обработку privileged operations.

## Требования к архитектуре

Код должен позволять отдельно определить:

1. Кто запускает runtime и с какими credentials.
2. Какие namespaces создаются.
3. Какой UID/GID получает процесс внутри контейнера.
4. Какие capabilities остаются у процесса.
5. Какие filesystem operations разрешены.
6. Как runtime создаёт и удаляет cgroup.
7. Какие операции требуют root или delegated permission.

Проверки прав должны находиться в platform/security boundary, а не быть разбросаны по CLI и domain model.

## План для rootless mode после MVP

Rootless mode будет отдельной change, включающей:

- user namespace;
- UID/GID mapping;
- проверку subordinate ID ranges;
- доступ к cgroup v2 и delegation policy;
- ограничения rootless mounts;
- rootless networking strategy;
- отдельные integration tests от обычного пользователя;
- анализ residual capabilities и escape boundaries.

До начала этой change потребуется отдельный security design, а не только набор implementation tasks.

## Альтернативы

### Сделать rootless mode частью MVP

Отклонено из-за резкого увеличения сложности первой реализации и числа зависимых Linux concepts.

### Навсегда требовать root

Отклонено, поскольку это ограничивает практическую ценность инструмента и не позволяет изучить важную для современных runtime модель user namespaces.

### Использовать setuid helper

Отклонено. Setuid-компонент значительно увеличивает security risk и не нужен для первой версии.

## Проверка решения

До первой реализации runtime нужно явно проверить:

```text
id -u
mount | grep cgroup
cat /proc/self/status
```

Integration tests MVP должны проверять поведение при запуске с root privileges и выдавать понятную ошибку при недостаточных правах. Тесты не должны без необходимости изменять постоянное состояние хоста.

## Связанные документы

- [`docs/adr/0001-cgroup-v2.md`](0001-cgroup-v2.md)
- [`docs/product/vision.md`](../product/vision.md)
- [`docs/architecture/overview.md`](../architecture/overview.md)
