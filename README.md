# 大肥鱼 ERP

一套能直接跑起来的进销存系统：主数据、库存、采购/销售、退货、调拨、盘点、审核中心，
外加一个 Android 客户端。前后端一体，一个镜像即可运行。

它还有另一个身份：**大肥鱼项目实训平台的参考被测系统（SUT）**——打成镜像放进镜像广场，
学生一键拿到一台属于自己的 ERP，在真实系统里做测试任务。平台与它的耦合只有三条契约
（见 §6），其余互不依赖，所以这个仓库可以单独交付、单独升级、单独发版。

## 1. 组成

| 目录 | 内容 | 技术 | 端口 |
| --- | --- | --- | --- |
| `erp-system/server/` | 后端（含静态资源托管） | Go 1.25 + Gin + MySQL 8.0.16+ | 8889 |
| `erp-system/web/` | 前端 | Svelte 5 + Vite + Tailwind + bits-ui | 构建产物由后端托管 |
| `erp-android/` | Android 客户端 | Kotlin + Jetpack Compose | — |
| `deploy/` | 独立部署（MySQL + 本镜像） | Docker Compose | 23307 / 8889 |

## 2. 功能

- **主数据**：商品（分类/品牌/单位）、客户、供应商、仓库，被引用时禁止删除（409）。
- **库存**：余额 + 流水（7 种 `biz_type`），手工调整、调拨、盘点都以单据形式入账。
- **单据**：采购入库、销售出库、采购退货、销售退货，统一的
  `draft → submitted → approved → completed` 状态机，驳回必须写原因，操作全程留痕。
- **审核中心**：入库待办 / 出库待办两个页面，聚合 7 类单据的通过/驳回。
- **仪表盘**：主数据计数、库存汇总、最近待办。
- **开关**：缺陷注入（6 个编码，4 个实际生效）与 11 个菜单开关，按需打开。
- **系统能力**：认证 + RBAC + 按钮级权限、审计日志、定时任务、文件上传、CSV 导入导出。

## 3. 快速开始

### 3.1 建库

```sql
CREATE DATABASE pt_erp DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
CREATE USER 'pt_app'@'%' IDENTIFIED BY '<强密码>';
GRANT ALL PRIVILEGES ON pt_erp.* TO 'pt_app'@'%';
```

> 不要用 `erp_system`：那是另一个项目的现网库。迁移由服务启动时自动执行，应用不负责建库。

### 3.2 跑起来

```bash
cd erp-system/server
cp config/config.example.yaml config/config.yaml   # 改数据库连接
go run ./cmd/server                                 # :8889

cd erp-system/web
npm ci && npm run dev                               # :5173，/api 代理到 8889
```

默认账号 `admin / admin123`；演示账号（种子数据，仅供教学，部署后立即改）：

| 账号 | 密码 | 角色 | 职责 |
| --- | --- | --- | --- |
| `admin` | `admin123` | admin | 全部权限 |
| `manager` | `manager123` | warehouse_manager | 审核 + 库存 |
| `keeper` | `keeper123` | warehouse_keeper | 收发与库存 |
| `buyer` | `buyer123` | buyer | 采购 + 主数据 |
| `seller` | `seller123` | seller | 销售 + 主数据 |


### 3.3 用 Compose 起一套

```bash
cd deploy
cp .env.example .env      # 改掉 MYSQL_ROOT_PASSWORD 与 INTERNAL_TOKEN
docker compose up -d --build
```

它只部署本系统（MySQL + ERP），不包含实训平台。详见 [deploy/README.md](deploy/README.md)。

### 3.4 Android

```bash
cd erp-android
adb reverse tcp:8889 tcp:8889     # 模拟器/真机最稳的联调方式
./gradlew assembleDebug
```

登录页服务器地址填 `http://127.0.0.1:8889`（走局域网时换成电脑 IP）。详见
[doc/03-Android端.md](doc/03-Android端.md)。

## 4. 测试

```bash
export TEST_MYSQL_DSN='root:***@tcp(127.0.0.1:23306)/'

cd erp-system/server && go test ./... -count=1          # 214 个用例，DB 用例自动建随机库
cd erp-system/web    && npm run check && npm run build  # svelte-check + 构建
cd erp-system/web    && npx playwright test             # e2e（需先起后端）
```

