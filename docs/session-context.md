# Session context

## Project

`cocker` — Linux-only учебный container runtime с production-oriented архитектурой.

## Current status

- Репозиторий создан с нуля.
- Базовый process runner завершён; в блоке конфигурации завершены Tasks 1–3, следующая — Task 4.
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
- Реальная process структура уточнена: contract находится в `internal/runtime/process`, Linux implementation — в `internal/platform/linux/process/runner_linux.go`, composition root — в `cmd/cocker/main.go`.
- Активный change — `docs/changes/002-container-config/`: Container Configuration and Domain Model.
- Task 1 (`Domain types and invariants`) завершена: domain-модель, defaults, validation errors и тесты добавлены; formatting, tests и vet прошли.
- Task 2 (`Resource value parsers`) завершена и принята после повторного review: исправлены parsing memory без суффикса и промежуточное переполнение CPU; boundary tests, formatting, tests, vet и build прошли.
- Task 3 (`JSON decoding and normalization`) завершена и принята; коммит `e7b5eb8` (`feat: add JSON configuration decoding and normalization`). Реализованы `Input`, `DecodeJSON → Input` и отдельный `Normalize → ContainerSpec`, ошибки с сохранением причин и unit tests. Tests, vet, build, gofmt и diff check прошли. После review добавлены GoDoc-комментарии.
- Формат сопровождения зафиксирован: по умолчанию давать сигнатуры, цели, стартовые детали, assertions и команды проверки без полного листинга кода.
- После завершения каждого связанного блока задач требуется отдельное большое milestone review до начала следующего блока.
- Границы блоков определяет агент до реализации и явно фиксирует цель, задачи, критерий завершения, исключения и проверки.
- Блок `Basic Process Runner` реализован и прошёл milestone review: Go module, runtime process contract, Linux runner, CLI `run`, runner tests и CLI tests включают проверку приоритета `Canceled` над диагностическим `Signal`.

## Agreed constraints

- При каждой новой сессии агент после чтения этого файла должен до начала работы ознакомиться с документацией из `docs/development/`, начиная с `workflow.md`.
- Разработка и запуск ориентированы на Ubuntu/Linux.
- Код пишет пользователь вручную.
- Работа ведётся через спецификации, design-документы и небольшие задачи.
- Основной API — стандартная библиотека Go.
- Низкоуровневый доступ к Linux выполняется через `golang.org/x/sys/unix`.
- Абстракции runtime, storage и platform должны позволять дальнейшее расширение.
- Каждая задача сопровождается краткой справкой по нужным темам Go и Linux.
- Все решения и текущий прогресс должны быть доступны следующей сессии без обращения к истории чата.

## Next step

Следующая задача — Task 4: CLI overrides and merge. Рабочая ветка —
`feature/change-002-container-config`. Явные CLI overrides объединяются с
JSON до normalization. JSON pids имеет тип `*uint64`; `ParsePidsLimit`
сохранён для строкового CLI-ввода. Финальный вызов `ContainerSpec.Validate()`
подключается отдельно после merge/normalization в Task 5.

Согласована foreground IO policy для Task 3: normalization устанавливает
`IO.AttachStdin`, `IO.AttachStdout` и `IO.AttachStderr` в `true`, используя
потоки вызывающего CLI без PTY. Поле `IO` отсутствует в `config.Input` и JSON;
пользовательские IO overrides и detached mode в текущий change не входят.

Большое milestone review всего блока обязательно после Task 5. Отложены
упрощение helpers проверки цифр и единый overflow sentinel между parser'ами.

## Open questions
