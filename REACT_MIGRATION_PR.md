# React Migration: Main Tracking PR

## Overview

Migrate the Positions OS web UI from HTMX + Go Templates to React + TypeScript.

**Status**: 🚧 In Progress (3 parallel work threads)

**Branches**:
- `react-migration` - Main integration branch
- `react-thread-a` - Foundation & Data Layer
- `react-thread-b` - UI Components Library
- `react-thread-c` - Page Implementation

**Goal**: Complete React UI with WebSocket real-time updates, maintaining all existing functionality.

---

## Work Division: 3 Parallel Threads

```
┌─────────────────────────────────────────────────────────────────────┐
│                         COMMON GOAL: MIGRATE TO REACT               │
└─────────────────────────────────────────────────────────────────────┘
                                │
        ┌───────────────────────┼───────────────────────┐
        │                       │                       │
        ▼                       ▼                       ▼
┌───────────────┐       ┌───────────────┐       ┌───────────────┐
│   THREAD A    │       │   THREAD B    │       │   THREAD C    │
│  Foundation   │──────▶│  UI Library   │──────▶│    Pages      │
│               │       │               │       │               │
│ 14 tasks      │       │ 9 tasks       │       │ 10 tasks      │
│ ─────────     │       │ ────────      │       │ ─────────     │
│ • Vite setup  │       │ • Button      │       │ • Dashboard   │
│ • TypeScript  │       │ • Badge       │       │ • Jobs page   │
│ • Router      │       │ • Card        │       │ • Settings    │
│ • Layout      │       │ • Input       │       │ • E2E tests   │
│ • Types       │       │ • Select      │       │               │
│ • API client  │       │ • Spinner     │       │               │
│ • Hooks       │       │ • ErrorBound  │       │               │
└───────────────┘       └───────────────┘       └───────────────┘
        │                       │                       │
        └───────────────────────┴───────────────────────┘
                                │
                                ▼
                    ┌───────────────────────┐
                    │   Integration PR      │
                    │   (all threads merge) │
                    └───────────────────────┘
```

---

## Thread A: Foundation & Data Layer

**Branch**: `react-thread-a`
**Worktree**: `/home/kcnc/code/positions-os-react-thread-a`
**Tasks**: 14 (Foundation, Types, Hooks)

### Responsibilities
- Vite + React + TypeScript setup
- React Router configuration
- Pico.css integration
- Layout components (Sidebar, Main)
- TypeScript type definitions
- API client (fetch wrapper)
- WebSocket event types
- Custom hooks (useJobs, useTargets, useStats, useWebSocket)

### Must Complete BEFORE: Threads B and C can fully start

---

## Thread B: UI Components Library

**Branch**: `react-thread-b`
**Worktree**: `/home/kcnc/code/positions-os-react-thread-b`
**Tasks**: 9 (UI Components, Polish)

### Responsibilities
- Button component (variants, sizes, loading)
- Badge component (status indicators)
- Card component (containers)
- Input component (text, search)
- Select component (dropdowns)
- Spinner & Skeleton loaders
- Error Boundary
- Pico.css theme customization

### Waits For: Thread A (Foundation)
### Provides To: Thread C (UI components for pages)

---

## Thread C: Page Implementation

**Branch**: `react-thread-c`
**Worktree**: `/home/kcnc/code/positions-os-react-thread-c`
**Tasks**: 10 (Jobs, Settings, Dashboard)

### Responsibilities
- **Dashboard**: StatsCards, RecentJobs
- **Jobs Page**: JobsTable, JobRow, FilterBar, JobDetail, pagination
- **Settings Page**: TargetForm, TargetList, TelegramAuth (QR code)
- **E2E Tests**: Playwright scenarios

### Waits For: Thread A AND Thread B
### Delivers: Functional React UI

---

## Task Files Location

All task definitions are in: `docs/tasks/react_migration/`

