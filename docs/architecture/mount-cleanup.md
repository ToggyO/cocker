# Mount Cleanup

## Статус документа

- Статус: draft
- Версия: 0.1
- Связанные документы:
  - [`docs/adr/0007-root-filesystem-strategy.md`](../adr/0007-root-filesystem-strategy.md)
  - [`docs/architecture/security-policy.md`](security-policy.md)
  - [`docs/specs/container-lifecycle/states.md`](../specs/container-lifecycle/states.md)

## Решение

Runtime ведёт cleanup ledger — упорядоченный список mount operations, которые были успешно выполнены.

При cleanup записи обрабатываются в обратном порядке. Каждая операция имеет достаточно данных для безопасного unmount и диагностики.

## Краткая справка: mount tree

Mount points образуют дерево. Один filesystem может быть смонтирован поверх другого, а mount points могут быть вложенными.

Если сначала убрать родительский mount, пока внутри него существует дочерний mount, операция может завершиться ошибкой или оставить неоднозначное состояние. Поэтому cleanup вложенных mounts выполняется перед их родителями.

## Краткая справка: `umount2`

`umount2` удаляет mount из текущего mount namespace.

Обычный unmount требует, чтобы mount не использовался. Флаг `MNT_DETACH` выполняет lazy unmount: mount отсоединяется от namespace, а фактическое освобождение завершается, когда исчезают активные ссылки.

Lazy unmount полезен для аварийной очистки, но не должен скрывать ошибки обычного cleanup.

## Cleanup record

Предварительная модель:

```go
type MountRecord struct {
	Target       string
	Filesystem   string
	Source       string
	Flags        uintptr
	UnmountFlags uintptr
}
```

В ledger добавляется запись только после успешного завершения mount operation.

Неуспешная попытка mount не должна создавать запись, будто ресурс был создан.

## Предварительный ledger для rootfs

```text
1. make mount propagation private
2. prepare new root mount
3. mount /proc
4. mount /dev или необходимые device nodes
5. mount optional /tmp
6. pivot_root
7. switch to /
8. detach old root
```

Не все операции имеют одинаковую cleanup semantics. Например, изменение propagation не является обычным mount, а old root после `pivot_root` требует отдельного `umount2`.

Поэтому ledger должен уметь хранить разные cleanup actions, а не только слепо вызывать `umount` для каждой строки.

## Обратный порядок

Пример:

```text
mount /tmp       → cleanup /tmp
mount /dev       → cleanup /dev
mount /proc      → cleanup /proc
pivot_root       → cleanup old-root
new root mount   → cleanup new root mount if owned
```

Cleanup выполняется best-effort по всем записям. Ошибка одной операции не должна остановить попытку освобождения остальных ресурсов.

## Mount namespace boundary

Все mount operations и cleanup выполняются в mount namespace child или в специально сохранённом namespace handle согласно runtime design.

Parent не должен выполнять обычный unmount, предполагая, что его mount namespace совпадает с child namespace.

Это критично: unmount в неправильном namespace может ничего не очистить или изменить host mount tree.

## Ownership

Ledger хранит только ресурсы, созданные текущим runtime invocation.

Существующие mounts, обнаруженные в rootfs или host environment, не считаются принадлежащими `cocker`, если runtime явно не взял на себя ownership.

Cleanup не должен удалять user-provided rootfs directory или чужой mount.

## `pivot_root` и old root

После `pivot_root(newRoot, oldRoot)` старый root становится доступен внутри нового namespace через `oldRoot`.

Child должен:

1. сменить current directory на `/`;
2. выполнить unmount old root;
3. удалить каталог `oldRoot`, если он был создан runtime;
4. передать ошибку, если старый root нельзя detach.

Если old root cleanup не завершён до `exec`, target command может сохранить неожиданный доступ к старому mount tree.

## Error policy

Cleanup собирает несколько ошибок в diagnostics, не теряя первую существенную причину.

Нужно различать:

- resource already absent;
- target busy;
- permission denied;
- wrong namespace;
- invalid mount path;
- unexpected kernel error.

Already absent может быть нормальным результатом повторного cleanup. Остальные ошибки должны сохраняться.

## Fallback: lazy unmount

`MNT_DETACH` разрешён как controlled fallback:

- после завершения process set;
- при аварийном shutdown;
- если обычный unmount сообщает busy и policy допускает lazy cleanup.

Lazy unmount нельзя применять автоматически ко всем ошибкам. В metadata нужно сохранять, что использовался fallback и что окончательное освобождение могло быть отложено kernel.

## Crash recovery

Если parent завершился до cleanup, ledger в памяти будет потерян. Поэтому для сложных mounts runtime должен иметь достаточно metadata, чтобы следующая recovery operation могла найти созданные ресурсы.

Полный recovery ledger в persistent storage не входит в первую реализацию. До его появления crash recovery ограничивается диагностикой stale state и проверяемыми ownership paths.

## Причины выбора

- обратный порядок соответствует mount tree dependencies;
- ledger делает ownership явным;
- best-effort cleanup освобождает максимум ресурсов;
- ошибки не теряются;
- обычный unmount остаётся предпочтительным перед lazy unmount.

## Критерии готовности

- каждая успешная mount operation попадает в ledger;
- cleanup идёт в обратном порядке;
- операции выполняются в правильном mount namespace;
- user-provided rootfs не удаляется;
- old root обрабатывается отдельно;
- повторный cleanup безопасен;
- integration test проверяет отсутствие оставшихся mounts после запуска;
- fallback `MNT_DETACH` диагностируется явно.
