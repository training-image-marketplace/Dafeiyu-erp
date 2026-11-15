package com.kong.erp.ui

import android.widget.Toast
import androidx.compose.foundation.clickable
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.rememberScrollState
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FilterChip
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Tab
import androidx.compose.material3.TabRow
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.kong.erp.data.ApiException
import com.kong.erp.data.AuditBill
import com.kong.erp.data.BillRef
import com.kong.erp.data.ErpApi
import com.kong.erp.data.Order
import com.kong.erp.data.OrderLine
import com.kong.erp.data.Session
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

private val statusFilters = listOf(
    "" to "全部",
    "draft" to "草稿",
    "submitted" to "已提交",
    "approved" to "已审批",
    "completed" to "已完成",
    "rejected" to "已驳回",
)

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun BillsScreen(
    api: ErpApi,
    refreshKey: Int,
    onOpenBill: (BillRef) -> Unit,
    modifier: Modifier = Modifier,
) {
    var type by remember { mutableStateOf("purchase") }
    var status by remember { mutableStateOf("") }
    var input by remember { mutableStateOf("") }
    var keyword by remember { mutableStateOf("") }
    val scope = rememberCoroutineScope()

    LaunchedEffect(input) {
        delay(400)
        if (input.trim() != keyword) keyword = input.trim()
    }

    val list = remember(type, keyword, status, refreshKey) {
        PagedList<Order> { page -> api.orders(type, keyword, status, page) }
    }
    LaunchedEffect(list) { list.reload() }

    Column(modifier.fillMaxSize()) {
        TabRow(selectedTabIndex = if (type == "purchase") 0 else 1) {
            Tab(selected = type == "purchase", onClick = { type = "purchase" }, text = { Text("采购单") })
            Tab(selected = type == "sales", onClick = { type = "sales" }, text = { Text("销售单") })
        }
        Column(Modifier.padding(horizontal = 16.dp)) {
            Spacer(Modifier.height(8.dp))
            SearchField(input, { input = it }, "单号搜索")
            Row(
                Modifier
                    .horizontalScroll(rememberScrollState())
                    .padding(vertical = 4.dp),
                horizontalArrangement = Arrangement.spacedBy(8.dp),
            ) {
                statusFilters.forEach { (value, label) ->
                    FilterChip(
                        selected = status == value,
                        onClick = { status = value },
                        label = { Text(label) },
                    )
                }
            }
        }
        when {
            list.items.isEmpty() && list.loading -> LoadingBox(Modifier.weight(1f))
            list.items.isEmpty() && list.error != null ->
                ErrorBox(list.error!!) { scope.launch { list.reload() } }
            else -> LazyColumn(
                modifier = Modifier.weight(1f),
                contentPadding = PaddingValues(start = 16.dp, end = 16.dp, bottom = 16.dp),
                verticalArrangement = Arrangement.spacedBy(8.dp),
            ) {
                if (list.items.isEmpty()) item { EmptyHint("暂无单据") }
                items(list.items, key = { it.id }) { order ->
                    OrderCard(order) { onOpenBill(BillRef(order.type, order.id)) }
                }
                item {
                    ListFooter(
                        loading = list.loading,
                        hasMore = list.hasMore,
                        error = list.error,
                        isEmpty = list.items.isEmpty(),
                        onRetry = { scope.launch { list.loadMore() } },
                        onLoadMore = { scope.launch { list.loadMore() } },
                    )
                }
            }
        }
    }
}

