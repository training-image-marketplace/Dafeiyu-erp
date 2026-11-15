package com.kong.erp.data

import android.content.Context
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import org.json.JSONArray
import org.json.JSONObject
import java.io.IOException
import java.net.HttpURLConnection
import java.net.URL
import java.net.URLEncoder

/** 保存服务器地址与登录会话，下次启动可自动恢复。 */
class Prefs(context: Context) {
    private val sp = context.getSharedPreferences("erp", Context.MODE_PRIVATE)

    var baseUrl: String
        get() = sp.getString("base_url", DEFAULT_BASE_URL) ?: DEFAULT_BASE_URL
        set(value) {
            sp.edit().putString("base_url", value).apply()
        }

    var username: String
        get() = sp.getString("username", "") ?: ""
        set(value) {
            sp.edit().putString("username", value).apply()
        }

    var cookie: String?
        get() = sp.getString("cookie", null)
        set(value) {
            sp.edit().putString("cookie", value).apply()
        }

    var csrf: String?
        get() = sp.getString("csrf", null)
        set(value) {
            sp.edit().putString("csrf", value).apply()
        }

    fun clearSession() {
        sp.edit().remove("cookie").remove("csrf").apply()
    }

    companion object {
        /** 模拟器访问宿主机用的地址；真机请在登录页改成局域网 IP。 */
        const val DEFAULT_BASE_URL = "http://10.0.2.2:8889"
    }
}

data class Session(
    val displayName: String,
    val username: String,
    val isAdmin: Boolean,
    val perms: Set<String>,
) {
    fun can(perm: String): Boolean = isAdmin || perm in perms
}

data class Page<T>(val list: List<T>, val total: Int)

data class Dashboard(
    val goodsCount: Int,
    val customerCount: Int,
    val supplierCount: Int,
    val warehouseCount: Int,
    val stockGoodsCount: Int,
    val stockTotal: Double,
    val pendingIn: Int,
    val pendingOut: Int,
    val stockAlerts: Int,
    val pending: List<AuditBill>,
)

data class AuditBill(
    val billType: String,
    val billId: Long,
    val billNo: String,
    val partnerName: String,
    val warehouseName: String,
    val totalAmount: Double,
    val direction: String,
    val qty: Double,
    val status: String,
    val creatorName: String,
    val submittedAt: String?,
)

data class Goods(
    val id: Long,
    val code: String,
    val name: String,
    val spec: String,
    val unit: String,
    val salePrice: Double,
    val stockQty: Double,
    val status: String,
)

data class Stock(
    val warehouseId: Long,
    val warehouseName: String,
    val goodsId: Long,
    val goodsCode: String,
    val goodsName: String,
    val unit: String,
    val qty: Double,
)

data class OrderLine(
    val goodsName: String,
    val qty: Double,
    val price: Double,
    val amount: Double,
)

/** 采购单与销售单形状一致，用 type 区分（purchase / sales）。 */
data class Order(
    val id: Long,
    val type: String,
    val no: String,
    val partner: String,
    val warehouse: String,
    val status: String,
    val totalAmount: Double,
    val createdAt: String,
    val submittedAt: String?,
    val approvedAt: String?,
    val items: List<OrderLine>,
)

/** 指向一张单据，用于打开详情。 */
data class BillRef(val type: String, val id: Long)

class ApiException(val status: Int, val key: String) : Exception(localize(key, status))

private fun localize(key: String, status: Int): String = when (key) {
    "error.badCredentials" -> "用户名或密码错误"
    "error.unauthorized" -> "登录已过期，请重新登录"
    "error.forbidden" -> "没有操作权限"
    "error.network" -> "网络连接失败，请检查服务器地址"
    "error.accountLocked" -> "账号已锁定，请稍后再试"
    "error.tooManyRequests" -> "请求过于频繁，请稍后再试"
    "error.invalidState" -> "单据状态已变化，请刷新后重试"
    "error.insufficientStock" -> "库存不足"
    "error.returnQtyExceeded" -> "退货数量超过可退数量"
    "error.notFound" -> "数据不存在"
    "error.duplicate" -> "数据重复"
    "error.validationFailed" -> "参数校验失败"
    "error.hasDependents" -> "存在关联数据，无法删除"
    "error.csrfInvalid" -> "会话校验失败，请重新登录"
    "error.insecureUrl" -> "公网地址必须使用 https"
    "error.internal" -> "服务器内部错误"
    else -> if (status == 0) "网络连接失败，请检查服务器地址" else key
}

