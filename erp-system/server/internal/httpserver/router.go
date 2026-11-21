// 有问题欢迎联系 hot_kun@hotmail.com
package httpserver

import (
	"database/sql"
	"fmt"
	"net/http"
	"reflect"
	"time"

	"github.com/gin-gonic/gin"

	"practical-training/erp-system/internal/handler"
	"practical-training/erp-system/internal/httpx"
	"practical-training/erp-system/internal/middleware"
	"practical-training/erp-system/internal/perm"
)

const (
	defaultRateLimitRPS   = 20
	defaultRateLimitBurst = 40

	rateLimitTTL = 3 * time.Minute
	maxBodyBytes = 1 << 20 // 1MiB
)

// Deps 是路由装配所需的依赖。
type Deps struct {
	DB            *sql.DB
	Auth          *handler.AuthHandler
	User          *handler.UserHandler
	Role          *handler.RoleHandler
	Menu          *handler.MenuHandler
	Perm          *handler.PermHandler
	Job           *handler.JobHandler
	File          *handler.FileHandler
	Profile       *handler.ProfileHandler
	Monitor       *handler.MonitorHandler
	ERP           *handler.ERPHandler
	Internal      *handler.InternalHandler
	InternalToken string
	Session       *handler.SessionHandler
	Log           *handler.LogHandler
	// OperLog 由 service.LogService 实现，用于记录写操作审计
	OperLog middleware.OperLogRecorder
	AuthMW  *middleware.Auth
	WebDir  string
	// TrustedProxies 为空表示不信任任何反向代理（默认）。
	// 部署在 nginx 后面时必须填上代理地址，否则按 IP 限流会退化成全局限流。
	TrustedProxies []string
	// RateLimitRPS / RateLimitBurst 为 0 时用默认值（20 rps / burst 40）。
	RateLimitRPS   float64
	RateLimitBurst int
}

// validate 保证依赖已全部注入。
//
// 少注入一个 handler 的表现是「某些接口 panic 后被 Recovery 兜成 500」，
// 排查起来会往业务逻辑上找，实际只是装配遗漏。所以在启动阶段直接拒绝。
func (d Deps) validate() error {
	v := reflect.ValueOf(d)
	for i := 0; i < v.NumField(); i++ {
		if f := v.Field(i); f.Kind() == reflect.Ptr && f.IsNil() {
			return fmt.Errorf("Deps.%s 未注入（装配遗漏，会导致对应接口 500）", v.Type().Field(i).Name)
		}
	}
	return nil
}

