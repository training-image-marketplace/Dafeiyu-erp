package com.kong.erp.ui

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
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
import androidx.compose.material3.Card
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.kong.erp.data.ApiException
import com.kong.erp.data.BillRef
import com.kong.erp.data.Dashboard
import com.kong.erp.data.ErpApi

/** 工作台：主数据计数、库存汇总、审核待办。 */
@Composable
fun DashboardScreen(
    api: ErpApi,
    refreshKey: Int,
    onOpenAudit: (direction: Int) -> Unit,
    onOpenBill: (BillRef) -> Unit,
    modifier: Modifier = Modifier,
) {
    var dashboard by remember { mutableStateOf<Dashboard?>(null) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var reload by remember { mutableIntStateOf(0) }

    LaunchedEffect(reload, refreshKey) {
        loading = true
        error = null
        try {
            dashboard = api.dashboard()
        } catch (e: ApiException) {
            error = e.message
        } finally {
            loading = false
        }
    }

    Box(modifier.fillMaxSize()) {
        val data = dashboard
        when {
            data == null && loading -> LoadingBox()
            data == null -> ErrorBox(error ?: "加载失败") { reload++ }
            else -> LazyColumn(
                contentPadding = PaddingValues(16.dp),
                verticalArrangement = Arrangement.spacedBy(12.dp),
                modifier = Modifier.fillMaxSize(),
            ) {
                item {
                    Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                        StatCard("商品", data.goodsCount.toString(), Modifier.weight(1f))
                        StatCard("客户", data.customerCount.toString(), Modifier.weight(1f))
                    }
                }
                item {
                    Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                        StatCard("供应商", data.supplierCount.toString(), Modifier.weight(1f))
                        StatCard("仓库", data.warehouseCount.toString(), Modifier.weight(1f))
                    }
                }
                item {
                    Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                        StatCard("有库存商品", data.stockGoodsCount.toString(), Modifier.weight(1f))
                        StatCard("库存总量", formatQty(data.stockTotal), Modifier.weight(1f))
                    }
                }
                item {
                    Card(Modifier.fillMaxWidth()) {
                        Column(Modifier.padding(12.dp)) {
                            Text("审核待办", style = MaterialTheme.typography.titleSmall)
                            Spacer(Modifier.height(8.dp))
                            Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                                PendingTile("入库待办", data.pendingIn, Modifier.weight(1f)) { onOpenAudit(0) }
                                PendingTile("出库待办", data.pendingOut, Modifier.weight(1f)) { onOpenAudit(1) }
                            }
                            if (data.stockAlerts > 0) {
                                Spacer(Modifier.height(8.dp))
                                Text(
                                    "库存预警：" + data.stockAlerts + " 种商品低于最低库存",
                                    color = MaterialTheme.colorScheme.error,
                                    style = MaterialTheme.typography.bodySmall,
                                )
                            }
                        }
                    }
                }
                item {
                    Text("最近待办", style = MaterialTheme.typography.titleMedium)
                }
                if (data.pending.isEmpty()) {
                    item { EmptyHint("暂无待审核单据") }
                } else {
                    items(data.pending, key = { it.billType + it.billId }) { bill ->
                        Card(Modifier.fillMaxWidth()) {
                            Column(
                                Modifier
                                    .fillMaxWidth()
                                    .clickable {
                                        if (bill.billType == "purchase" || bill.billType == "sales") {
                                            onOpenBill(BillRef(bill.billType, bill.billId))
                                        } else {
                                            onOpenAudit(if (bill.direction == "out") 1 else 0)
                                        }
                                    }
                                    .padding(12.dp),
                            ) {
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
                                    listOfNotNull(
                                        bill.partnerName.ifBlank { null },
                                        bill.warehouseName.ifBlank { null },
                                        "制单 " + bill.creatorName,
                                    ).joinToString(" · "),
                                    style = MaterialTheme.typography.bodySmall,
                                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                                )
                                Text(
                                    formatTime(bill.submittedAt),
                                    style = MaterialTheme.typography.bodySmall,
                                    color = MaterialTheme.colorScheme.outline,
                                )
                            }
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun StatCard(title: String, value: String, modifier: Modifier = Modifier) {
    Card(modifier) {
        Column(Modifier.padding(12.dp)) {
            Text(title, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
            Text(value, style = MaterialTheme.typography.headlineSmall)
        }
    }
}

@Composable
private fun PendingTile(label: String, count: Int, modifier: Modifier = Modifier, onClick: () -> Unit) {
    Column(
        modifier
            .clickable(onClick = onClick)
            .padding(vertical = 8.dp),
    ) {
        Text(
            count.toString(),
            style = MaterialTheme.typography.headlineMedium,
            color = if (count > 0) Color(0xFFE65100) else MaterialTheme.colorScheme.outline,
        )
        Text(label, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
    }
}
