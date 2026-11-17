# 05 API 设计

> 代码即契约：路由表在 `erp-system/server/internal/httpserver/router.go`，
> 每条路由都显式声明权限码。

## 1. 通用约定

- 前缀 `/api/v1`；内部接口前缀 `/internal`（仅内网 + 令牌）。
- 编码 JSON / UTF-8；时间 RFC3339 UTC（`2026-10-25T04:00:00Z`）。
- 认证：`Cookie: erp_session=<token>`；写请求额外带 `X-CSRF-Token`。
  Cookie 名与平台（`pt_session`）刻意不同：浏览器 Cookie 不区分端口，同名会互相覆盖。
- 响应信封：

```json
{"code":200,"msg":"ok","data":{...}}
{"code":404,"msg":"error.notFound"}
{"code":409,"msg":"error.duplicate","errors":[{"field":"username","rule":"unique"}]}
```

- `msg` 是 i18n 键而不是文案，由前端翻译；`code` 恒等于 HTTP 状态码。
- 分页：`?page=1&page_size=20`（默认 20，上限 100，越界自动钳制）。
- 权限不足 403 `error.forbidden`；`admin` 角色隐式全量。

### 错误码

| HTTP | msg 键 | 场景 |
| --- | --- | --- |
| 400 | `error.invalidId` / `error.invalidValue` / `error.invalidState` | 参数非法 / 状态机不允许 |
| 401 | `error.unauthorized` / `error.badCredentials` | 未登录 / 密码错 |
| 403 | `error.forbidden` / `error.csrfInvalid` / `error.protected` | 无权限 / CSRF / 受保护 |
| 404 | `error.notFound` | 资源不存在 |
| 409 | `error.duplicate` / `error.hasDependents` / `error.returnQtyExceeded` | 冲突 / 有依赖 / 超退 |
| 422 | `error.validationFailed` | 字段校验 |
| 429 | `error.tooManyRequests` / `error.accountLocked` | 限流 / 锁定 |

## 2. 权限码

```
erp:master:list/edit     基础档案（商品/客户/供应商/仓库/分类/品牌/单位）
erp:stock:list/edit      库存、调整、调拨、盘点、库存报表
erp:purchase:list/edit   采购单
erp:sales:list/edit      销售单
erp:return:list/edit     两类退货
erp:audit:list/approve   审核中心（所有通过/驳回动作共用 approve）
erp:switch:list/edit     ERP 自身的开关调试入口
system:*                 用户/角色/菜单/权限点
monitor:* / tool:*       监控与工具
```

新增权限码必须同时：声明常量、挂到 `internal/perm/buttons.go` 的页面授权树、
补前端 `perm.*` 双语键——否则 `TestEveryDeclaredPermIsUsedByARoute` / `TestFrontendDictCoversPermKeys` 失败。

## 3. 基础档案

| 方法 | 路径 | 权限 | 说明 |
| --- | --- | --- | --- |
| GET/POST | `/goods`、`/customers`、`/suppliers`、`/warehouses` | `erp:master:list` / `edit` | 列表/新增 |
| PUT/DELETE | `/{resource}/:id` | `erp:master:edit` | 编辑/删除（被引用时 409） |
| GET/POST | `/categories`、`/brands`、`/units` | `erp:master:list` / `edit` | 分类/品牌/单位 |
| PUT/DELETE | `/categories/:id`、`/brands/:id`、`/units/:id` | `erp:master:edit` | 改名/删除（被商品引用时 409） |

## 4. 库存

| 方法 | 路径 | 权限 | 说明 |
| --- | --- | --- | --- |
| GET | `/stocks` | `erp:stock:list` | 库存余额 |
| GET | `/stocks?logs=1` | `erp:stock:list` | 出入库流水（同一端点切换） |
| GET/POST | `/stock-adjustments` | `erp:stock:list` / `edit` | 调整单：`{warehouse_id, goods_id, direction: in/out, qty>0, reason}`，创建即草稿，不动库存 |
| GET | `/stock-adjustments/:id` | `erp:stock:list` | 详情（含 direction/qty/unit） |
| POST | `/stock-adjustments/:id/submit` | `erp:stock:edit` | 提交进审核中心 |
| DELETE | `/stock-adjustments/:id` | `erp:stock:edit` | 删除草稿 |
| GET/POST | `/transfers` | `erp:stock:list` / `edit` | 调拨单：`{from_warehouse_id, to_warehouse_id, items:[{goods_id,qty}]}` |
| GET | `/transfers/:id` | `erp:stock:list` | 详情 |
| POST | `/transfers/:id/submit` | `erp:stock:edit` | 提交 |
| POST | `/transfers/:id/approve` / `reject` | `erp:audit:approve` | 审核，不动库存 |
| POST | `/transfers/:id/complete` | `erp:stock:edit` | 完成：调出仓扣、调入仓加，写两条流水 |
| DELETE | `/transfers/:id` | `erp:stock:edit` | 删除草稿 |
| GET/POST | `/stock-checks` | `erp:stock:list` / `edit` | 盘点单：`{warehouse_id, items:[{goods_id, actual_qty}]}`，账面数服务端读 |
| GET | `/stock-checks/:id` | `erp:stock:list` | 详情（含 book_qty/actual_qty/diff_qty） |
| POST | `/stock-checks/:id/submit` | `erp:stock:edit` | 提交 |
| DELETE | `/stock-checks/:id` | `erp:stock:edit` | 删除草稿 |
| GET | `/reports/stock-flow?from=&to=&warehouse_id=` | `erp:stock:list` | 出入库统计（按商品汇总 in/out/net） |
| GET | `/reports/stock-summary?warehouse_id=&keyword=` | `erp:stock:list` | 库存汇总 |