```
docs/tasks/react_migration/
├── 00_README.md              # Overview
├── 01_foundation/            # Thread A
│   ├── 01_01_vite_project.md
│   ├── 01_02_typescript_config.md
│   ├── 01_03_pico_css.md
│   ├── 01_04_router.md
│   └── 01_05_layout.md
├── 02_ui_components/         # Thread B
│   ├── 02_01_button.md
│   ├── 02_02_badge.md
│   ├── 02_03_card.md
│   ├── 02_04_input.md
│   ├── 02_05_select.md
│   └── 02_06_index.md
├── 03_types_and_api/         # Thread A
│   ├── 03_01_types.md
│   ├── 03_02_api_client.md
│   └── 03_03_ws_types.md
├── 04_hooks/                 # Thread A
│   ├── 04_01_query_client.md
│   ├── 04_02_use_jobs.md
│   ├── 04_03_use_targets.md
│   ├── 04_04_use_stats.md
│   ├── 04_05_use_websocket.md
│   └── 04_06_index.md
├── 05_jobs_page/             # Thread C
│   ├── 05_01_job_row.md
│   ├── 05_02_jobs_table.md
│   ├── 05_03_jobs_page.md
│   ├── 05_04_index.md
│   └── 05_05_pages_index.md
├── 06_settings_page/         # Thread C
│   ├── 06_01_target_form.md
│   ├── 06_02_target_list.md
│   └── 06_03_telegram_auth.md
├── 07_dashboard/             # Thread C
│   ├── 07_01_stats_cards.md
│   └── 07_02_recent_jobs.md
└── 08_polish/                # Thread B
    ├── 08_01_pico_css.md
    ├── 08_02_error_boundary.md
    └── 08_03_loading_states.md
```

---

## Progress Tracking

### Thread A (Foundation) - 14/33 tasks
- [ ] 01_01_vite_project
- [ ] 01_02_typescript_config
- [ ] 01_03_pico_css
- [ ] 01_04_router
- [ ] 01_05_layout
- [ ] 03_01_types
- [ ] 03_02_api_client
- [ ] 03_03_ws_types
- [ ] 04_01_query_client
- [ ] 04_02_use_jobs
- [ ] 04_03_use_targets
- [ ] 04_04_use_stats
- [ ] 04_05_use_websocket
- [ ] 04_06_index

### Thread B (UI Library) - 9/33 tasks
- [ ] 02_01_button
- [ ] 02_02_badge
- [ ] 02_03_card
- [ ] 02_04_input
- [ ] 02_05_select
- [ ] 02_06_index
- [ ] 08_01_pico_css
- [ ] 08_02_error_boundary
- [ ] 08_03_loading_states

### Thread C (Pages) - 10/33 tasks
- [ ] 05_01_job_row
- [ ] 05_02_jobs_table
- [ ] 05_03_jobs_page
- [ ] 05_04_index
- [ ] 05_05_pages_index
- [ ] 06_01_target_form
- [ ] 06_02_target_list
- [ ] 06_03_telegram_auth
- [ ] 07_01_stats_cards
- [ ] 07_02_recent_jobs

---

## Integration Strategy

1. **Thread A** merges first → `react-migration`
2. **Thread B** merges into `react-migration` (has UI components)
3. **Thread C** merges into `react-migration` (has pages)
4. Final integration tests
5. Deploy to production

---

## Acceptance Criteria

Migration is complete when:
- [ ] All 33 tasks are done
- [ ] Vite builds without errors
- [ ] All pages render correctly
- [ ] WebSocket events work
- [ ] E2E tests pass
- [ ] No console errors
- [ ] Lighthouse score > 90
- [ ] Code coverage > 80%

---

## Developer Notes

- **Package Manager**: Bun (not npm)
- **Dev Server**: `bun run dev` (port 5173)
- **Tests**: `bun run test` (Vitest)
- **E2E**: `bun run test:e2e` (Playwright)
- **Build**: `bun run build` (outputs to `static/dist/`)

See individual `THREAD_?_README.md` files for detailed instructions.

---

**Last Updated**: 2025-01-14
**Related**: `docs/phase-3.5-react-migration.md`
