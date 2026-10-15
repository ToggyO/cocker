# Session context

## Project

`cocker` — Linux-only учебный container runtime с production-oriented архитектурой.

## Current status

- Репозиторий создан с нуля.
- Реализация базового process runner завершена; следующий функциональный слой ещё не начат.
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
- Формат сопровождения зафиксирован: по умолчанию давать сигнатуры, цели, стартовые детали, assertions и команды проверки без полного листинга кода.
- После завершения каждого связанного блока задач требуется отдельное большое milestone review до начала следующего блока.
- Границы блоков определяет агент до реализации и явно фиксирует цель, задачи, критерий завершения, исключения и проверки.
- Блок `Basic Process Runner` реализован и прошёл milestone review: Go module, runtime process contract, Linux runner, CLI `run`, runner tests и CLI tests включают проверку приоритета `Canceled` над диагностическим `Signal`.

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

CLI-тест на одновременные `Canceled` и `Signal` добавлен. На машине разработчика повторить `gofmt`, `go test ./...` и `go vet ./...`, затем зафиксировать change и перейти к следующему блоку.

## Open questions
