// 有问题欢迎联系 hot_kun@hotmail.com
package httpserver

import (
	"net/http"
	"testing"
)

// TestBugSwitchCatalogueEndpoint 缺陷目录接口只对开关权限开放。
func TestBugSwitchCatalogueEndpoint(t *testing.T) {
	r, _ := newTestRouter(t)
	admin := login(t, r, "admin", "admin123")

	w := call(t, r, http.MethodGet, "/api/v1/switches/bugs", "", admin)
	if w.Code != http.StatusOK {
		t.Fatalf("读取缺陷目录: %d %s", w.Code, w.Body.String())
	}
	var res struct {
		List []struct {
			Code  string `json:"code"`
			Name  string `json:"name"`
			Area  string `json:"area"`
			Wired bool   `json:"wired"`
		} `json:"list"`
	}
	decodeData(t, w.Body.Bytes(), &res)
	if len(res.List) != 6 {
		t.Fatalf("缺陷目录应有 6 条（4 条生效 + 2 条未接线），实际 %d", len(res.List))
	}
	wired := 0
	for _, b := range res.List {
		if b.Code == "" || b.Name == "" || b.Area == "" {
			t.Errorf("目录条目字段不完整: %+v", b)
		}
		if b.Wired {
			wired++
		}
	}
	if wired != 4 {
		t.Errorf("已接线缺陷应为 4 条，实际 %d", wired)
	}

	// 拼错的缺陷码不能写进配置
	if w := call(t, r, http.MethodPost, "/api/v1/switches/bug", `{"bug_code":"BUG-999","enabled":true}`, admin);
		w.Code != http.StatusBadRequest {
		t.Fatalf("未知缺陷码应返回 400，实际 %d %s", w.Code, w.Body.String())
	}

	// 没有任何角色的账号不能读目录
	createUser(t, r, admin, "no-perm", "no-perm-pw")
	viewer := login(t, r, "no-perm", "no-perm-pw")
	if w := call(t, r, http.MethodGet, "/api/v1/switches/bugs", "", viewer); w.Code != http.StatusForbidden {
		t.Errorf("无权限账号读取缺陷目录返回 %d，期望 403", w.Code)
	}
}
