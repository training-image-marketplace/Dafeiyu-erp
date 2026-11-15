// 有问题欢迎联系 hot_kun@hotmail.com
package handler

import (
	"testing"

	"practical-training/erp-system/internal/domain"
)

// TestMenuRegistryMatchesGateway 保证「平台能勾的」与「网关会拦的」是同一套码。
//
// 两处分开维护很容易漂移：注册表里多一条，平台会显示一个关不掉的开关；
// 网关映射里多一个码，那个菜单就永远不会出现在平台面板上。
func TestMenuRegistryMatchesGateway(t *testing.T) {
	registered := make(map[string]bool, len(domain.MenuSwitches))
	for _, m := range domain.MenuSwitches {
		if registered[m.Code] {
			t.Fatalf("菜单码 %q 在注册表里重复", m.Code)
		}
		if m.Name == "" || m.NameEN == "" {
			t.Errorf("菜单 %q 缺少名称（中英都要有，平台按语言渲染）", m.Code)
		}
		registered[m.Code] = true
	}

	used := make(map[string]bool, len(menuCodeByResource))
	for _, code := range menuCodeByResource {
		used[code] = true
	}

	for code := range used {
		if !registered[code] {
			t.Errorf("网关用到的菜单码 %q 没有登记在 domain.MenuSwitches：平台面板里看不到它", code)
		}
	}
	for code := range registered {
		if !used[code] {
			t.Errorf("登记的菜单码 %q 没有任何资源段映射：勾掉它也不会有接口被拦", code)
		}
	}
}
