# erp-system / web

大肥鱼 ERP 的前端：Svelte 5 + Vite + Tailwind + bits-ui。产物 `dist/` 由后端托管
（`APP_WEB_DIR`），生产环境不需要 nginx、也不需要 Node。

路由、组件、列表状态机与双语约定见 [前端架构](../../doc/02-前端架构.md)。

## 快速开始

```bash
npm ci
npm run dev       # :5173，/api 与 /healthz 代理到 :8889
```

> 只起前端是打不开的：`/api` 只是代理，后端没起会 502。

## 脚本

| 命令 | 作用 |
| --- | --- |
| `npm run dev` | Vite 开发服务器（:5173） |
| `npm run build` | 产出 `dist/`，交给后端托管 |
| `npm run check` | svelte-check + tsc，提交前必须 0 error |
| `npm run preview` | 本地预览构建产物 |
| `npx playwright test` | e2e（11 个 spec，自动起被测配置） |

## 目录

```
src/lib/api/          fetch 封装（CSRF、401）、类型、删除确认
src/lib/components/   app-shell、列表通用件、ui/*、erp/*
src/lib/i18n/         zh-CN / en-US（622 条），缺键由 Go 测试兜底
src/lib/stores/       session / confirm / notify / tabs
src/lib/list-state.svelte.ts   分页 + 筛选 + URL 同步
src/lib/router.svelte.ts       基于 import.meta.glob 的路由
src/pages/            每个 .svelte 自动成为一个路由（27 个页面）
e2e/                  Playwright 用例
```

## 约定

- **页面即路由**：`pages/**/*.svelte` 由 `import.meta.glob` 收集，菜单的 `component` 就是路径；
  `pages/` 下不能放复用组件（会被当成路由），组件放 `lib/components/`。
- **列表页用 `createListState`**：分页、筛选、导出、URL 同步不要重写。
- **文案一律 `t('key')`**：新键同时加 `zh-CN.ts` 与 `en-US.ts`；`en-US` 的类型约束会拦住漏键。
- **权限门控用 `session.can('...')`**，但服务端仍会再校验——前端隐藏只是体验。
- 删除用 `confirm()` / `deleteWithConfirm()`；409 的「有依赖」是确认信号，不是错误。

## 许可证

MIT，见 [LICENSE](../../LICENSE)。
