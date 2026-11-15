package com.kong.erp.ui

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.systemBarsPadding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.Home
import androidx.compose.material.icons.automirrored.filled.List
import androidx.compose.material.icons.filled.Person
import androidx.compose.material.icons.filled.Place
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material.icons.filled.ShoppingCart
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.NavigationBar
import androidx.compose.material3.NavigationBarItem
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
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
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.unit.dp
import androidx.compose.material3.HorizontalDivider
import com.kong.erp.data.ApiException
import com.kong.erp.data.BillRef
import com.kong.erp.data.ErpApi
import com.kong.erp.data.Prefs
import com.kong.erp.data.Session
import kotlinx.coroutines.launch

/** 应用根：自动恢复会话，未登录进登录页。 */
@Composable
fun ErpApp() {
    val context = LocalContext.current
    val prefs = remember { Prefs(context.applicationContext) }
    val api = remember { ErpApi(prefs) }
    val scope = rememberCoroutineScope()
    var session by remember { mutableStateOf<Session?>(null) }
    var booting by remember { mutableStateOf(api.hasStoredSession()) }

    LaunchedEffect(Unit) {
        // 会话过期时后端返回 401；这里统一切回登录页
        api.onUnauthorized = { scope.launch { session = null } }
        if (api.hasStoredSession()) {
            session = try {
                api.me()
            } catch (_: ApiException) {
                api.clearSession()
                null
            }
        }
        booting = false
    }

    when {
        booting -> LoadingBox()
        session == null -> LoginScreen(api, prefs) { session = it }
        else -> MainScreen(
            api = api,
            session = session!!,
            onLogout = {
                scope.launch {
                    api.logout()
                    session = null
                }
            },
        )
    }
}

