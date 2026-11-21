// 有问题欢迎联系 hot_kun@hotmail.com
/** 与后端 DTO 一一对应的类型。字段名保持后端的 snake_case，不做本地改名。 */

import type { FieldError } from './client'

export type { FieldError } from './client'

export type Status = 'active' | 'inactive'
export type MenuType = 'directory' | 'menu'

export type User = {
  id: number
  created_at: string
  updated_at: string
  status: Status
  username: string
  nickname: string
  mobile: string
  email: string
  login_ip: string
  /** null 表示从未登录 */
  login_at: string | null
}

export type UserDetail = User & {
  role_ids: number[]
}

/** 一条绑在**文件行号**上的导入错误。row 是文件里的行号（表头算第 1 行）。 */
export type ImportRowError = FieldError & {
  row: number
}

/**
 * 导入接口的返回体，对应后端 handler.importReport。
 *
 * 逐行错误在这里而不在 ApiError.fieldErrors 里：校验失败也返回 200，
 * 因为「哪一行错了」是导入结果的核心信息，而失败响应只保留 code/msg/errors。
 */
export type ImportReport = {
  total: number
  /** 能进库的行数（一行可能同时踩多条规则） */
  valid: number
  imported: number
  errors: ImportRowError[]
}

export type Role = {
  id: number
  created_at: string
  updated_at: string
  status: Status
  code: string
  name: string
  remark: string
}

export type Grants = {
  menu_ids: number[]
  perm_codes: string[]
}

export type MenuButton = {
  code: string
  label_key: string
}

export type Menu = {
  id: number
  created_at: string
  updated_at: string
  status: Status
  /** null 表示根节点 */
  parent_id: number | null
  sort: number
  menu_type: MenuType
  /** i18n 键，不是文案 */
  title_key: string
  path: string
  component: string
  icon: string
  buttons: MenuButton[]
}

export type MenuNode = Menu & {
  children: MenuNode[]
}

export type Page<T> = {
  list: T[]
  total: number
  page: number
  page_size: number
}

/** 登录与 /auth/me 的响应。 */
export type SessionPayload = {
  user: User
  is_admin: boolean
  /** 内置管理员这里会拿到全部已声明的权限码，所以前端只需一种判断 */
  perms: string[]
  csrf_token: string
  expires_at: string
  menus: MenuNode[]
}

export type SessionView = {
  /** 用于踢人的标识。它是明文 token 的 SHA-256，反推不出 token。 */
  token_hash: string
  user_id: number
  username: string
  nickname: string
  created_at: string
  expires_at: string
  last_seen_at: string
  ip: string
  user_agent: string
}

export type LoginLog = {
  id: number
  created_at: string
  username: string
  status: 'success' | 'failed'
  /** 失败原因的 i18n 键 */
  reason: string
  ip: string
  user_agent: string
}

export type OperLog = {
  id: number
  created_at: string
  user_id: number
  username: string
  method: string
  path: string
  status: number
  /** 失败时的 i18n 键 */
  result: string
  duration_ms: number
  ip: string
  user_agent: string
}

export type FileItem = {
  id: number
  created_at: string
  group_name: string
  original_name: string
  size: number
  content_type: string
  uploader_id: number
  uploader_name: string
}

export type FileUsage = {
  used: number
  quota: number
  count: number
  /** 单文件上限，由后端配置决定 */
  max_size: number
}

export type FilePage = Page<FileItem> & {
  usage: FileUsage
}

export type SystemSnapshot = {
  os: string
  arch: string
  go_version: string
  uptime_seconds: number
  cpu: {
    /** false 表示本平台读不到（比如 macOS 没有 /proc） */
    available: boolean
    cores: number
    usage_percent: number
    load_avg: number[] | null
    load_available: boolean
  }
  memory: {
    available: boolean
    total: number
    used: number
    free: number
    usage_percent: number
  }
  disk: {
    path: string
    available: boolean
    total: number
    used: number
    free: number
    usage_percent: number
  }
  process: {
    goroutines: number
    heap_alloc: number
    heap_sys: number
    sys: number
    num_gc: number
    rss_available: boolean
    rss: number
  }
}

export type Job = {
  status: 'active' | 'inactive'
  job_key: string
  cron: string
  remark: string
  /** 说明文案的 i18n 键，来自代码注册表 */
  description_key: string
  /** 代码里注册的默认 cron，用于「恢复默认」时参考 */
  default_cron: string
  last_run_at: string | null
  /** 空串表示从未执行过 */
  last_status: '' | 'success' | 'failed' | 'skipped'
  last_error: string
  last_duration_ms: number
  next_run_at: string | null
}

export type JobLog = {
  id: number
  /** 执行开始时间，与任务列表上的「上次执行」同源 */
  created_at: string
  job_key: string
  /** cron = 调度器触发，manual = 界面上点了「立即执行」 */
  trigger: 'cron' | 'manual'
  status: 'success' | 'failed' | 'skipped'
  /** 失败原因或跳过原因，成功时为空 */
  error: string
  duration_ms: number
}

export type PermEndpoint = {
  method: string
  path: string
  /** 与 endpoints 同级冗余一份，便于前端按需索引 */
  perm: string
}

export type PermItem = {
  code: string
  label_key: string
  /** 该权限点保护的接口。后端由路由表生成，不查库。 */
  endpoints: PermEndpoint[]
}

export type PermGroup = {
  title_key: string
  perms: PermItem[]
}

export type PermCatalogue = {
  groups: PermGroup[]
}

// ---------- ERP ----------

export type Customer = {
  id: number
  created_at: string
  updated_at: string
  code: string
  name: string
  contact: string
  phone: string
  address: string
  payment_method: string
  status: Status
}

