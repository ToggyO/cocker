# Change 002: Container Configuration and Domain Model

## Статус

Proposed

## Проблема

Текущий process runner умеет запускать процесс, но проект ещё не имеет единой модели желаемого контейнера. CLI и будущий JSON-файл должны преобразовываться в одинаковую нормализованную конфигурацию до создания процессов и Linux resources.

## Цель

Создать чистую domain/config основу для `cocker run`:

- определить `ContainerSpec` и связанные типы;
- реализовать parsing JSON и пользовательских resource values;
- объединять JSON и CLI overrides с приоритетом CLI;
- применять defaults и normalization;
- отклонять невалидную конфигурацию до создания metadata, processes и kernel resources.

## Пользовательская ценность

Пользователь получает один предсказуемый способ описать запуск контейнера через CLI или JSON. Одинаковые значения дают одинаковый результат независимо от источника, а ошибки содержат поле и причину проблемы.

## Критерии приёмки

- CLI и JSON проходят общий pipeline `decode → merge → normalize → validate`.
- Нормализованный `ContainerSpec` не содержит строковых resource formats.
- Неизвестные JSON-поля отклоняются.
- Невалидная конфигурация не запускает процесс и не создаёт kernel resources.
- Domain package не зависит от CLI, JSON decoder или Linux APIs.
- Unit tests покрывают defaults, overrides, resource boundaries, duplicate environment variables и invalid values.

## Не входит в change

- cgroup implementation;
- namespace или rootfs setup;
- metadata storage;
- parent/child protocol;
- mounts, volumes, networking configuration, capabilities и seccomp;
- rootless mode.