## 5. 采购 / 销售

| 方法 | 路径 | 权限 | 说明 |
| --- | --- | --- | --- |
| GET/POST | `/purchases` | `erp:purchase:list` / `edit` | 列表/新建 |
| GET/PUT | `/purchases/:id` | `erp:purchase:list` / `edit` | 详情/编辑（仅 draft） |
| POST | `/purchases/:id/submit` | `erp:purchase:edit` | 提交 |
| POST | `/purchases/:id/approve` | `erp:audit:approve` | 审批，可选 `{"comment":""}` |
| POST | `/purchases/:id/reject` | `erp:audit:approve` | 驳回，`{"comment":"原因"}` 必填 |
| POST | `/purchases/:id/withdraw` | `erp:purchase:edit` | 撤销提交，回 draft |
| POST | `/purchases/:id/receive` | `erp:purchase:edit` | 入库（写库存 + 流水） |
| DELETE | `/purchases/:id` | `erp:purchase:edit` | 删除草稿（操作记录保留） |
| GET/POST | `/sales` | `erp:sales:list` / `edit` | 销售单，动作同上（出库为 `/ship`） |

## 6. 退货

| 方法 | 路径 | 权限 | 说明 |
| --- | --- | --- | --- |
| GET/POST | `/purchase-returns` | `erp:return:list` / `edit` | 列表/新建（`source_order_id` + 明细 `qty`） |
| GET/PUT | `/purchase-returns/:id` | `erp:return:list` / `edit` | 详情/编辑（仅 draft） |
| POST | `/purchase-returns/:id/submit` | `erp:return:edit` | 提交（事务内复核可退数量） |
| POST | `/purchase-returns/:id/approve` / `reject` | `erp:audit:approve` | 审核 |
| POST | `/purchase-returns/:id/withdraw` | `erp:return:edit` | 撤销提交 |
| POST | `/purchase-returns/:id/complete` | `erp:return:edit` | 退货出库（扣库存 + 流水） |
| DELETE | `/purchase-returns/:id` | `erp:return:edit` | 删除草稿 |
| GET/POST | `/sales-returns` | 同上 | 销售退货，完成时入库（加库存） |

退货单价一律取来源单明细，请求只传 `goods_id` 与 `qty`；超退返回 409
`error.returnQtyExceeded`。

## 7. 审核中心与仪表盘

| 方法 | 路径 | 权限 | 说明 |
| --- | --- | --- | --- |
| GET | `/audits?direction=in&status=&keyword=` | `erp:audit:list` | 聚合待办：`in` = 采购 + 销售退货 + 入库调整 + 调拨 + 盘点；`out` = 销售 + 采购退货 + 出库调整 + 调拨 + 盘点 |
| POST | `/audits/:type/:id` | `erp:audit:approve` | `{"pass":true,"comment":""}`；驳回 `pass=false` 且 comment 必填；`type` 为 `purchase`/`sales`/`purchase-return`/`sale-return`/`stock-adjustment`/`transfer`/`stock-check` |
| GET | `/bills/:type/:id/operations` | `erp:master:list` | 单据操作历史 |
| GET | `/dashboard` | 仅需登录 | 主数据计数、库存汇总、最近 5 条待办 |

仪表盘挂在「仅需登录」而不是权限码上：它只读、只返回计数与待办的公开字段；
业务列表与操作接口仍按权限码拒绝。

## 8. 开关调试

| 方法 | 路径 | 权限 | 说明 |
| --- | --- | --- | --- |
| GET | `/switches` | `erp:switch:list` | 读当前生效开关（作用域固定 all） |
| POST | `/switches/bug` | `erp:switch:edit` | `{"bug_code":"BUG-002","enabled":true}`，读改写 all 作用域里的单个缺陷 |
| GET | `/audit/switches` | `erp:switch:list` | 列出库里所有作用域的配置，供排查 |

## 9. 系统接口

`/users`、`/roles`、`/menus`、`/perms`、`/jobs`、`/files`、`/sessions`、
`/login-logs`、`/oper-logs`、`/system` 与 mini-ruoyi 骨架同构：

- 每个写请求都要 CSRF 头；`/files` 下载走鉴权接口，文件不静态挂载；
- 导入/导出复用列表权限（能看到列表就能导出同一份数据）；
- 重设密码单独授权（能冒充任意用户，比改资料危险）。

## 10. 内部接口

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/internal/switches?scope=&scope_id=` | 读当前生效开关 |
| PUT | `/internal/switches` | 覆盖某作用域开关 |
| GET | `/internal/bugs` | 缺陷清单 |
| GET | `/internal/menus` | 可关闭菜单清单（`[{code,name,name_en}]`），平台据此渲染开关面板 |
| GET | `/internal/health` | 存活 |

鉴权：`X-Internal-Token`，常量时间比较；未配置令牌时只允许 `127.0.0.1`。