@Composable
private fun LoginScreen(api: ErpApi, prefs: Prefs, onLoggedIn: (Session) -> Unit) {
    var url by remember { mutableStateOf(prefs.baseUrl) }
    var username by remember { mutableStateOf(prefs.username) }
    var password by remember { mutableStateOf("") }
    var loading by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    val submit: () -> Unit = {
        if (!loading) {
            if (username.isBlank() || password.isBlank()) {
                error = "请输入用户名和密码"
            } else {
                scope.launch {
                    loading = true
                    error = null
                    try {
                        onLoggedIn(api.login(url, username, password))
                    } catch (e: ApiException) {
                        error = e.message
                    } finally {
                        loading = false
                    }
                }
            }
        }
    }

    Column(
        modifier = Modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .systemBarsPadding()
            .imePadding()
            .padding(24.dp),
        verticalArrangement = Arrangement.Center,
    ) {
        Text("ERP 移动端", style = MaterialTheme.typography.headlineMedium)
        Spacer(Modifier.height(4.dp))
        Text(
            "登录到实训 ERP 系统",
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
        Spacer(Modifier.height(32.dp))
        OutlinedTextField(
            value = url,
            onValueChange = {
                url = it
                error = null
            },
            label = { Text("服务器地址") },
            placeholder = { Text(Prefs.DEFAULT_BASE_URL) },
            singleLine = true,
            modifier = Modifier.fillMaxWidth(),
        )
        Spacer(Modifier.height(12.dp))
        OutlinedTextField(
            value = username,
            onValueChange = {
                username = it
                error = null
            },
            label = { Text("用户名") },
            singleLine = true,
            modifier = Modifier.fillMaxWidth(),
        )
        Spacer(Modifier.height(12.dp))
        OutlinedTextField(
            value = password,
            onValueChange = {
                password = it
                error = null
            },
            label = { Text("密码") },
            singleLine = true,
            visualTransformation = PasswordVisualTransformation(),
            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Password, imeAction = ImeAction.Done),
            keyboardActions = KeyboardActions(onDone = { submit() }),
            modifier = Modifier.fillMaxWidth(),
        )
        if (error != null) {
            Spacer(Modifier.height(12.dp))
            Text(error!!, color = MaterialTheme.colorScheme.error, style = MaterialTheme.typography.bodySmall)
        }
        Spacer(Modifier.height(24.dp))
        Button(onClick = submit, enabled = !loading, modifier = Modifier.fillMaxWidth()) {
            if (loading) {
                CircularProgressIndicator(
                    modifier = Modifier.size(18.dp),
                    color = MaterialTheme.colorScheme.onPrimary,
                    strokeWidth = 2.dp,
                )
            } else {
                Text("登录")
            }
        }
        Spacer(Modifier.height(12.dp))
        Text(
            "模拟器连不上宿主机时，先执行 adb reverse tcp:8889 tcp:8889，再填 http://127.0.0.1:8889；真机填电脑局域网 IP。",
            style = MaterialTheme.typography.bodySmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
    }
}

private data class Tab(
    val key: String,
    val label: String,
    val icon: @Composable () -> Unit,
)

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun MainScreen(api: ErpApi, session: Session, onLogout: () -> Unit) {
    var tab by remember { mutableStateOf("dashboard") }
    var detail by remember { mutableStateOf<BillRef?>(null) }
    var listRefresh by remember { mutableIntStateOf(0) }
    var auditDirection by remember { mutableIntStateOf(0) }
    var menuOpen by remember { mutableStateOf(false) }

    val currentDetail = detail
    if (currentDetail != null) {
        BillDetailScreen(
            api = api,
            session = session,
            ref = currentDetail,
            onBack = { detail = null },
            onChanged = { listRefresh++ },
        )
        return
    }

    // Tab 按权限过滤：没有 list 码的用户不该看到必然 403 的入口。
    val tabs = buildList {
        add(Tab("dashboard", "工作台") { Icon(Icons.Default.Home, contentDescription = null) })
        if (session.can("erp:master:list")) {
            add(Tab("goods", "商品") { Icon(Icons.AutoMirrored.Filled.List, contentDescription = null) })
        }
        if (session.can("erp:stock:list")) {
            add(Tab("stock", "库存") { Icon(Icons.Default.Place, contentDescription = null) })
        }
        if (session.can("erp:purchase:list") || session.can("erp:sales:list")) {
            add(Tab("bills", "单据") { Icon(Icons.Default.ShoppingCart, contentDescription = null) })
        }
        if (session.can("erp:audit:list")) {
            add(Tab("audit", "审核") { Icon(Icons.Default.CheckCircle, contentDescription = null) })
        }
    }
    if (tabs.none { it.key == tab }) tab = tabs.first().key

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text(tabs.first { it.key == tab }.label) },
                actions = {
                    IconButton(onClick = { listRefresh++ }) {
                        Icon(Icons.Default.Refresh, contentDescription = "刷新")
                    }
                    IconButton(onClick = { menuOpen = true }) {
                        Icon(Icons.Default.Person, contentDescription = "我的")
                    }
                    DropdownMenu(expanded = menuOpen, onDismissRequest = { menuOpen = false }) {
                        DropdownMenuItem(
                            text = { Text(session.displayName) },
                            onClick = { menuOpen = false },
                        )
                        DropdownMenuItem(
                            text = { Text(api.baseUrl, style = MaterialTheme.typography.bodySmall) },
                            onClick = { menuOpen = false },
                        )
                        HorizontalDivider()
                        DropdownMenuItem(
                            text = { Text("退出登录") },
                            onClick = {
                                menuOpen = false
                                onLogout()
                            },
                        )
                    }
                },
            )
        },
        bottomBar = {
            NavigationBar {
                tabs.forEach { item ->
                    NavigationBarItem(
                        selected = tab == item.key,
                        onClick = { tab = item.key },
                        icon = item.icon,
                        label = { Text(item.label) },
                    )
                }
            }
        },
    ) { padding ->
        val modifier = Modifier.padding(padding)
        when (tab) {
            "dashboard" -> DashboardScreen(
                api = api,
                refreshKey = listRefresh,
                onOpenAudit = { direction ->
                    auditDirection = direction
                    tab = "audit"
                },
                onOpenBill = { detail = it },
                modifier = modifier,
            )
            "goods" -> GoodsScreen(api, listRefresh, modifier)
            "stock" -> StockScreen(api, listRefresh, modifier)
            "bills" -> BillsScreen(
                api = api,
                refreshKey = listRefresh,
                onOpenBill = { detail = it },
                modifier = modifier,
            )
            else -> AuditScreen(
                api = api,
                session = session,
                direction = auditDirection,
                onDirectionChange = { auditDirection = it },
                refreshKey = listRefresh,
                modifier = modifier,
            )
        }
    }
}