/**
 * ERP 后端客户端。
 *
 * 认证沿用服务端的 Cookie + CSRF：登录后保存会话 cookie，写请求带 X-CSRF-Token。
 * 只做移动端需要的子集：工作台、主数据、库存、采购/销售单、审核中心。
 */
class ErpApi(private val prefs: Prefs) {

    /** 会话失效时的回调，由界面注册为「回到登录页」。 */
    var onUnauthorized: (() -> Unit)? = null

    var baseUrl: String = prefs.baseUrl
        private set
    private var cookie: String? = prefs.cookie
    private var csrf: String? = prefs.csrf
    private var lastSetCookie: String? = null

    fun hasStoredSession(): Boolean = cookie != null

    suspend fun login(url: String, username: String, password: String): Session =
        withContext(Dispatchers.IO) {
            val normalized = normalizeBaseUrl(url)
            if (!isAllowedBaseUrl(normalized)) {
                throw ApiException(0, "error.insecureUrl")
            }
            baseUrl = normalized
            val body = JSONObject().put("username", username).put("password", password)
            val data = request("POST", "/auth/login", body, useCsrf = false)
            val cookieValue = lastSetCookie ?: throw ApiException(0, "error.network")
            cookie = cookieValue
            csrf = data.optString("csrf_token")
            prefs.baseUrl = normalized
            prefs.username = username
            prefs.cookie = cookie
            prefs.csrf = csrf
            parseSession(data)
        }

    suspend fun me(): Session = withContext(Dispatchers.IO) {
        val data = request("GET", "/auth/me")
        csrf = data.optString("csrf_token").ifBlank { csrf }
        prefs.csrf = csrf
        parseSession(data)
    }

    suspend fun logout() {
        withContext(Dispatchers.IO) {
            try {
                if (cookie != null) request("POST", "/auth/logout")
            } catch (_: ApiException) {
                // 本地会话无论如何都要清掉
            }
        }
        clearSession()
    }

    fun clearSession() {
        cookie = null
        csrf = null
        lastSetCookie = null
        prefs.clearSession()
    }

    suspend fun dashboard(): Dashboard = withContext(Dispatchers.IO) {
        val d = request("GET", "/dashboard")
        Dashboard(
            goodsCount = d.optInt("goods_count"),
            customerCount = d.optInt("customer_count"),
            supplierCount = d.optInt("supplier_count"),
            warehouseCount = d.optInt("warehouse_count"),
            stockGoodsCount = d.optInt("stock_goods_count"),
            stockTotal = d.optDouble("stock_total", 0.0),
            pendingIn = d.optInt("pending_in"),
            pendingOut = d.optInt("pending_out"),
            stockAlerts = d.optInt("stock_alerts"),
            pending = d.array("pending").map(::parseAuditBill),
        )
    }

    suspend fun goods(keyword: String, page: Int): Page<Goods> = withContext(Dispatchers.IO) {
        val d = request("GET", "/goods?page=$page&page_size=20" + query("keyword", keyword))
        Page(d.array("list").map { o ->
            Goods(
                id = o.optLong("id"),
                code = o.str("code"),
                name = o.str("name"),
                spec = o.str("spec"),
                unit = o.str("unit"),
                salePrice = o.optDouble("sale_price", 0.0),
                stockQty = o.optDouble("stock_qty", 0.0),
                status = o.str("status"),
            )
        }, d.optInt("total"))
    }

    suspend fun stocks(keyword: String, page: Int): Page<Stock> = withContext(Dispatchers.IO) {
        val d = request("GET", "/stocks?page=$page&page_size=20" + query("keyword", keyword))
        Page(d.array("list").map { o ->
            Stock(
                warehouseId = o.optLong("warehouse_id"),
                warehouseName = o.str("warehouse_name"),
                goodsId = o.optLong("goods_id"),
                goodsCode = o.str("goods_code"),
                goodsName = o.str("goods_name"),
                unit = o.str("unit"),
                qty = o.optDouble("qty", 0.0),
            )
        }, d.optInt("total"))
    }

