package com.kong.erp.ui

import androidx.compose.foundation.Image
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Search
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.unit.dp
import com.kong.erp.R
import com.kong.erp.data.ApiException
import com.kong.erp.data.Page
import java.time.Instant
import java.time.ZoneId
import java.time.format.DateTimeFormatter
import java.util.Locale

/** 列表分页状态：首屏加载、加载更多、错误重试共用。 */
class PagedList<T>(private val fetch: suspend (page: Int) -> Page<T>) {
    var items by mutableStateOf<List<T>>(emptyList())
        private set
    var total by mutableStateOf(0)
        private set
    var loading by mutableStateOf(false)
        private set
    var error by mutableStateOf<String?>(null)
        private set

    private var loadedPage = 0
    val hasMore: Boolean get() = items.size < total

    suspend fun reload() {
        loadedPage = 0
        items = emptyList()
        total = 0
        loadMore()
    }

    suspend fun loadMore() {
        if (loading) return
        loading = true
        error = null
        try {
            val res = fetch(loadedPage + 1)
            items = if (loadedPage == 0) res.list else items + res.list
            total = res.total
            loadedPage += 1
        } catch (e: ApiException) {
            error = e.message
        } catch (e: Exception) {
            error = "加载失败"
        } finally {
            loading = false
        }
    }
}

@Composable
fun SearchField(
    value: String,
    onValueChange: (String) -> Unit,
    placeholder: String = "搜索",
    onSearch: () -> Unit = {},
) {
    OutlinedTextField(
        value = value,
        onValueChange = onValueChange,
        modifier = Modifier.fillMaxWidth(),
        placeholder = { Text(placeholder) },
        leadingIcon = { Icon(Icons.Default.Search, contentDescription = null) },
        singleLine = true,
        keyboardOptions = KeyboardOptions(imeAction = ImeAction.Search),
        keyboardActions = KeyboardActions(onSearch = { onSearch() }),
    )
}

/**
 * 全屏加载：吉祥物「赶路」+ 上下浮动。
 *
 * 用动画而不是转圈：这是在给用户一个「正在跑过去」的暗示，
 * 比一个无意义的 spinner 更容易让人接受等待。
 * 无障碍：图片是装饰性的（contentDescription = null），状态由下面的文字承担。
 */
@Composable
fun LoadingBox(modifier: Modifier = Modifier, text: String = "加载中…") {
    Box(modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
        Column(horizontalAlignment = Alignment.CenterHorizontally) {
            Image(
                painter = painterResource(R.drawable.whale_running),
                contentDescription = null,
                modifier = Modifier.size(64.dp),
            )
            Text(
                text,
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                modifier = Modifier.padding(top = 8.dp),
            )
        }
    }
}

@Composable
fun ErrorBox(message: String, onRetry: () -> Unit) {
    Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
        Column(horizontalAlignment = Alignment.CenterHorizontally) {
            Text(message, color = MaterialTheme.colorScheme.error)
            TextButton(onClick = onRetry) { Text("重试") }
        }
    }
}

/**
 * 列表空态：吉祥物「脑袋空白」+ 一行说明。
 *
 * 空态是最容易被忽略、也最容易被用户看见的地方：一条冷冰冰的「暂无数据」
 * 会让人怀疑是不是坏了，一张发呆的鲸鱼会让人知道「确实没有，不是出错」。
 */
@Composable
fun EmptyHint(text: String) {
    Box(Modifier.fillMaxWidth().padding(24.dp), contentAlignment = Alignment.Center) {
        Column(horizontalAlignment = Alignment.CenterHorizontally) {
            Image(
                painter = painterResource(R.drawable.whale_blank),
                contentDescription = null,
                modifier = Modifier.size(56.dp),
            )
            Text(
                text,
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                modifier = Modifier.padding(top = 8.dp),
            )
        }
    }
}

