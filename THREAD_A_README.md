# Thread A: Foundation & Data Layer

## Summary

**Thread A** builds the foundation for the React migration. This includes project setup, TypeScript configuration, API client, and all custom hooks.

**Programmer Focus**: Backend integration, data fetching, state management, TypeScript types.

## Branch: `react-thread-a`

## Tasks Overview

### Phase 1: Foundation (5 tasks)
- `01_foundation/01_01_vite_project.md` - Initialize Vite + React + TypeScript
- `01_foundation/01_02_typescript_config.md` - Configure path aliases (@/ imports)
- `01_foundation/01_03_pico_css.md` - Pico.css integration
- `01_foundation/01_04_router.md` - React Router setup
- `01_foundation/01_05_layout.md` - Base layout (Sidebar + Main)

### Phase 3: Types & API (3 tasks)
- `03_types_and_api/03_01_types.md` - Core TypeScript types (Job, Target, etc.)
- `03_types_and_api/03_02_api_client.md` - API client (fetch wrapper)
- `03_types_and_api/03_03_ws_types.md` - WebSocket event types

### Phase 4: Hooks (6 tasks)
- `04_hooks/04_01_query_client.md` - React Query setup
- `04_hooks/04_02_use_jobs.md` - useJobs hook (jobs CRUD)
- `04_hooks/04_03_use_targets.md` - useTargets hook (targets CRUD)
- `04_hooks/04_04_use_stats.md` - useStats hook (dashboard stats)
- `04_hooks/04_05_use_websocket.md` - useWebSocket hook (real-time)
- `04_hooks/04_06_index.md` - Hooks barrel export

## Total: 14 tasks

## Dependencies

**Thread A must complete BEFORE Threads B and C can start:**
- Thread B needs: Foundation (layout), Types, Hooks
- Thread C needs: Foundation (router), Types, Hooks, API client

## Acceptance Criteria

Thread A is complete when:
- [ ] Vite dev server runs on `localhost:5173`
- [ ] Router navigation works (/, /jobs, /settings)
- [ ] API client can fetch jobs/targets from Go backend
- [ ] WebSocket connects and receives events
- [ ] All hooks have tests with >80% coverage
- [ ] TypeScript compiles with no errors

## Testing Strategy

```bash
cd /home/kcnc/code/positions-os-react-thread-a

# Install dependencies (using Bun)
bun install

# Run dev server
bun run dev

# Run tests
bun run test

# Type check
bun run type-check
```

## Key Files to Create

```
frontend/
├── src/
│   ├── main.tsx
│   ├── App.tsx
│   ├── lib/
│   │   ├── types.ts          # 03_01
│   │   ├── api.ts            # 03_02
│   │   └── query-client.ts   # 04_01
│   ├── hooks/
│   │   ├── useJobs.ts        # 04_02
│   │   ├── useTargets.ts     # 04_03
│   │   ├── useStats.ts       # 04_04
│   │   ├── useWebSocket.ts   # 04_05
│   │   └── index.ts          # 04_06
│   └── components/
│       └── layout/
│           ├── Sidebar.tsx   # 01_05
│           └── Main.tsx      # 01_05
├── vite.config.ts            # 01_01, 01_02
├── tsconfig.json             # 01_02
└── package.json              # 01_01
```

## Coordination with Other Threads

| Thread | Waits for Thread A | Provides to Thread A |
|--------|-------------------|---------------------|
| B      | Foundation, Types | - |
| C      | Foundation, Types, Hooks, API | - |

## Common Goal: Migrate to React

**Thread A's Role**: Build the foundation. When Thread A completes, Threads B and C can implement UI components and pages on top of a solid data layer.

---

**Last Updated**: 2025-01-14
**Branch**: `react-thread-a`
**Worktree**: `/home/kcnc/code/positions-os-react-thread-a`