    /** type 取 purchase / sales。 */
    suspend fun orders(type: String, keyword: String, status: String, page: Int): Page<Order> =
        withContext(Dispatchers.IO) {
            val path = if (type == "purchase") "/purchases" else "/sales"
            val d = request(
                "GET",
                path + "?page=" + page + "&page_size=20" +
                    query("keyword", keyword) + query("status", status),
            )
            Page(d.array("list").map { parseOrder(type, it, withItems = false) }, d.optInt("total"))
        }

    suspend fun order(type: String, id: Long): Order = withContext(Dispatchers.IO) {
        val path = if (type == "purchase") "/purchases/" + id else "/sales/" + id
        parseOrder(type, request("GET", path), withItems = true)
    }

    suspend fun submitOrder(type: String, id: Long) = withContext(Dispatchers.IO) {
        request("POST", orderPath(type, id, "submit"))
        Unit
    }

    suspend fun withdrawOrder(type: String, id: Long) = withContext(Dispatchers.IO) {
        request("POST", orderPath(type, id, "withdraw"))
        Unit
    }

    /** 采购单入库，销售单出库。 */
    suspend fun completeOrder(type: String, id: Long) = withContext(Dispatchers.IO) {
        request("POST", orderPath(type, id, if (type == "purchase") "receive" else "ship"))
        Unit
    }

    suspend fun deleteOrder(type: String, id: Long) = withContext(Dispatchers.IO) {
        request("DELETE", orderPath(type, id, ""))
        Unit
    }

    suspend fun audits(direction: String, page: Int): Page<AuditBill> = withContext(Dispatchers.IO) {
        val d = request("GET", "/audits?direction=" + direction + "&status=submitted&page=" + page + "&page_size=20")
        Page(d.array("list").map(::parseAuditBill), d.optInt("total"))
    }

    suspend fun audit(billType: String, id: Long, pass: Boolean, comment: String) =
        withContext(Dispatchers.IO) {
            val body = JSONObject().put("pass", pass).put("comment", comment)
            request("POST", "/audits/" + billType + "/" + id, body)
            Unit
        }

    private fun orderPath(type: String, id: Long, action: String): String {
        val base = if (type == "purchase") "/purchases/" + id else "/sales/" + id
        return if (action.isEmpty()) base else base + "/" + action
    }

    private fun parseSession(data: JSONObject): Session {
        val user = data.optJSONObject("user") ?: JSONObject()
        val name = user.str("nickname").ifBlank { user.str("username") }
        val perms = buildSet {
            val arr = data.optJSONArray("perms") ?: return@buildSet
            for (i in 0 until arr.length()) {
                // 服务端契约是字符串数组；用 optString 容忍异常代理返回的非字符串，
                // 避免 200 响应把登录协程打崩。
                val code = arr.optString(i, "")
                if (code.isNotBlank()) add(code)
            }
        }
        return Session(name, user.str("username"), data.optBoolean("is_admin"), perms)
    }

    private fun parseAuditBill(o: JSONObject): AuditBill = AuditBill(
        billType = o.str("bill_type"),
        billId = o.optLong("bill_id"),
        billNo = o.str("bill_no"),
        partnerName = o.str("partner_name"),
        warehouseName = o.str("warehouse_name"),
        totalAmount = o.optDouble("total_amount", 0.0),
        direction = o.str("direction"),
        qty = o.optDouble("qty", 0.0),
        status = o.str("status"),
        creatorName = o.str("creator_name"),
        submittedAt = o.nullableString("submitted_at"),
    )

    private fun parseOrder(type: String, o: JSONObject, withItems: Boolean): Order = Order(
        id = o.optLong("id"),
        type = type,
        no = o.str("order_no"),
        partner = if (type == "purchase") o.str("supplier_name") else o.str("customer_name"),
        warehouse = o.str("warehouse_name"),
        status = o.str("status"),
        totalAmount = o.optDouble("total_amount", 0.0),
        createdAt = o.str("created_at"),
        submittedAt = o.nullableString("submitted_at"),
        approvedAt = o.nullableString("approved_at"),
        items = if (!withItems) emptyList() else o.array("items").map { item ->
            OrderLine(
                goodsName = item.str("goods_name"),
                qty = item.optDouble("qty", 0.0),
                price = item.optDouble("price", 0.0),
                amount = item.optDouble("amount", 0.0),
            )
        },
    )

