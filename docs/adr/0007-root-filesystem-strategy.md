# ADR-0007: Root filesystem через mount namespace и `pivot_root`

## Статус

Accepted for MVP

## Контекст

Процесс контейнера должен видеть заданный root filesystem вместо корня хостовой системы. Одной смены текущей директории недостаточно: абсолютные пути вроде `/bin/sh`, `/etc/hosts` и `/proc` должны разрешаться относительно нового корня.

Нужно также не допустить, чтобы mount operations внутри контейнера неожиданно меняли mount tree хоста.

## Краткая справка: mount namespace

Mount namespace даёт процессу собственное представление о смонтированных файловых системах и точках монтирования.

Если сначала создать отдельный mount namespace, операции mount, bind mount и unmount можно выполнять внутри него, не изменяя mount namespace родителя при корректной настройке propagation.

Mount namespace отвечает за видимость mount tree. Он не копирует автоматически все файлы: root filesystem должен быть подготовлен отдельно.

## Краткая справка: `chroot`

`chroot(path)` меняет корневой каталог, который используется при разрешении путей конкретного процесса и его потомков.

`chroot` не является полноценной границей безопасности контейнера. Привилегированный процесс может использовать доступные ему kernel mechanisms для выхода из ограничений, а mount tree и другие namespaces могут оставаться общими с хостом.

Поэтому `chroot` сам по себе не решает задачу container rootfs. В нашем проекте он не будет основным способом изоляции.

## Краткая справка: `pivot_root`

`pivot_root(new_root, put_old)` меняет root filesystem текущего mount namespace:

- `new_root` становится новым `/`;
- старый root временно перемещается в `put_old` внутри нового root;
- после этого старый root можно размонтировать.

В отличие от простого `chroot`, `pivot_root` работает с mount tree и позволяет корректно убрать старую root filesystem из namespace контейнера.

Для успешного вызова `new_root` должен быть подходящим mount point, а `put_old` должен находиться внутри нового root и быть отдельным каталогом.

## Решение

Для MVP используем:

1. отдельный mount namespace;
2. настройку mount propagation, чтобы изменения не распространялись обратно на host namespace;
3. подготовленный rootfs, доступный как mount point;
4. `pivot_root(newRoot, putOld)`;
5. unmount старого root через `MNT_DETACH` после перехода;
6. mount необходимых pseudo-filesystems внутри нового root.

`chroot` не используется как основной путь. Fallback без `pivot_root` не входит в базовый MVP и потребует отдельного ADR, потому что он меняет security и cleanup assumptions.

## Предварительный порядок setup

```text
create mount namespace
        │
        ▼
make mounts private
        │
        ▼
prepare rootfs as mount point
        │
        ▼
create <new-root>/old-root
        │
        ▼
pivot_root(new-root, new-root/old-root)
        │
        ▼
chdir("/")
        │
        ▼
mount /proc, /dev and required pseudo-filesystems
        │
        ▼
unmount /old-root
        │
        ▼
exec target command
```

Точная последовательность mount propagation и pseudo-filesystems будет описана в rootfs specification перед реализацией.

## Rootfs requirements

Rootfs для MVP:

- должен быть каталогом, доступным runtime;
- должен содержать исполняемый target command или нужный interpreter;
- должен содержать необходимые shared libraries, если команда динамически связана;
- должен иметь каталоги, необходимые для mounts;
- не должен быть заменён или удалён во время запуска контейнера;
- должен быть проверен до создания процесса.

`cocker` не собирает rootfs в MVP. Пользователь предоставляет его заранее.

## Pseudo-filesystems

После смены root runtime может смонтировать:

- `/proc` для process view контейнера;
- `/dev` в ограниченном варианте;
- `/sys` только если это необходимо и с отдельной security policy;
- `/tmp` как tmpfs, если это входит в configuration.

Каждый mount должен иметь явную цель, source, filesystem type, flags и cleanup policy. Автоматически пробрасывать host mounts внутрь контейнера нельзя.

## Propagation

Mount propagation определяет, передаются ли изменения mount tree между связанными namespaces.

Перед настройкой rootfs runtime должен сделать mount tree контейнера private или применить эквивалентную policy, предотвращающую propagation обратно на host.

Невыполнение этого требования может привести к тому, что mount, созданный внутри контейнера, появится на хосте.

## Cleanup

Cleanup должен:

1. размонтировать pseudo-filesystems и временные mounts;
2. удалить или отмонтировать old root;
3. закрыть связанные file descriptors;
4. обработать уже отсутствующие mounts идемпотентно;
5. сохранить ошибку cleanup отдельно от exit status процесса.

Если cleanup невозможно выполнить внутри завершившегося child, parent должен иметь достаточно metadata для восстановления или диагностики ресурсов.

## Причины выбора

### Корректная модель mount tree

`pivot_root` лучше соответствует задаче container runtime: он меняет root mount и позволяет удалить старый root из namespace.

### Изоляция от host mount namespace

Отдельный mount namespace и private propagation делают границу видимости явной.

### Учебная ценность

Решение позволяет последовательно изучить mount namespace, mount propagation, bind mounts, pseudo-filesystems, `pivot_root` и cleanup.

## Последствия

### Положительные

- полноценная модель rootfs для MVP;
- меньше зависимости от поведения `chroot`;
- явный контроль mount tree;
- подготовка к overlayfs и filesystem layers;
- возможность отдельно тестировать rootfs setup.

### Отрицательные

- `pivot_root` требует аккуратной подготовки mount points;
- mount operations сложны и требуют privileged permissions;
- cleanup может быть сложнее запуска;
- integration tests должны выполняться в Linux environment и не должны повреждать host mounts.

## Альтернативы

### Только `chroot`

Отклонено: `chroot` не изолирует mount namespace и не является достаточной runtime boundary.

### `chroot` плюс mount namespace

Отклонено как основной вариант: это может работать для простого rootfs, но хуже выражает управление mount tree и old root cleanup.

### Overlayfs с первого этапа

Отклонено: overlayfs нужен для image layers и copy-on-write, но не требуется для запуска заранее подготовленного rootfs.

### Готовый OCI runtime

Отклонено в рамках цели изучить kernel mechanisms самостоятельно.

## Требования к реализации

Rootfs package должен:

1. Проверять rootfs до запуска child.
2. Проверять отсутствие опасных неоднозначностей с path traversal и symlinks согласно security policy.
3. Выполнять операции только в child mount namespace.
4. Использовать `golang.org/x/sys/unix` для `mount`, `pivot_root`, `umount2` и связанных операций.
5. Возвращать ошибки с названием операции и путями.
6. Не смешивать rootfs setup с CLI parsing.
7. Иметь integration tests на disposable rootfs.

## Связанные документы

- [`docs/adr/0006-process-and-namespace-creation.md`](0006-process-and-namespace-creation.md)
- [`docs/adr/0002-root-mode-for-mvp.md`](0002-root-mode-for-mvp.md)
- [`docs/specs/container-lifecycle/states.md`](../specs/container-lifecycle/states.md)
