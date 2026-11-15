# 03 Android 端

> 代码：`erp-android/`。它对接的是 `erp-system/server` 的 `/api/v1`，与 Web 端共用同一套会话与权限。

## 1. 定位

移动端只覆盖**学生/仓储人员的高频动作**：看板、查商品与库存、处理单据与审核。
它不是一个完整客户端——新建/编辑单据、退货/调拨/盘点的详情页尚未实现，也没有国际化。
这样做是有意的：移动端用来在机房、仓库现场快速处理待办，录单仍在 Web 端。

## 2. 技术栈与工程

| 项 | 值 |
| --- | --- |
| 语言 / UI | Kotlin + Jetpack Compose（Material 3） |
| 构建 | Gradle 8.11 / AGP 8.10.1 / Kotlin 2.x |
| SDK | compileSdk 35，minSdk 26，targetSdk 35 |
| applicationId | `com.kong.erp` |
| HTTP | `HttpURLConnection` 手写封装（无 Retrofit，减少依赖） |

```
app/src/main/java/com/kong/erp/
├── MainActivity.kt          # 入口，装主题
├── data/ErpApi.kt           # HTTP 客户端、会话持久化、数据模型
└── ui/
    ├── ErpApp.kt            # 根状态：登录页 / 底部导航 / 页面切换
    ├── Components.kt        # 分页列表状态、通用组件、格式化
    ├── DashboardScreen.kt   # 工作台
    ├── GoodsStockScreens.kt # 商品、库存
    ├── BillScreens.kt       # 单据列表 / 详情 / 审核中心
    └── theme/               # 颜色、字体、主题
```

## 3. 会话与安全

- 登录走 `POST /auth/login`，成功后把 **会话 cookie** 与 **CSRF token** 存进
  `SharedPreferences`；启动时用 `GET /auth/me` 校验并恢复。
- 写请求自动带 `X-CSRF-Token`；收到 401 清本地会话回登录页。
- 服务器地址由用户在登录页填写，默认 `http://10.0.2.2:8889`（模拟器访问宿主机）。
  **https 永远允许；明文 http 只允许本机与内网**（`localhost` / `::1` / `10.0.2.2` /
  `10/8`、`172.16–31`、`192.168/16`、`127/8`），公网地址必须走 TLS——
  否则口令与会话 cookie 会在可嗅探网络上裸奔（`isAllowedBaseUrl`）。
- 明文 HTTP 通过 `usesCleartextTraffic="true"` 放行，**仅用于局域网联调**；
  生产应改用 HTTPS 并关闭。

## 4. 覆盖的功能

| 模块 | 能力 |
| --- | --- |
| 登录 | 登录、自动恢复会话、退出 |
| 工作台 | 主数据计数、库存汇总、审核待办、最近待办 |
| 商品 | 列表、关键字搜索、分页 |
| 库存 | 余额列表、关键字搜索、分页 |
| 单据 | 采购/销售列表、状态筛选、详情（含明细） |
| 单据动作 | 草稿提交 / 撤销 / 删除，审批通过 / 驳回（含原因），入库 / 出库 |
| 审核中心 | 入库待办 / 出库待办，通过 / 驳回 |

按钮按权限码显示（`erp:*:edit`、`erp:audit:approve`），**服务端仍会二次校验**——
前端隐藏只是体验，不是安全边界。

## 5. 联调

```bash
# 后端
cd erp-system/server && go run ./cmd/server      # :8889

# 模拟器：默认 10.0.2.2 即可；连不通时用端口反向映射（最稳）
adb reverse tcp:8889 tcp:8889
# 然后在登录页填 http://127.0.0.1:8889

# 真机（同一局域网）：填电脑的局域网 IP，并允许本机 Go 进程通过防火墙
ipconfig getifaddr en0
```

`10.0.2.2` 依赖模拟器自身的宿主机转发，新版 emulator（netsimd）或 macOS 本地网络权限
拦截时会连不通，表现就是「网络连接失败」——此时 `adb reverse` 是首选。

## 6. 已知限制

- 未实现新建/编辑单据、退货/调拨/盘点详情、站内信、国际化；
- 会话 cookie 存在 SharedPreferences（未加密），共享设备上要由系统账号隔离兜底；
- 未做离线缓存，断网只能看到错误提示；
- 依赖 AGP 8.10.1 / compileSdk 35；模板自带的更新版本要求 AGP 9.1，暂不跟进。