/** 列表底部：加载中 / 加载更多 / 错误 / 没有更多。 */
@Composable
fun ListFooter(
    loading: Boolean,
    hasMore: Boolean,
    error: String?,
    isEmpty: Boolean,
    onRetry: () -> Unit,
    onLoadMore: () -> Unit,
) {
    if (loading) {
        Box(Modifier.fillMaxWidth().padding(16.dp), contentAlignment = Alignment.Center) {
            CircularProgressIndicator(modifier = Modifier.size(24.dp), strokeWidth = 2.dp)
        }
        return
    }
    if (error != null) {
        Row(
            Modifier.fillMaxWidth().padding(8.dp),
            horizontalArrangement = Arrangement.Center,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(error, color = MaterialTheme.colorScheme.error, style = MaterialTheme.typography.bodySmall)
            TextButton(onClick = onRetry) { Text("重试") }
        }
        return
    }
    if (hasMore) {
        Box(Modifier.fillMaxWidth().padding(8.dp), contentAlignment = Alignment.Center) {
            OutlinedButton(onClick = onLoadMore) { Text("加载更多") }
        }
        return
    }
    if (!isEmpty) {
        Box(Modifier.fillMaxWidth().padding(16.dp), contentAlignment = Alignment.Center) {
            Text("没有更多了", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.outline)
        }
    }
}

/** 单据状态徽标。 */
@Composable
fun StatusChip(status: String) {
    val color = when (status) {
        "draft" -> MaterialTheme.colorScheme.outline
        "submitted" -> Color(0xFFE65100)
        "approved" -> Color(0xFF1565C0)
        "completed" -> Color(0xFF2E7D32)
        "rejected" -> MaterialTheme.colorScheme.error
        else -> MaterialTheme.colorScheme.outline
    }
    Surface(color = color.copy(alpha = 0.12f), contentColor = color, shape = MaterialTheme.shapes.small) {
        Text(
            statusLabel(status),
            modifier = Modifier.padding(horizontal = 8.dp, vertical = 2.dp),
            style = MaterialTheme.typography.labelSmall,
        )
    }
}

fun statusLabel(status: String): String = when (status) {
    "draft" -> "草稿"
    "submitted" -> "已提交"
    "approved" -> "已审批"
    "completed" -> "已完成"
    "rejected" -> "已驳回"
    else -> status
}

fun billTypeLabel(type: String): String = when (type) {
    "purchase" -> "采购入库"
    "sales" -> "销售出库"
    "purchase-return" -> "采购退货"
    "sale-return" -> "销售退货"
    "stock-adjustment" -> "库存调整"
    "transfer" -> "库存调拨"
    "stock-check" -> "库存盘点"
    else -> type
}

fun formatMoney(value: Double): String = "¥" + String.format(Locale.CHINA, "%,.2f", value)

fun formatQty(value: Double): String =
    if (value % 1.0 == 0.0) value.toLong().toString() else String.format(Locale.CHINA, "%.2f", value)

private val timeFormatter = DateTimeFormatter.ofPattern("yyyy-MM-dd HH:mm")

fun formatTime(raw: String?): String {
    if (raw.isNullOrBlank()) return "-"
    return runCatching {
        timeFormatter.withZone(ZoneId.systemDefault()).format(Instant.parse(raw))
    }.getOrElse { raw.take(16).replace('T', ' ') }
}

@Composable
fun ConfirmDialog(
    title: String,
    text: String,
    confirmText: String = "确定",
    onConfirm: () -> Unit,
    onDismiss: () -> Unit,
) {
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text(title) },
        text = { Text(text) },
        confirmButton = {
            TextButton(onClick = onConfirm) { Text(confirmText) }
        },
        dismissButton = {
            TextButton(onClick = onDismiss) { Text("取消") }
        },
    )
}

/** 通用带输入框的对话框，用于驳回原因。 */
@Composable
fun CommentDialog(
    title: String,
    label: String,
    confirmText: String,
    onConfirm: (String) -> Unit,
    onDismiss: () -> Unit,
) {
    var comment by remember { mutableStateOf("") }
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text(title) },
        text = {
            OutlinedTextField(
                value = comment,
                onValueChange = { comment = it },
                label = { Text(label) },
                singleLine = false,
                minLines = 2,
            )
        },
        confirmButton = {
            TextButton(
                onClick = { if (comment.isNotBlank()) onConfirm(comment.trim()) },
                enabled = comment.isNotBlank(),
            ) { Text(confirmText) }
        },
        dismissButton = {
            TextButton(onClick = onDismiss) { Text("取消") }
        },
    )
}
