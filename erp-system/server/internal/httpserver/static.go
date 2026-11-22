// 有问题欢迎联系 hot_kun@hotmail.com
package httpserver

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"

	"practical-training/erp-system/internal/httpx"
)

const AssetsPrefix = "/assets/"

// MountStatic 从磁盘目录挂载前端构建产物。
//
// 前端产物不 embed 进二进制，因此重新构建前端后无需重启后端，浏览器刷新即可生效。
// 代价是 index.html 必须不可缓存，否则浏览器会继续引用已不存在的旧 asset 哈希。
func MountStatic(r *gin.Engine, webDir string) error {
	indexPath := filepath.Join(webDir, "index.html")
	assetsDir := filepath.Join(webDir, "assets")

	// 整个目录不存在 = 这是一个纯 API 部署，只警告不阻断启动
	serveSPA := false
	if _, err := os.Stat(webDir); os.IsNotExist(err) {
		// 这句提示必须明确到「怎么修」：降到纯 API 模式的表现为
		// 「打开 :8889 看到一句 JSON 404」，很难联想到是前端目录的问题
		log.Printf("前端目录 %s 不存在，仅提供 API。"+
			"若要托管前端：执行 make build 生成 bin/web，或设置 APP_WEB_DIR 指向 dist 目录", webDir)
	} else { // 目录存在但结构不完整 = 部署配置错了，启动即失败好过运行期 404
		if _, err := os.Stat(indexPath); err != nil {
			return fmt.Errorf("前端目录 %q 缺少 index.html: %w", webDir, err)
		}
		if _, err := os.Stat(assetsDir); err != nil {
			return fmt.Errorf("前端目录 %q 缺少 assets/: %w", webDir, err)
		}
		serveSPA = true
		log.Printf("从 %s 托管前端", webDir)
		r.StaticFS(AssetsPrefix, http.Dir(assetsDir))

		// Android 安装包放这里由后端直接托管，不经过鉴权，也不用重新构建前端：
		// 把 apk 丢进 <webDir>/apk/ 就能被 /apk/<文件名> 下载到。
		// 目录不存在时不注册路由，避免 gin 把 /apk/* 的请求交给一个空文件服务器。
		apkDir := filepath.Join(webDir, "apk")
		if _, err := os.Stat(apkDir); err == nil {
			r.Static("/apk", apkDir)
			log.Printf("从 %s 托管 APK 下载", apkDir)
		}

		// 品牌插画（吉祥物）：不在 assets/ 下，不单独挂载会落到 SPA fallback，
		// 浏览器拿到 index.html 当图片解析，表现为「图片不显示且不报错」。
		brandDir := filepath.Join(webDir, "brand")
		if _, err := os.Stat(brandDir); err == nil {
			r.Static("/brand", brandDir)
			log.Printf("从 %s 托管品牌素材", brandDir)
		}
	}

	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path

		// API 路径和缺失的静态资源绝不能回落到 index.html：
		// 否则前端会把一整页 HTML 当 JSON 解析，报错信息完全看不出真正原因。
		//
		// /apk/ 也要在这里挡住：gin 的静态路由找不到文件时会交给 NoRoute，
		// 回落到 index.html 就成了「200 + HTML」——下载页据此判断安装包已上传，
		// 用户点下载拿到的是一个网页。缺失的安装包必须是 404。
		if strings.HasPrefix(path, "/api/") || strings.HasPrefix(path, AssetsPrefix) ||
			strings.HasPrefix(path, "/apk/") || strings.HasPrefix(path, "/brand/") {
			httpx.Fail(c, http.StatusNotFound, httpx.KeyNotFound)
			return
		}

		if !serveSPA {
			// 纯 API 模式下打开首页是最常见的困惑点：看到一句 JSON 404，
			// 会以为服务坏了。这里明说是「前端没部署」，并指出两种正确用法。
			httpx.Fail(c, http.StatusNotFound, httpx.KeyFrontendDisabled)
			return
		}
		// SPA fallback：刷新 /system/user 这类前端路由不返回 404。
		// no-cache 让浏览器每次回源校验，从而拿到最新的 asset 哈希。
		c.Header("Cache-Control", "no-cache")
		c.File(indexPath)
	})
	return nil
}
