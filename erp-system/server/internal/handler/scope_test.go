// 有问题欢迎联系 hot_kun@hotmail.com
package handler

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestScopeIgnoresClientHeaders 守住一条安全边界：作用域只能由服务端决定。
//
// 历史实现直接采信 X-Switch-Scope / X-Switch-Scope-Id，等于把「哪些缺陷生效」
// 的决定权交给了客户端——学生带一个 scope=all 就能绕开老师开的小组缺陷。
// 谁把请求头读回来，这个用例就会红。
func TestScopeIgnoresClientHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cases := []struct{ scope, id string }{
		{"", ""},
		{"all", "0"},
		{"group", "3"},
		{"personal", "7"},
		{"bogus", "not-a-number"},
	}

	for _, tc := range cases {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest("GET", "/api/v1/goods", nil)
		if tc.scope != "" {
			c.Request.Header.Set("X-Switch-Scope", tc.scope)
		}
		if tc.id != "" {
			c.Request.Header.Set("X-Switch-Scope-Id", tc.id)
		}

		scope, id := scopeFrom(c)
		if scope != "all" || id != 0 {
			t.Fatalf("scopeFrom 采信了请求头 %q/%q，返回 (%q, %d)，期望 (\"all\", 0)",
				tc.scope, tc.id, scope, id)
		}
	}
}
