# erp-system / server

大肥鱼 ERP 的后端：Go 1.25 + Gin + MySQL 8.0.16+，监听 `:8889`，库 `pt_erp`。
它同时托管前端构建产物（`web/`），所以生产只需要这一个进程。

架构、业务规则与接口清单不在本文件里：见
[后端架构](../../doc/01-后端架构.md)、[数据库设计](../../doc/04-数据库设计.md)、
[API 设计](../../doc/05-API设计.md)。

## 目录

```
cmd/server/            进程入口、手工 DI
internal/handler/      HTTP 参数与响应
internal/service/      业务规则、事务
internal/repository/   SQL、迁移
internal/domain/       实体与领域错误
internal/perm/         权限码 + 页面/按钮授权树
internal/middleware/   认证、CSRF、限流、审计
internal/httpserver/   路由注册与静态资源
config/                配置样例（config.yaml 已 gitignore）
```

## 快速开始

```bash
# 建库（应用只跑迁移，不负责 CREATE DATABASE）
#   CREATE DATABASE pt_erp DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;

cp config/config.example.yaml config/config.yaml   # 改数据库连接
go run ./cmd/server                                # :8889
```

默认账号 `admin / admin123`（首次登录后请改）。演示账号见
[仓库 README](../../README.md)。

## 环境变量

优先级：环境变量 > 配置文件 > 内置默认值。

| 变量 | 默认 | 说明 |
| --- | --- | --- |
| `APP_ADDR` | `:8889` | 监听地址 |
| `APP_ENV` | `prod` | `dev` 开 gin debug |
| `APP_DB_HOST` / `PORT` / `USER` / `PASSWORD` / `NAME` | — | 数据库连接（UTC） |
| `APP_WEB_DIR` | `web` | 前端产物目录 |
| `APP_UPLOAD_DIR` | `uploads` | 上传落盘目录 |
| `APP_SECURE_COOKIE` | false | HTTPS 下必须 true |
| `APP_TRUSTED_PROXIES` | 空 | 放在 nginx 后必须填 |
| `INTERNAL_TOKEN` | 空 | `/internal/*` 令牌；为空只允许 127.0.0.1 |

完整清单见 `config/config.example.yaml`。

## 测试

```bash
export TEST_MYSQL_DSN='root:***@tcp(127.0.0.1:23306)/'
go test ./... -count=1          # 207 个用例；未设置 DSN 时集成测试自动 skip
```

每个用例建独立随机库、结束 DROP。

## 约定（改代码前必读）

- 分层：handler 不碰 `*sql.DB`，service 不拼 SQL；错误用 `domain.ErrXxx` 由 `httpx` 统一映射。
- 每个路由必须声明权限码（`reg.protect(...)`），新增权限码要挂进 `internal/perm/buttons.go`
  并补前端 `perm.*` 双语键。
- 新增页面必须同时有 `web/src/pages/**.svelte`、`sys_menus` 种子与 `menu.*` 双语键。
- 迁移只向前：`NNNN_描述.sql`，写完把 `repository/migrate_test.go` 的 `migrationCount` 加一。
- 单据/库存的状态变更必须在事务里 `SELECT ... FOR UPDATE`，并写操作记录。

## 许可证

MIT，见 [LICENSE](../../LICENSE)。