    private fun request(
        method: String,
        path: String,
        body: JSONObject? = null,
        useCsrf: Boolean = true,
    ): JSONObject {
        if (!isAllowedBaseUrl(baseUrl)) {
            throw ApiException(0, "error.insecureUrl")
        }
        val url = try {
            URL(baseUrl.trimEnd('/') + "/api/v1" + path)
        } catch (e: Exception) {
            throw ApiException(0, "error.network")
        }
        var conn: HttpURLConnection? = null
        try {
            conn = (url.openConnection() as HttpURLConnection).apply {
                requestMethod = method
                connectTimeout = 15_000
                readTimeout = 20_000
                setRequestProperty("Accept", "application/json")
                cookie?.let { setRequestProperty("Cookie", it) }
                if (body != null) {
                    doOutput = true
                    setRequestProperty("Content-Type", "application/json; charset=utf-8")
                }
                if (method != "GET" && useCsrf) {
                    csrf?.let { setRequestProperty("X-CSRF-Token", it) }
                }
            }
            if (body != null) {
                conn.outputStream.use { it.write(body.toString().toByteArray()) }
            }
            val status = conn.responseCode
            val text = (if (status in 200..299) conn.inputStream else conn.errorStream)
                ?.bufferedReader()?.use { it.readText() }.orEmpty()
            if (path == "/auth/login") {
                lastSetCookie = conn.getHeaderField("Set-Cookie")?.substringBefore(';')
            }
            val envelope = if (text.isBlank()) JSONObject() else runCatching { JSONObject(text) }
                .getOrElse { JSONObject() }
            if (status !in 200..299) {
                if (status == 401 && path != "/auth/login") {
                    clearSession()
                    onUnauthorized?.invoke()
                }
                throw ApiException(status, envelope.optString("msg", "error.internal"))
            }
            // 200 但 body 不是本服务的信封，说明被网关/代理拦截了
            if (envelope.optInt("code", 0) != 200) {
                throw ApiException(status, envelope.optString("msg", "error.internal"))
            }
            return envelope.optJSONObject("data") ?: JSONObject()
        } catch (e: ApiException) {
            throw e
        } catch (e: IOException) {
            throw ApiException(0, "error.network")
        } finally {
            conn?.disconnect()
        }
    }

    companion object {
        fun normalizeBaseUrl(raw: String): String {
            val trimmed = raw.trim().trimEnd('/')
            if (trimmed.isEmpty()) return Prefs.DEFAULT_BASE_URL
            return if (trimmed.startsWith("http://") || trimmed.startsWith("https://")) {
                trimmed
            } else {
                "http://" + trimmed
            }
        }

        /**
         * 明文 HTTP 只允许本地/内网地址；公网必须 https，避免登录口令与
         * 会话 cookie 在可嗅探网络上暴露。真机连局域网 IP 仍然可用。
         */
        fun isAllowedBaseUrl(url: String): Boolean {
            val uri = runCatching { java.net.URI(url) }.getOrNull() ?: return false
            if (uri.scheme.equals("https", ignoreCase = true)) return true
            val host = uri.host?.lowercase() ?: return false
            if (host == "localhost" || host == "::1" || host == "10.0.2.2") return true
            val parts = host.split(".").mapNotNull { it.toIntOrNull() }
            if (parts.size == 4) {
                return parts[0] == 10 ||
                    (parts[0] == 172 && parts[1] in 16..31) ||
                    (parts[0] == 192 && parts[1] == 168) ||
                    parts[0] == 127
            }
            return false
        }

        private fun query(key: String, value: String): String =
            if (value.isBlank()) "" else "&" + key + "=" + URLEncoder.encode(value, "UTF-8")
    }
}

private fun JSONObject.str(key: String): String =
    if (isNull(key)) "" else optString(key, "")

private fun JSONObject.nullableString(key: String): String? =
    if (isNull(key)) null else optString(key, "").ifBlank { null }

private fun JSONObject.array(key: String): List<JSONObject> {
    val arr: JSONArray = optJSONArray(key) ?: return emptyList()
    return (0 until arr.length()).mapNotNull { arr.optJSONObject(it) }
}