@Composable
private fun OrderCard(order: Order, onClick: () -> Unit) {
    Card(Modifier.fillMaxWidth()) {
        Column(
            Modifier
                .fillMaxWidth()
                .clickable(onClick = onClick)
                .padding(12.dp),
        ) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text(
                    order.no,
                    style = MaterialTheme.typography.bodyMedium,
                    fontWeight = FontWeight.Medium,
                    modifier = Modifier.weight(1f),
                )
                StatusChip(order.status)
            }
            Spacer(Modifier.height(4.dp))
            Text(
                listOf(order.partner, order.warehouse).filter { it.isNotBlank() }.joinToString(" · "),
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            Spacer(Modifier.height(4.dp))
            Row {
                Text(formatMoney(order.totalAmount), style = MaterialTheme.typography.bodyMedium)
                Spacer(Modifier.weight(1f))
                Text(
                    formatTime(order.createdAt),
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.outline,
                )
            }
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun BillDetailScreen(
    api: ErpApi,
    session: Session,
    ref: BillRef,
    onBack: () -> Unit,
    onChanged: () -> Unit,
) {
    var order by remember { mutableStateOf<Order?>(null) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var reloadKey by remember { mutableIntStateOf(0) }
    var pendingAction by remember { mutableStateOf<String?>(null) }
    var busy by remember { mutableStateOf(false) }
    var actionError by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    LaunchedEffect(ref, reloadKey) {
        loading = true
        error = null
        try {
            order = api.order(ref.type, ref.id)
        } catch (e: ApiException) {
            error = e.message
        } finally {
            loading = false
        }
    }

    val runAction: (suspend () -> Unit, Boolean) -> Unit = { block, popBack ->
        if (!busy) {
            scope.launch {
                busy = true
                actionError = null
                try {
                    block()
                    onChanged()
                    pendingAction = null
                    if (popBack) onBack() else reloadKey++
                } catch (e: ApiException) {
                    // 关掉确认框，错误显示在按钮上方
                    pendingAction = null
                    actionError = e.message
                } finally {
                    busy = false
                }
            }
        }
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text(if (ref.type == "purchase") "采购单详情" else "销售单详情") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "返回")
                    }
                },
            )
        },
    ) { padding ->
        val data = order
        val modifier = Modifier.padding(padding)
        when {
            data == null && loading -> LoadingBox(modifier)
            data == null -> ErrorBox(error ?: "加载失败") { reloadKey++ }
            else -> {
                val editable = session.can(
                    if (data.type == "purchase") "erp:purchase:edit" else "erp:sales:edit",
                )
                LazyColumn(
                    modifier = modifier.fillMaxSize(),
                    contentPadding = PaddingValues(16.dp),
                    verticalArrangement = Arrangement.spacedBy(12.dp),
                ) {
                    item {
                        Card(Modifier.fillMaxWidth()) {
                            Column(Modifier.padding(12.dp)) {
                                Row(verticalAlignment = Alignment.CenterVertically) {
                                    Text(
                                        data.no,
                                        style = MaterialTheme.typography.titleMedium,
                                        modifier = Modifier.weight(1f),
                                    )
                                    StatusChip(data.status)
                                }
                                Spacer(Modifier.height(8.dp))
                                InfoRow(if (data.type == "purchase") "供应商" else "客户", data.partner)
                                InfoRow("仓库", data.warehouse)
                                InfoRow("金额", formatMoney(data.totalAmount))
                                InfoRow("创建时间", formatTime(data.createdAt))
                                InfoRow("提交时间", formatTime(data.submittedAt))
                                InfoRow("审核时间", formatTime(data.approvedAt))
                            }
                        }
                    }
                    item { Text("明细", style = MaterialTheme.typography.titleMedium) }
                    if (data.items.isEmpty()) {
                        item { EmptyHint("暂无明细") }
                    } else {
                        items(data.items.size) { index -> ItemRow(data.items[index]) }
                    }
                    item {
                        Column {
                            if (actionError != null) {
                                Text(
                                    actionError!!,
                                    color = MaterialTheme.colorScheme.error,
                                    style = MaterialTheme.typography.bodySmall,
                                )
                                Spacer(Modifier.height(8.dp))
                            }
                            if (busy) {
                                CircularProgressIndicator(Modifier.width(24.dp), strokeWidth = 2.dp)
                            }
                            val actions = when (data.status) {
                                "draft" -> {
                                    val list = mutableListOf<String>()
                                    if (editable) list += "submit"
                                    if (editable) list += "delete"
                                    list
                                }
                                "submitted" -> if (editable) listOf("withdraw") else emptyList()
                                "approved" -> if (editable) listOf("complete") else emptyList()
                                else -> emptyList()
                            }
                            if (actions.isEmpty()) {
                                Text(
                                    "当前状态下没有可执行的操作",
                                    style = MaterialTheme.typography.bodySmall,
                                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                                )
                            }
                            actions.forEach { action ->
                                ActionButton(
                                    action = action,
                                    enabled = !busy,
                                    onClick = { pendingAction = action },
                                )
                                Spacer(Modifier.height(8.dp))
                            }
                        }
                    }
                }
            }
        }
    }

    val data = order
    if (data != null) {
        when (pendingAction) {
            "submit" -> ConfirmDialog(
                title = "提交单据",
                text = "提交后不能再修改，确定提交吗？",
                onConfirm = { runAction({ api.submitOrder(data.type, data.id) }, false) },
                onDismiss = { pendingAction = null },
            )
            "withdraw" -> ConfirmDialog(
                title = "撤销提交",
                text = "撤销后回到草稿状态，可以继续编辑。确定撤销吗？",
                onConfirm = { runAction({ api.withdrawOrder(data.type, data.id) }, false) },
                onDismiss = { pendingAction = null },
            )
            "complete" -> ConfirmDialog(
                title = if (data.type == "purchase") "入库" else "出库",
                text = "确认执行并更新库存吗？",
                onConfirm = { runAction({ api.completeOrder(data.type, data.id) }, false) },
                onDismiss = { pendingAction = null },
            )
            "delete" -> ConfirmDialog(
                title = "删除草稿",
                text = "删除后不可恢复，确定删除吗？",
                confirmText = "删除",
                onConfirm = { runAction({ api.deleteOrder(data.type, data.id) }, true) },
                onDismiss = { pendingAction = null },
            )
        }
    }
}