未设置 `TEST_MYSQL_DSN` 时集成测试自动 skip，纯单元测试照常跑。

## 5. 文档

| 文档 | 内容 |
| --- | --- |
| [doc/01-后端架构](doc/01-后端架构.md) | 分层、请求链路、权限模型、单据与库存规则、开关注入 |
| [doc/02-前端架构](doc/02-前端架构.md) | Svelte 5 路由、菜单、列表状态机、双语、页面清单 |
| [doc/03-Android端](doc/03-Android端.md) | Compose 客户端结构、会话、联调与限制 |
| [doc/04-数据库设计](doc/04-数据库设计.md) | `pt_erp` 表清单、关键 DDL、状态机、迁移策略 |
| [doc/05-API设计](doc/05-API设计.md) | 通用约定、权限码、全部接口 |
| [doc/06-缺陷目录与开关](doc/06-缺陷目录与开关.md) | 6 个缺陷码、11 个菜单开关、作用域与下发链路 |
| [doc/07-镜像与部署](doc/07-镜像与部署.md) | 单镜像构建、镜像广场启动命令、每实例独立库 |
| [doc/08-与参考实现对照](doc/08-与参考实现对照.md) | 与 ContainerManagement 版 ERP 的差异与取舍 |

约定（后端/前端/数据库/提交规范）见 [erp-system/server/README.md](erp-system/server/README.md)
与各子目录 README。

## 6. 与实训平台的契约

平台把它当成「任意一个被测系统」来管，只依赖三件事：

1. **HTTP 服务**：镜像 `EXPOSE` 一个端口（本项目 `8889`），平台按端口映射开放访问。
2. **内部令牌**：`/internal/*` 校验 `X-Internal-Token`。平台按实例派生并注入
   `INTERNAL_TOKEN`（同时改写启动命令里的同名赋值），用于下发缺陷/菜单开关。
3. **库名与数据目录**：镜像广场通过 `command` 注入 `APP_DB_NAME` / `APP_UPLOAD_DIR`，
   平台创建实例时会改写成该实例自己的库名与挂载目录。

不做这三件事也不影响单独使用——只是平台无法开关它、也无法给每个学生独立的库。

## 7. 许可证

**MIT**，见 [LICENSE](LICENSE)。可以自由使用、修改、分发、商用，保留版权与许可声明即可。

- 版权：Copyright (c) 2026 Dafeiyu。
- 本项目以 [mini-ruoyi](https://github.com/Kun-GitHub/mini-ruoyi) 的 RBAC 骨架重写，
  该骨架同样是 MIT（Copyright (c) 2025 Kong），声明已并入 LICENSE。
- **实训平台本体不开源**（按校授权私有化部署）；开源的只有镜像广场里的项目，也就是本仓库。

## 8. 商标

「**大肥鱼**」与「**大肥鱼 ERP**」是本项目的商标（品牌名称）。镜像广场里的项目以 MIT
开源，代码可以自由使用、修改、分发，但**名称不是**：

- 衍生或二次开发的项目请起自己的名字，不要使用「大肥鱼」或「大肥鱼 ERP」作为产品名；
- 可以如实说明「基于大肥鱼 ERP 构建」，但不要暗示官方背书；
- 平台（大肥鱼项目实训平台）不开源，其名称同理。

## 9. 目录结构

```
Dafeiyu-erp/
├── README.md
├── doc/                     # 本组文档
├── erp-system/
│   ├── Dockerfile           # 前后端单镜像（node 构建 → go 编译 → alpine）
│   ├── server/              # Go 服务，:8889，库 pt_erp
│   └── web/                 # Svelte 5 前端
└── erp-android/             # Compose 客户端
```

## 10. 相关链接

- 平台官网：[training.wkong.kdns.fr](https://training.wkong.kdns.fr/)
- 镜像广场：[training.wkong.kdns.fr/#market](https://training.wkong.kdns.fr/#market)
- 帮助文档：[wkong.kdns.fr/docs/training-platform](https://wkong.kdns.fr/docs/training-platform/)

本仓库是镜像广场里的一个项目，平台本体不开源（见 §7、§8）。