// NewRouter 组装路由，同时返回路由表供审计与测试断言。
//
// 依赖不完整或前端产物目录不可用时直接返回错误，让进程启动即失败，
// 而不是等用户访问了才发现。
func NewRouter(deps Deps) (*gin.Engine, *RouteTable, error) {
	if err := deps.validate(); err != nil {
		return nil, nil, err
	}

	// 校验错误的字段名用 json tag 输出，需在注册路由前生效
	httpx.RegisterJSONFieldNames()

	r := gin.New()
	r.Use(
		middleware.Recovery(),
		// request-id 必须早于 RequestLogger：日志是它最主要的消费方
		middleware.RequestID(),
		middleware.RequestLogger(),
		// 安全头在任何路由之前登记：因此对所有响应生效，包括 404、panic 兜底和静态页面
		middleware.SecureHeaders(),
	)

	// 默认为空 = 不信任任何 X-Forwarded-For。
	// 本服务可以直接对外监听，无条件采信该头会让客户端能伪装成任意 IP，
	// 按 IP 限流和登录 IP 记录都会失真。
	// 部署在反向代理后面时，用 APP_TRUSTED_PROXIES 声明代理地址。
	if err := r.SetTrustedProxies(deps.TrustedProxies); err != nil {
		return nil, nil, fmt.Errorf("APP_TRUSTED_PROXIES 配置无效（%v）: %w", deps.TrustedProxies, err)
	}

	r.Use(
		// /files 自己控制上限（上传要放行到 APP_UPLOAD_MAX_MB），
		// 这里把它从全局的 1 MiB 限制里排掉
		middleware.BodyLimit(maxBodyBytes, "/api/v1/files"),
		middleware.RateLimit(rateLimit(deps.RateLimitRPS, defaultRateLimitRPS), rateLimit(deps.RateLimitBurst, defaultRateLimitBurst), rateLimitTTL),
		middleware.ImmutableAssets(AssetsPrefix),
	)

	r.GET("/healthz", func(c *gin.Context) {
		if err := deps.DB.PingContext(c.Request.Context()); err != nil {
			httpx.Fail(c, http.StatusServiceUnavailable, httpx.KeyServiceUnavailable)
			return
		}
		httpx.Success(c, gin.H{"status": "up"})
	})

	v1 := r.Group("/api/v1")
	// 顺序即执行顺序：登录 → CSRF → 操作日志 → 权限校验 → 业务 handler。
	// 操作日志必须排在权限校验之前，这样 403（越权试探）也会被记下来。
	authed := v1.Group("",
		deps.AuthMW.Require(),
		deps.AuthMW.CSRF(),
		middleware.OperationLog(deps.OperLog),
	)
	// 业务菜单开关统一在这里拦：读接口与写接口用同一套判断。
	authed.Use(deps.ERP.MenuSwitchGate())
	reg := newRegistrar(v1, authed)

	// ---- 公开端点 ----
	// 只允许一个：登录接口不可能要求「已经登录」。
	// 新增公开端点会让 TestPublicEndpointsAreMinimal 失败，是刻意的减速带。
	reg.open(http.MethodPost, "/auth/login", "登录是获得会话的前提，不可能要求已登录", deps.Auth.Login)

	// ---- 仅需登录（操作自己的数据） ----
	reg.self(http.MethodGet, "/auth/me", deps.Auth.Me)
	reg.self(http.MethodPost, "/auth/logout", deps.Auth.Logout)
	// 「改自己的东西」不需要权限码：任何能登录的账号都该能改自己的密码和资料。
	// 用 self 而不是 protect，是为了让没有管理权限的账号也有自助能力。
	reg.self(http.MethodPut, "/profile", deps.Profile.Update)
	reg.self(http.MethodPut, "/profile/password", deps.Profile.ChangePassword)
	// 仪表盘是所有角色登录后的首页：只读聚合，不需要业务权限码。
	// 放在 self 分组而不是 protect，新增角色即使没配任何权限也能看到全局概览。
	reg.self(http.MethodGet, "/dashboard", deps.ERP.Dashboard)

	// ---- 菜单 ----
	reg.protect(http.MethodGet, "/menus", perm.SystemMenuList, deps.Menu.Tree)
	reg.protect(http.MethodGet, "/menus/:id", perm.SystemMenuList, deps.Menu.Get)
	reg.protect(http.MethodPost, "/menus", perm.SystemMenuAdd, deps.Menu.Create)
	reg.protect(http.MethodPut, "/menus/:id", perm.SystemMenuEdit, deps.Menu.Update)
	reg.protect(http.MethodDelete, "/menus/:id", perm.SystemMenuDelete, deps.Menu.Delete)

	// ---- 角色 ----
	// 读授权用 list 权限、写授权用 edit 权限：能看授权和能改授权是两件事。
	reg.protect(http.MethodGet, "/roles", perm.SystemRoleList, deps.Role.List)
	reg.protect(http.MethodGet, "/roles/:id", perm.SystemRoleList, deps.Role.Get)
	reg.protect(http.MethodGet, "/roles/:id/grants", perm.SystemRoleList, deps.Role.Grants)
	reg.protect(http.MethodPost, "/roles", perm.SystemRoleAdd, deps.Role.Create)
	reg.protect(http.MethodPut, "/roles/:id", perm.SystemRoleEdit, deps.Role.Update)
	reg.protect(http.MethodPut, "/roles/:id/grants", perm.SystemRoleEdit, deps.Role.SetGrants)
	reg.protect(http.MethodDelete, "/roles/:id", perm.SystemRoleDelete, deps.Role.Delete)
	// 导出复用 list：理由同用户导出（见下）
	reg.protect(http.MethodGet, "/roles/export", perm.SystemRoleList, deps.Role.Export)

	// 权限清单本身要单独授权，不能因为「能进后台」就看到全部权限点
	reg.protect(http.MethodGet, "/perms", perm.SystemPermList, deps.Perm.Catalogue)

	// ---- 用户 ----
	reg.protect(http.MethodGet, "/users", perm.SystemUserList, deps.User.List)
	reg.protect(http.MethodGet, "/users/:id", perm.SystemUserList, deps.User.Get)
	reg.protect(http.MethodPost, "/users", perm.SystemUserAdd, deps.User.Create)
	reg.protect(http.MethodPut, "/users/:id", perm.SystemUserEdit, deps.User.Update)
	reg.protect(http.MethodPut, "/users/:id/roles", perm.SystemUserEdit, deps.User.SetRoles)
	// 重设密码单独授权：它能让持有者冒充任意用户，比「改资料」危险得多
	reg.protect(http.MethodPut, "/users/:id/password", perm.SystemUserResetPwd, deps.User.ResetPassword)
	reg.protect(http.MethodDelete, "/users/:id", perm.SystemUserDelete, deps.User.Delete)
	// 导入复用「新增用户」的权限码：它做的事就是批量建号，多一个码只会
	// 多一个可以配错的开关——能导入却不能建号毫无意义，反之亦然。
	// 导出复用 list：能看到列表就能导出列表，导出的只是同一份数据的另一种格式。
	reg.protect(http.MethodGet, "/users/export", perm.SystemUserList, deps.User.Export)
	reg.protect(http.MethodPost, "/users/import", perm.SystemUserAdd, deps.User.Import)

	// ---- 系统工具 ----
	// 任务清单以**代码注册表**为准（数据库里可能残留已删除任务），所以没有新增/删除接口。
	reg.protect(http.MethodGet, "/jobs", perm.ToolJobList, deps.Job.List)
	reg.protect(http.MethodPut, "/jobs/:key", perm.ToolJobEdit, deps.Job.Update)
	// 手动触发与改 cron 分开授权：能改调度时间不等于能立刻把任务跑起来
	reg.protect(http.MethodPost, "/jobs/:key/run", perm.ToolJobRun, deps.Job.RunNow)
	// 每次执行（含被跳过的）都留一条历史，看历史只需「能看任务清单」这一条权限：
	// 执行结果本来就是列表页上的内容，不该变成一个新的授权点
	reg.protect(http.MethodGet, "/jobs/:key/logs", perm.ToolJobList, deps.Job.ListLogs)

	reg.protect(http.MethodGet, "/files", perm.ToolFileList, deps.File.List)
	// 下载也走权限校验：上传的文件**不能静态挂载**，
	// 那会绕过鉴权，任何人拼一个 URL 就能拿到别人上传的东西
	reg.protect(http.MethodGet, "/files/:id/download", perm.ToolFileList, deps.File.Download)
	reg.protect(http.MethodPost, "/files", perm.ToolFileUpload, deps.File.Upload)
	reg.protect(http.MethodDelete, "/files/:id", perm.ToolFileDelete, deps.File.Delete)

	// ---- 系统监控 ----
	reg.protect(http.MethodGet, "/system", perm.MonitorSystemList, deps.Monitor.System)
	reg.protect(http.MethodGet, "/sessions", perm.MonitorSessionList, deps.Session.List)
	reg.protect(http.MethodDelete, "/sessions/:hash", perm.MonitorSessionKick, deps.Session.Kill)
	// 强退挂在用户下：入口在用户列表上，不在会话列表里
	reg.protect(http.MethodDelete, "/users/:id/sessions", perm.MonitorSessionKick, deps.Session.KillUserSessions)

	reg.protect(http.MethodGet, "/login-logs", perm.MonitorLoginLogList, deps.Log.ListLoginLogs)
	reg.protect(http.MethodGet, "/oper-logs", perm.MonitorOperLogList, deps.Log.ListOperLogs)
	// 日志导出同样复用 list 权限，理由同上：导出的就是列表里那些行
	reg.protect(http.MethodGet, "/login-logs/export", perm.MonitorLoginLogList, deps.Log.ExportLoginLogs)
	reg.protect(http.MethodGet, "/oper-logs/export", perm.MonitorOperLogList, deps.Log.ExportOperLogs)

	// ---- ERP 业务 ----
	// 主数据
	reg.protect(http.MethodGet, "/customers", perm.ERPMasterList, deps.ERP.ListCustomers)
	reg.protect(http.MethodPost, "/customers", perm.ERPMasterEdit, deps.ERP.CreateCustomer)
	reg.protect(http.MethodPut, "/customers/:id", perm.ERPMasterEdit, deps.ERP.UpdateCustomer)
	reg.protect(http.MethodDelete, "/customers/:id", perm.ERPMasterEdit, deps.ERP.DeleteCustomer)

	reg.protect(http.MethodGet, "/suppliers", perm.ERPMasterList, deps.ERP.ListSuppliers)
	reg.protect(http.MethodPost, "/suppliers", perm.ERPMasterEdit, deps.ERP.CreateSupplier)
	reg.protect(http.MethodPut, "/suppliers/:id", perm.ERPMasterEdit, deps.ERP.UpdateSupplier)
	reg.protect(http.MethodDelete, "/suppliers/:id", perm.ERPMasterEdit, deps.ERP.DeleteSupplier)

	reg.protect(http.MethodGet, "/warehouses", perm.ERPMasterList, deps.ERP.ListWarehouses)
	reg.protect(http.MethodPost, "/warehouses", perm.ERPMasterEdit, deps.ERP.CreateWarehouse)
	reg.protect(http.MethodPut, "/warehouses/:id", perm.ERPMasterEdit, deps.ERP.UpdateWarehouse)
	reg.protect(http.MethodDelete, "/warehouses/:id", perm.ERPMasterEdit, deps.ERP.DeleteWarehouse)

	reg.protect(http.MethodGet, "/goods", perm.ERPMasterList, deps.ERP.ListGoods)
	reg.protect(http.MethodPost, "/goods", perm.ERPMasterEdit, deps.ERP.CreateGoods)
	reg.protect(http.MethodPut, "/goods/:id", perm.ERPMasterEdit, deps.ERP.UpdateGoods)
	reg.protect(http.MethodDelete, "/goods/:id", perm.ERPMasterEdit, deps.ERP.DeleteGoods)

	reg.protect(http.MethodGet, "/categories", perm.ERPMasterList, deps.ERP.ListCategories)
	reg.protect(http.MethodPost, "/categories", perm.ERPMasterEdit, deps.ERP.CreateCategory)
	reg.protect(http.MethodPut, "/categories/:id", perm.ERPMasterEdit, deps.ERP.UpdateCategory)
	reg.protect(http.MethodDelete, "/categories/:id", perm.ERPMasterEdit, deps.ERP.DeleteCategory)
	reg.protect(http.MethodGet, "/units", perm.ERPMasterList, deps.ERP.ListUnits)
	reg.protect(http.MethodPost, "/units", perm.ERPMasterEdit, deps.ERP.CreateUnit)
	reg.protect(http.MethodPut, "/units/:id", perm.ERPMasterEdit, deps.ERP.UpdateUnit)
	reg.protect(http.MethodDelete, "/units/:id", perm.ERPMasterEdit, deps.ERP.DeleteUnit)
	reg.protect(http.MethodGet, "/brands", perm.ERPMasterList, deps.ERP.ListBrands)
	reg.protect(http.MethodPost, "/brands", perm.ERPMasterEdit, deps.ERP.CreateBrand)
	reg.protect(http.MethodPut, "/brands/:id", perm.ERPMasterEdit, deps.ERP.UpdateBrand)
	reg.protect(http.MethodDelete, "/brands/:id", perm.ERPMasterEdit, deps.ERP.DeleteBrand)

	// 库存
	reg.protect(http.MethodGet, "/stocks", perm.ERPStockList, deps.ERP.ListStocks)
	reg.protect(http.MethodPost, "/stock-adjustments", perm.ERPStockEdit, deps.ERP.CreateStockAdjustment)
	reg.protect(http.MethodGet, "/stock-adjustments/:id", perm.ERPStockList, deps.ERP.GetStockAdjustment)
	reg.protect(http.MethodPost, "/stock-adjustments/:id/submit", perm.ERPStockEdit, deps.ERP.SubmitStockAdjustment)
	reg.protect(http.MethodDelete, "/stock-adjustments/:id", perm.ERPStockEdit, deps.ERP.DeleteDraftStockAdjustment)

	// 报表
	reg.protect(http.MethodGet, "/reports/stock-flow", perm.ERPStockList, deps.ERP.StockFlowReport)
	reg.protect(http.MethodGet, "/reports/stock-summary", perm.ERPStockList, deps.ERP.StockSummaryReport)

	// 库存盘点
	reg.protect(http.MethodGet, "/stock-checks", perm.ERPStockList, deps.ERP.ListStockChecks)
	reg.protect(http.MethodGet, "/stock-checks/:id", perm.ERPStockList, deps.ERP.GetStockCheck)
	reg.protect(http.MethodPost, "/stock-checks", perm.ERPStockEdit, deps.ERP.CreateStockCheck)
	reg.protect(http.MethodPost, "/stock-checks/:id/submit", perm.ERPStockEdit, deps.ERP.SubmitStockCheck)
	reg.protect(http.MethodDelete, "/stock-checks/:id", perm.ERPStockEdit, deps.ERP.DeleteDraftStockCheck)

	// 库存调拨
	reg.protect(http.MethodGet, "/transfers", perm.ERPStockList, deps.ERP.ListTransfers)
	reg.protect(http.MethodGet, "/transfers/:id", perm.ERPStockList, deps.ERP.GetTransfer)
	reg.protect(http.MethodPost, "/transfers", perm.ERPStockEdit, deps.ERP.CreateTransfer)
	reg.protect(http.MethodPost, "/transfers/:id/submit", perm.ERPStockEdit, deps.ERP.SubmitTransfer)
	reg.protect(http.MethodPost, "/transfers/:id/approve", perm.ERPAuditApprove, deps.ERP.ApproveTransfer)
	reg.protect(http.MethodPost, "/transfers/:id/reject", perm.ERPAuditApprove, deps.ERP.RejectTransfer)
	reg.protect(http.MethodPost, "/transfers/:id/complete", perm.ERPStockEdit, deps.ERP.CompleteTransfer)
	reg.protect(http.MethodDelete, "/transfers/:id", perm.ERPStockEdit, deps.ERP.DeleteDraftTransfer)

	// 采购
	reg.protect(http.MethodGet, "/purchases", perm.ERPPurchaseList, deps.ERP.ListPurchases)
	reg.protect(http.MethodGet, "/purchases/:id", perm.ERPPurchaseList, deps.ERP.GetPurchase)
	reg.protect(http.MethodPost, "/purchases", perm.ERPPurchaseEdit, deps.ERP.CreatePurchase)
	reg.protect(http.MethodPut, "/purchases/:id", perm.ERPPurchaseEdit, deps.ERP.UpdatePurchase)
	reg.protect(http.MethodPost, "/purchases/:id/submit", perm.ERPPurchaseEdit, deps.ERP.SubmitPurchase)
	reg.protect(http.MethodPost, "/purchases/:id/approve", perm.ERPAuditApprove, deps.ERP.ApprovePurchase)
	reg.protect(http.MethodPost, "/purchases/:id/reject", perm.ERPAuditApprove, deps.ERP.RejectPurchase)
	reg.protect(http.MethodPost, "/purchases/:id/withdraw", perm.ERPPurchaseEdit, deps.ERP.WithdrawPurchase)
	reg.protect(http.MethodPost, "/purchases/:id/receive", perm.ERPPurchaseEdit, deps.ERP.ReceivePurchase)
	reg.protect(http.MethodDelete, "/purchases/:id", perm.ERPPurchaseEdit, deps.ERP.DeleteDraftPurchase)

	// 销售
	reg.protect(http.MethodGet, "/sales", perm.ERPSalesList, deps.ERP.ListSales)
	reg.protect(http.MethodGet, "/sales/:id", perm.ERPSalesList, deps.ERP.GetSales)
	reg.protect(http.MethodPost, "/sales", perm.ERPSalesEdit, deps.ERP.CreateSales)
	reg.protect(http.MethodPut, "/sales/:id", perm.ERPSalesEdit, deps.ERP.UpdateSales)
	reg.protect(http.MethodPost, "/sales/:id/submit", perm.ERPSalesEdit, deps.ERP.SubmitSales)
	reg.protect(http.MethodPost, "/sales/:id/approve", perm.ERPAuditApprove, deps.ERP.ApproveSales)
	reg.protect(http.MethodPost, "/sales/:id/reject", perm.ERPAuditApprove, deps.ERP.RejectSales)
	reg.protect(http.MethodPost, "/sales/:id/withdraw", perm.ERPSalesEdit, deps.ERP.WithdrawSales)
	reg.protect(http.MethodPost, "/sales/:id/ship", perm.ERPSalesEdit, deps.ERP.ShipSales)
	reg.protect(http.MethodDelete, "/sales/:id", perm.ERPSalesEdit, deps.ERP.DeleteDraftSales)

	// 退货：建单/改单/提交/完成用 return:edit，审核动作用统一的 audit:approve
	reg.protect(http.MethodGet, "/purchase-returns", perm.ERPReturnList, deps.ERP.ListPurchaseReturns)
	reg.protect(http.MethodGet, "/purchase-returns/:id", perm.ERPReturnList, deps.ERP.GetPurchaseReturn)
	reg.protect(http.MethodPost, "/purchase-returns", perm.ERPReturnEdit, deps.ERP.CreatePurchaseReturn)
	reg.protect(http.MethodPut, "/purchase-returns/:id", perm.ERPReturnEdit, deps.ERP.UpdatePurchaseReturn)
	reg.protect(http.MethodPost, "/purchase-returns/:id/submit", perm.ERPReturnEdit, deps.ERP.SubmitPurchaseReturn)
	reg.protect(http.MethodPost, "/purchase-returns/:id/approve", perm.ERPAuditApprove, deps.ERP.ApprovePurchaseReturn)
	reg.protect(http.MethodPost, "/purchase-returns/:id/reject", perm.ERPAuditApprove, deps.ERP.RejectPurchaseReturn)
	reg.protect(http.MethodPost, "/purchase-returns/:id/withdraw", perm.ERPReturnEdit, deps.ERP.WithdrawPurchaseReturn)
	reg.protect(http.MethodPost, "/purchase-returns/:id/complete", perm.ERPReturnEdit, deps.ERP.CompletePurchaseReturn)
	reg.protect(http.MethodDelete, "/purchase-returns/:id", perm.ERPReturnEdit, deps.ERP.DeleteDraftPurchaseReturn)

	reg.protect(http.MethodGet, "/sales-returns", perm.ERPReturnList, deps.ERP.ListSalesReturns)
	reg.protect(http.MethodGet, "/sales-returns/:id", perm.ERPReturnList, deps.ERP.GetSalesReturn)
	reg.protect(http.MethodPost, "/sales-returns", perm.ERPReturnEdit, deps.ERP.CreateSalesReturn)
	reg.protect(http.MethodPut, "/sales-returns/:id", perm.ERPReturnEdit, deps.ERP.UpdateSalesReturn)
	reg.protect(http.MethodPost, "/sales-returns/:id/submit", perm.ERPReturnEdit, deps.ERP.SubmitSalesReturn)
	reg.protect(http.MethodPost, "/sales-returns/:id/approve", perm.ERPAuditApprove, deps.ERP.ApproveSalesReturn)
	reg.protect(http.MethodPost, "/sales-returns/:id/reject", perm.ERPAuditApprove, deps.ERP.RejectSalesReturn)
	reg.protect(http.MethodPost, "/sales-returns/:id/withdraw", perm.ERPReturnEdit, deps.ERP.WithdrawSalesReturn)
	reg.protect(http.MethodPost, "/sales-returns/:id/complete", perm.ERPReturnEdit, deps.ERP.CompleteSalesReturn)
	reg.protect(http.MethodDelete, "/sales-returns/:id", perm.ERPReturnEdit, deps.ERP.DeleteDraftSalesReturn)

	// 审核中心
	reg.protect(http.MethodGet, "/audits", perm.ERPAuditList, deps.ERP.ListAudits)
	reg.protect(http.MethodPost, "/audits/:type/:id", perm.ERPAuditApprove, deps.ERP.AuditBill)

	reg.protect(http.MethodGet, "/bills/:type/:id/operations", perm.ERPMasterList, deps.ERP.BillOperations)

	// 开关（ERP 自己也有一份只读/调试入口）
	reg.protect(http.MethodGet, "/switches", perm.ERPSwitchList, deps.ERP.GetSwitches)
	// 缺陷目录与当前配置分开：目录是静态声明，配置才随开关变化。
	reg.protect(http.MethodGet, "/switches/bugs", perm.ERPSwitchList, deps.ERP.ListBugSwitches)
	reg.protect(http.MethodPost, "/switches/bug", perm.ERPSwitchEdit, deps.ERP.SetBug)
	reg.protect(http.MethodGet, "/audit/switches", perm.ERPSwitchList, deps.ERP.AuditSwitches)

	// ---- 内部接口：平台调用，不经过用户会话 ----
	internal := r.Group("/internal", handler.RequireInternalToken(deps.InternalToken))
	internal.GET("/switches", deps.Internal.GetSwitches)
	internal.PUT("/switches", deps.Internal.PutSwitches)
	internal.GET("/bugs", deps.Internal.Bugs)
	internal.GET("/menus", deps.Internal.Menus)
	internal.GET("/health", deps.Internal.Health)

	if err := MountStatic(r, deps.WebDir); err != nil {
		return nil, nil, err
	}
	return r, reg.table, nil
}

// rateLimit 在未配置时回落到默认值。0 或负数都视为「没配」。
func rateLimit[T int | float64](configured, fallback T) T {
	if configured <= 0 {
		return fallback
	}
	return configured
}
