# erp-android

实训 ERP 的 Android 客户端（Jetpack Compose）。对接 `erp-system/server` 的 `/api/v1` 接口，
沿用服务端的 Cookie 会话 + CSRF 认证。

## 功能

- 登录 / 自动恢复会话 / 退出
- 工作台：主数据计数、库存汇总、审核待办、最近待办
- 商品：列表、关键字搜索、分页
- 库存：余额列表、关键字搜索、分页
- 单据：采购单 / 销售单列表，状态筛选；详情查看明细；草稿提交、撤销、删除，审批通过 / 驳回，入库 / 出库
- 审核中心：入库待办 / 出库待办，通过 / 驳回（含驳回原因）
- 操作按钮按权限码显示（`erp:*:edit`、`erp:audit:approve`），服务端仍会二次校验

## 运行

1. 启动 ERP 后端（默认 `http://127.0.0.1:8889`）：

   ```bash
   cd erp-system/server && go run ./cmd/server
   ```

2. 用 Android Studio 打开 `erp-android/`，或命令行构建：

   ```bash
   ./gradlew assembleDebug
   ```

3. 安装 APK 后，在登录页填写服务器地址：

   - 模拟器访问宿主机：`http://10.0.2.2:8889`（默认值）
   - 真机：`http://<电脑局域网 IP>:8889`

   默认账号见 `erp-system/server/README.md`（开发种子为 `admin / admin123`）。

### 模拟器 / 真机连不上宿主机

模拟器里的 `10.0.2.2` 依赖模拟器自身的宿主机转发；新版 emulator（netsimd）或 macOS 防火墙/本地网络权限拦截时，
这条链路可能连不通，表现为「网络连接失败」。

最稳的联调方式是 `adb reverse`（模拟器与 USB 真机都适用）：

```bash
adb reverse tcp:8889 tcp:8889
```

然后在登录页把服务器地址改成 `http://127.0.0.1:8889`。该端口转发只对当前 adb 连接有效，断开后需要重新执行。

真机走局域网时，还要确认：

- 手机与电脑在同一网段，电脑当前网卡 IP 用 `ipconfig getifaddr en1`（macOS）确认；
- macOS「系统设置 → 网络 → 防火墙」放行 Go 进程，且「本地网络」权限允许终端/GoLand。

## 代码结构

```
app/src/main/java/com/kong/erp/
├── MainActivity.kt
├── data/ErpApi.kt        # HTTP 客户端、会话持久化、数据模型
└── ui/
    ├── ErpApp.kt         # 根状态、登录页、底部导航
    ├── Components.kt     # 分页列表状态、通用组件、格式化
    ├── DashboardScreen.kt
    ├── GoodsStockScreens.kt
    └── BillScreens.kt    # 单据列表 / 详情 / 审核中心
```

## 说明与限制

- 会话 cookie 与 CSRF 令牌保存在 SharedPreferences 中，启动时通过 `/auth/me` 校验并恢复。
- 明文 HTTP 已通过 `usesCleartextTraffic="true"` 放行，便于局域网联调；生产环境应改用 HTTPS 并关闭。
- 当前只覆盖移动端高频操作；新建 / 编辑单据、退货 / 调拨 / 盘点详情、站内信、国际化文案尚未实现。
- `gradle/libs.versions.toml` 里的 AndroidX 版本与 AGP 8.10.1 / compileSdk 35 对齐（模板自带的更新版本要求 AGP 9.1）。
