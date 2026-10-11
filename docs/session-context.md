# Session context

## Project

`cocker` — Linux-only учебный container runtime с production-oriented архитектурой.

## Current status

- Репозиторий создан с нуля.
- Реализация ещё не начата.
- Product vision, architecture overview и lifecycle specification подготовлены.
- ADR-0001 зафиксировал cgroup v2 как модель для MVP.
- ADR-0002 зафиксировал запуск MVP от root с архитектурной подготовкой к rootless mode.
- ADR-0003 зафиксировал Go 1.26 как минимальную поддерживаемую версию.
- ADR-0004 зафиксировал CLI-флаги и JSON-файл как источники конфигурации с единой ContainerSpec.
- ADR-0005 зафиксировал `/var/lib/cocker` как default runtime state directory с `--state-dir` override.
- ADR-0006 зафиксировал `os/exec` и внутренний child mode для создания процесса и настройки namespaces.
- ADR-0007 зафиксировал mount namespace и `pivot_root` как основную модель rootfs для MVP.
- Архитектурный review завершён; end-to-end flow согласован и готов к декомпозиции changes.
- Git flow зафиксирован в `docs/development/git-flow.md`: `master` — stable, `develop` — latest checked, рабочие ветки создаются от `develop`.

## Agreed constraints

- Разработка и запуск ориентированы на Ubuntu/Linux.
- Код пишет пользователь вручную.
- Работа ведётся через спецификации, design-документы и небольшие задачи.
- Основной API — стандартная библиотека Go.
- Низкоуровневый доступ к Linux выполняется через `golang.org/x/sys/unix`.
- Абстракции runtime, storage и platform должны позволять дальнейшее расширение.
- Каждая задача сопровождается краткой справкой по нужным темам Go и Linux.
- Все решения и текущий прогресс должны быть доступны следующей сессии без обращения к истории чата.

## Next step

Создать первую change specification для базового Go-модуля, domain types и запуска обычного дочернего процесса без namespaces и cgroups.

## Open questions
