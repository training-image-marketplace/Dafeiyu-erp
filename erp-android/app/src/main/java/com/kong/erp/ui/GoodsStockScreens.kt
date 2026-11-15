package com.kong.erp.ui

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.Card
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.kong.erp.data.ErpApi
import com.kong.erp.data.Goods
import com.kong.erp.data.Stock
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

@Composable
fun GoodsScreen(api: ErpApi, refreshKey: Int, modifier: Modifier = Modifier) {
    var input by remember { mutableStateOf("") }
    var keyword by remember { mutableStateOf("") }
    val scope = rememberCoroutineScope()

    LaunchedEffect(input) {
        delay(400)
        if (input.trim() != keyword) keyword = input.trim()
    }

    val list = remember(keyword, refreshKey) {
        PagedList<Goods> { page -> api.goods(keyword, page) }
    }
    LaunchedEffect(list) { list.reload() }

    Column(modifier.fillMaxSize().padding(horizontal = 16.dp)) {
        Spacer(Modifier.height(8.dp))
        SearchField(input, { input = it }, "商品名称 / 编码")
        Spacer(Modifier.height(8.dp))
        when {
            list.items.isEmpty() && list.loading -> LoadingBox(Modifier.weight(1f))
            list.items.isEmpty() && list.error != null ->
                ErrorBox(list.error!!) { scope.launch { list.reload() } }
            else -> LazyColumn(
                modifier = Modifier.weight(1f),
                contentPadding = PaddingValues(bottom = 16.dp),
                verticalArrangement = Arrangement.spacedBy(8.dp),
            ) {
                if (list.items.isEmpty()) item { EmptyHint("暂无商品") }
                items(list.items, key = { it.id }) { goods -> GoodsCard(goods) }
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
private fun GoodsCard(goods: Goods) {
    Card(Modifier.fillMaxWidth()) {
        Column(Modifier.padding(12.dp)) {
            Row(verticalAlignment = Alignment.Top) {
                Column(Modifier.weight(1f)) {
                    Text(goods.name, style = MaterialTheme.typography.titleSmall, fontWeight = FontWeight.Medium)
                    Text(
                        listOf(goods.code, goods.spec).filter { it.isNotBlank() }.joinToString(" · "),
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
                Column(horizontalAlignment = Alignment.End) {
                    Text(formatMoney(goods.salePrice), style = MaterialTheme.typography.bodyMedium)
                    Text(
                        "库存 " + formatQty(goods.stockQty) + " " + goods.unit,
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
            }
        }
    }
}

@Composable
fun StockScreen(api: ErpApi, refreshKey: Int, modifier: Modifier = Modifier) {
    var input by remember { mutableStateOf("") }
    var keyword by remember { mutableStateOf("") }
    val scope = rememberCoroutineScope()

    LaunchedEffect(input) {
        delay(400)
        if (input.trim() != keyword) keyword = input.trim()
    }

    val list = remember(keyword, refreshKey) {
        PagedList<Stock> { page -> api.stocks(keyword, page) }
    }
    LaunchedEffect(list) { list.reload() }

    Column(modifier.fillMaxSize().padding(horizontal = 16.dp)) {
        Spacer(Modifier.height(8.dp))
        SearchField(input, { input = it }, "商品 / 仓库")
        Spacer(Modifier.height(8.dp))
        when {
            list.items.isEmpty() && list.loading -> LoadingBox(Modifier.weight(1f))
            list.items.isEmpty() && list.error != null ->
                ErrorBox(list.error!!) { scope.launch { list.reload() } }
            else -> LazyColumn(
                modifier = Modifier.weight(1f),
                contentPadding = PaddingValues(bottom = 16.dp),
                verticalArrangement = Arrangement.spacedBy(8.dp),
            ) {
                if (list.items.isEmpty()) item { EmptyHint("暂无库存记录") }
                items(list.items, key = { it.warehouseId.toString() + "-" + it.goodsId }) { stock ->
                    StockCard(stock)
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
private fun StockCard(stock: Stock) {
    Card(Modifier.fillMaxWidth()) {
        Row(Modifier.padding(12.dp), verticalAlignment = Alignment.CenterVertically) {
            Column(Modifier.weight(1f)) {
                Text(stock.goodsName, style = MaterialTheme.typography.titleSmall, fontWeight = FontWeight.Medium)
                Text(
                    stock.goodsCode + " · " + stock.warehouseName,
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
            Text(
                formatQty(stock.qty) + " " + stock.unit,
                style = MaterialTheme.typography.titleMedium,
            )
        }
    }
}