export type Supplier = Customer

export type Warehouse = {
  id: number
  created_at: string
  updated_at: string
  code: string
  name: string
  location: string
  keeper: string
  phone: string
  address: string
  status: Status
}

export type Unit = {
  id: number
  created_at: string
  updated_at: string
  name: string
  status: Status
}

export type Goods = {
  id: number
  created_at: string
  updated_at: string
  code: string
  name: string
  spec: string
  category_id: number | null
  brand_id: number | null
  unit: string
  unit_id: number | null
  price: number
  cost: number
  purchase_price: number
  sale_price: number
  min_qty: number
  status: Status
  stock_qty: number
}

export type StockRow = {
  warehouse_id: number
  warehouse_name: string
  goods_id: number
  goods_code: string
  goods_name: string
  unit: string
  qty: number
  updated_at: string
}

export type StockFlowRow = {
  goods_id: number
  goods_code: string
  goods_name: string
  unit: string
  in_qty: number
  out_qty: number
  net_qty: number
}

export type StockLog = {
  id: number
  created_at: string
  warehouse_id: number
  goods_id: number
  goods_name: string
  change_qty: number
  biz_type: string
  ref_no: string
}

export type TransferItem = {
  id: number
  goods_id: number
  goods_name: string
  unit: string
  qty: number
}

export type StockTransfer = {
  id: number
  created_at: string
  updated_at: string
  transfer_no: string
  from_warehouse_id: number
  from_warehouse_name: string
  to_warehouse_id: number
  to_warehouse_name: string
  status: OrderStatus
  remark: string
  created_by: number
  submitted_at: string | null
  approved_by: number | null
  approved_at: string | null
  completed_at: string | null
  items: TransferItem[]
}

export type StockCheckItem = {
  id: number
  goods_id: number
  goods_name: string
  unit: string
  book_qty: number
  actual_qty: number
  diff_qty: number
}

export type StockCheck = {
  id: number
  created_at: string
  updated_at: string
  check_no: string
  warehouse_id: number
  warehouse_name: string
  status: OrderStatus
  remark: string
  created_by: number
  submitted_at: string | null
  approved_by: number | null
  approved_at: string | null
  items: StockCheckItem[]
}

export type StockAdjustment = {
  id: number
  created_at: string
  updated_at: string
  adjustment_no: string
  warehouse_id: number
  warehouse_name: string
  goods_id: number
  goods_name: string
  unit: string
  direction: 'in' | 'out'
  qty: number
  reason: string
  status: OrderStatus
  created_by: number
  submitted_at: string | null
  approved_by: number | null
  approved_at: string | null
}

export type OrderItem = {
  id?: number
  goods_id: number
  goods_name?: string
  qty: number
  price: number
  amount?: number
}

export type OrderStatus = 'draft' | 'submitted' | 'approved' | 'completed' | 'rejected'

export type PurchaseOrder = {
  id: number
  created_at: string
  updated_at: string
  order_no: string
  supplier_id: number
  supplier_name: string
  warehouse_id: number
  warehouse_name: string
  status: OrderStatus
  total_amount: number
  created_by: number
  submitted_at: string | null
  approved_by: number | null
  approved_at: string | null
  items: OrderItem[]
}

export type SalesOrder = {
  id: number
  created_at: string
  updated_at: string
  order_no: string
  customer_id: number
  customer_name: string
  warehouse_id: number
  warehouse_name: string
  status: OrderStatus
  total_amount: number
  created_by: number
  submitted_at: string | null
  approved_by: number | null
  approved_at: string | null
  items: OrderItem[]
}

/** 退货单：采购退货与销售退货共用同一形状，差别在来源单与往来单位。 */
export type ReturnOrder = {
  id: number
  created_at: string
  updated_at: string
  return_no: string
  source_order_id: number
  source_order_no: string
  supplier_id?: number
  supplier_name?: string
  customer_id?: number
  customer_name?: string
  warehouse_id: number
  warehouse_name: string
  status: OrderStatus
  total_amount: number
  remark: string
  created_by: number
  submitted_at: string | null
  approved_by: number | null
  approved_at: string | null
  items?: OrderItem[]
}

/** 审核中心的一行：四类单据的统一视图。 */
/** 一条可注入的教学缺陷。wired 为 false 表示还没有业务分支，打开不会有变化。 */
export type BugSwitch = {
  code: string
  name: string
  area: string
  wired: boolean
}

export type Dashboard = {
  goods_count: number
  customer_count: number
  supplier_count: number
  warehouse_count: number
  stock_goods_count: number
  stock_total: number
  pending_total: number
  stock_alerts: number
  pending_in: number
  pending_out: number
  pending: AuditBill[]
}

export type AuditBill = {
  bill_type: AuditBillType
  bill_id: number
  bill_no: string
  partner_name: string
  warehouse_name: string
  total_amount: number
  direction: 'in' | 'out' | ''
  qty: number
  status: OrderStatus
  created_by: number
  creator_name: string
  submitted_at: string | null
  approved_by: number | null
  approver_name: string
  approved_at: string | null
}

/** 与 URL 路径一致，前端据此拼详情与审核端点。 */
export type AuditBillType =
  | 'purchase'
  | 'sales'
  | 'purchase-return'
  | 'sale-return'
  | 'stock-adjustment'
  | 'transfer'
  | 'stock-check'

export type BillOperation = {
  id: number
  created_at: string
  bill_type: string
  bill_id: number
  action: string
  operator_id: number
  operator_name: string
  comment: string
}

export type BugInfo = {
  code: string
  name: string
  area: string
}

export type Category = {
  id: number
  parent_id: number | null
  name: string
  sort: number
}

export type Brand = {
  id: number
  name: string
}