@Composable
private fun InfoRow(label: String, value: String) {
    Row(Modifier.fillMaxWidth().padding(vertical = 2.dp)) {
        Text(
            label,
            style = MaterialTheme.typography.bodySmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            modifier = Modifier.width(80.dp),
        )
        Text(value.ifBlank { "-" }, style = MaterialTheme.typography.bodySmall)
    }
}

@Composable
private fun ItemRow(line: OrderLine) {
    Card(Modifier.fillMaxWidth()) {
        Row(Modifier.padding(12.dp), verticalAlignment = Alignment.CenterVertically) {
            Column(Modifier.weight(1f)) {
                Text(line.goodsName, style = MaterialTheme.typography.bodyMedium)
                Text(
                    formatQty(line.qty) + " × " + formatMoney(line.price),
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
            Text(formatMoney(line.amount), style = MaterialTheme.typography.bodyMedium)
        }
    }
}

@Composable
private fun ActionButton(action: String, enabled: Boolean, onClick: () -> Unit) {
    val label = when (action) {
        "submit" -> "提交"
        "withdraw" -> "撤销提交"
        "complete" -> "执行出入库"
        "delete" -> "删除草稿"
        else -> action
    }
    if (action == "complete" || action == "submit") {
        Button(onClick = onClick, enabled = enabled, modifier = Modifier.fillMaxWidth()) { Text(label) }
    } else {
        OutlinedButton(onClick = onClick, enabled = enabled, modifier = Modifier.fillMaxWidth()) { Text(label) }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun AuditScreen(
    api: ErpApi,
    session: Session,
    direction: Int,
    onDirectionChange: (Int) -> Unit,
    refreshKey: Int,
    modifier: Modifier = Modifier,
) {
    val dir = if (direction == 0) "in" else "out"
    val canAudit = session.can("erp:audit:approve")
    var pending by remember { mutableStateOf<Pair<AuditBill, Boolean>?>(null) }
    val scope = rememberCoroutineScope()
    val context = LocalContext.current

    val list = remember(dir, refreshKey) {
        PagedList<AuditBill> { page -> api.audits(dir, page) }
    }
    LaunchedEffect(list) { list.reload() }

    val run: (AuditBill, Boolean, String) -> Unit = { bill, pass, comment ->
        scope.launch {
            try {
                api.audit(bill.billType, bill.billId, pass, comment)
                Toast.makeText(context, if (pass) "已通过" else "已驳回", Toast.LENGTH_SHORT).show()
                pending = null
                list.reload()
            } catch (e: ApiException) {
                Toast.makeText(context, e.message, Toast.LENGTH_SHORT).show()
            }
        }
    }

    Column(modifier.fillMaxSize()) {
        TabRow(selectedTabIndex = direction) {
            Tab(selected = direction == 0, onClick = { onDirectionChange(0) }, text = { Text("入库待办") })
            Tab(selected = direction == 1, onClick = { onDirectionChange(1) }, text = { Text("出库待办") })
        }
        when {
            list.items.isEmpty() && list.loading -> LoadingBox(Modifier.weight(1f))
            list.items.isEmpty() && list.error != null ->
                ErrorBox(list.error!!) { scope.launch { list.reload() } }
            else -> LazyColumn(
                modifier = Modifier.weight(1f),
                contentPadding = PaddingValues(16.dp),
                verticalArrangement = Arrangement.spacedBy(8.dp),
            ) {
                if (list.items.isEmpty()) item { EmptyHint("当前没有待审核的单据") }
                items(list.items, key = { it.billType + it.billId }) { bill ->
                    AuditCard(
                        bill = bill,
                        canAudit = canAudit,
                        onPass = { pending = bill to true },
                        onReject = { pending = bill to false },
                    )
                }
                item {
                    ListFooter(
                        loading = list.loading,
                        hasMore = list.hasMore,
                        error = list.error,
                        isEmpty = list.items.isEmpty(),
                        onRetry = { scope.launch { list.loadMore() } },
                        onLoadMore = { scope.launch { list.loadMore() } },
                    )
                }
            }
        }
    }

    pending?.let { (bill, pass) ->
        if (pass) {
            ConfirmDialog(
                title = "审核通过",
                text = "确认通过 " + bill.billNo + " 吗？",
                confirmText = "通过",
                onConfirm = { run(bill, true, "") },
                onDismiss = { pending = null },
            )
        } else {
            CommentDialog(
                title = "驳回单据",
                label = "驳回原因",
                confirmText = "驳回",
                onConfirm = { comment -> run(bill, false, comment) },
                onDismiss = { pending = null },
            )
        }
    }
}

@Composable
private fun AuditCard(
    bill: AuditBill,
    canAudit: Boolean,
    onPass: () -> Unit,
    onReject: () -> Unit,
) {
    Card(Modifier.fillMaxWidth()) {
        Column(Modifier.padding(12.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text(
                    billTypeLabel(bill.billType),
                    style = MaterialTheme.typography.labelMedium,
                    color = MaterialTheme.colorScheme.primary,
                )
                Spacer(Modifier.width(8.dp))
                Text(
                    bill.billNo,
                    style = MaterialTheme.typography.bodyMedium,
                    fontWeight = FontWeight.Medium,
                    modifier = Modifier.weight(1f),
                )
                StatusChip(bill.status)
            }
            Spacer(Modifier.height(4.dp))
            Text(
                listOf(bill.partnerName, bill.warehouseName)
                    .filter { it.isNotBlank() }
                    .joinToString(" · "),
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            Spacer(Modifier.height(4.dp))
            Text(
                "制单 " + bill.creatorName +
                    (if (bill.submittedAt != null) " · " + formatTime(bill.submittedAt) else ""),
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.outline,
            )
            Spacer(Modifier.height(8.dp))
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text(
                    when {
                        bill.totalAmount > 0 -> formatMoney(bill.totalAmount)
                        bill.qty > 0 -> formatQty(bill.qty)
                        else -> "—"
                    },
                    style = MaterialTheme.typography.bodyMedium,
                    modifier = Modifier.weight(1f),
                )
                if (canAudit) {
                    OutlinedButton(onClick = onReject) { Text("驳回") }
                    Spacer(Modifier.width(8.dp))
                    Button(onClick = onPass) { Text("通过") }
                }
            }
        }
    }
}
