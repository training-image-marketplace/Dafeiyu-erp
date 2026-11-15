# erp-system / web

Frontend of Dafeiyu ERP: Svelte 5 + Vite + Tailwind + bits-ui. The build output (`dist/`) is served
by the backend (`APP_WEB_DIR`), so production needs neither nginx nor Node.

Routing, components, the list state machine and the i18n rules are covered in
[frontend architecture](../../doc/02-前端架构.md) (Chinese).

## Quick start

```bash
npm ci
npm run dev       # :5173, /api and /healthz proxy to :8889
```

> The frontend alone is not enough: `/api` is only a proxy, so you get 502 without the backend.

## Scripts

| Command | Purpose |
| --- | --- |
| `npm run dev` | Vite dev server (:5173) |
| `npm run build` | emit `dist/` for the backend to serve |
| `npm run check` | svelte-check + tsc; must be error-free before committing |
| `npm run preview` | preview the build locally |
| `npx playwright test` | e2e (11 specs, starts its own server) |

## Layout

```
src/lib/api/          fetch wrapper (CSRF, 401), types, delete helpers
src/lib/components/   app shell, list primitives, ui/*, erp/*
src/lib/i18n/         zh-CN / en-US (622 entries); Go tests catch missing keys
src/lib/stores/       session / confirm / notify / tabs
src/lib/list-state.svelte.ts   pagination + filters + URL sync
src/lib/router.svelte.ts       routes via import.meta.glob
src/pages/            one route per .svelte file (27 pages)
e2e/                  Playwright specs
```

## Conventions

- **A page is a route**: `pages/**/*.svelte` is collected by `import.meta.glob`, and a menu's
  `component` is that path. No shared components under `pages/` — they belong in `lib/components/`.
- **Lists use `createListState`**: do not re-implement paging, filters, export or URL sync.
- **All copy goes through `t('key')`**: add new keys to both `zh-CN.ts` and `en-US.ts`; the type of
  `en-US` catches omissions.
- **Gate buttons with `session.can('...')`**, but the server re-checks: hiding is UX, not security.
- Deletes use `confirm()` / `deleteWithConfirm()`; a 409 "has dependents" is a confirmation
  signal, not an error.

## License

MIT — see [LICENSE](../../LICENSE).
