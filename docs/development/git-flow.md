# Git Flow

## Статус

- Статус: accepted
- Версия: 0.1

## Основные ветки

`master` содержит последний стабильный код. Разработка напрямую от `master` не ведётся.

`develop` содержит наиболее актуальные проверенные изменения. Стабильность `develop` на уровне production release не гарантируется.

Вся обычная разработка начинается от актуального `develop`.

## Начальная настройка

Первоначальную настройку выполняет владелец репозитория:

1. Создать initial commit с текущими спецификациями и документацией.
2. Зафиксировать его в `master`.
3. Создать `develop` от этого состояния.
4. Опубликовать обе ветки в GitHub.
5. При необходимости защитить обе ветки от прямых push.

После этого рабочие ветки не создаются от `master`.

## Рабочие ветки

Любая работа с кодом, спецификациями, ADR и документацией ведётся в отдельной ветке.

Новые возможности и архитектурные изменения используют шаблон `feature/<feature-name-or-tag>`.

Исправления ошибок используют шаблон `fix/<bug-name-or-tag>`.

Примеры: `feature/task-001-process-runner`, `feature/cgroup-v2-limits`, `fix/invalid-rootfs-validation`.

Имена используют lowercase и дефисы.

## Создание ветки

```text
git switch develop
git pull --ff-only origin develop
git switch -c feature/<feature-name-or-tag>
```

Для исправления используется `fix/<bug-name-or-tag>`. Перед созданием рабочей ветки локальный `develop` синхронизируется с удалённым branch.

## Область ветки

Одна ветка решает одну связанную change или один bug fix. Независимые features и случайные cleanup changes не смешиваются.

## Commit messages

Каждый commit message обязан начинаться с префикса Semantic Commit Messages: `<type>: <short description>`.

Источник соглашения: [Semantic Commit Messages](https://gist.github.com/joshbuchea/6f47e86d2510bce28f8e7f42ae84c716).

Разрешённые типы: `feat` — новая функциональность, `fix` — ошибка, `docs` — документация, `style` — форматирование, `refactor` — структура без изменения поведения, `perf` — производительность, `test` — тесты, `build` — сборка и зависимости, `ci` — CI/CD, `chore` — технические изменения, `revert` — отмена commit.

Примеры: `docs: define container lifecycle states`, `feat: add process runner`, `test: cover resource value parsing`, `fix: handle stale container state`.

Описание должно быть коротким и конкретным. Точка в конце первой строки не добавляется. Идентификатор задачи можно добавить в subject: `feat: add child process runner [TASK-001]`.

## Commit granularity

Commit представляет логически завершённое изменение: domain type, parser, repository operation, integration test или исправление конкретной ошибки.

Не следует делать один commit на всю feature, если он скрывает несколько независимых этапов review.

## Проверки перед commit

Перед commit нужно:

1. проверить `git status`;
2. проверить staged files;
3. выполнить проверки task;
4. сверить код со specification;
5. использовать корректный commit prefix.

Для документации дополнительно проверяются локальные ссылки, статус документа и `docs/session-context.md`.

## Обновление рабочей ветки

Если `develop` изменился, выполнить `git fetch origin` и `git rebase origin/develop` либо использовать согласованную merge strategy. Перед rebase незакоммиченные изменения сохраняются отдельно.

## Pull request flow

Рабочие ветки вливаются сначала в `develop`: `feature/*` и `fix/*` → `develop`.

После проверки `develop` вливается в `master` отдельным release PR.

PR содержит описание изменения, ссылку на specification/task, проверки, ограничения и информацию о пропущенных privileged tests.

После merge рабочая ветка удаляется, если в ней больше нет незавершённой работы.

Прямой merge feature или fix branch в `master` запрещён, кроме emergency fix process.

## Release flow

Когда `develop` готов к стабильному релизу:

1. выполнить полный validation;
2. создать PR из `develop` в `master`;
3. проверить release notes и документацию;
4. влить изменения в `master`;
5. при необходимости создать version tag;
6. синхронизировать `develop` с `master`.

## Emergency fixes

Для production-критичной ошибки создать `fix/*` от `master`, проверить исправление, влить его в `master`, затем перенести то же исправление в `develop` и синхронизировать документацию. Исключение описывается в PR.

## Handoff между сессиями

Перед завершением работы обновляются `docs/session-context.md`, active specification, task status и список проверок.

Новая сессия начинает с проверки `git status`, текущей ветки через `git branch --show-current` и последних commits через `git log -5 --oneline`. Затем читаются `README.md`, `docs/session-context.md`, активная specification и связанные ADR.

## Правила для агента

- обычные рабочие ветки создаются только от `develop`;
- прямые commits в `master` и `develop` не выполняются;
- независимые changes не смешиваются;
- commit prefix обязателен;
- история опубликованных веток не переписывается без явного решения владельца;
- в handoff сообщаются branch, commits и checks.
