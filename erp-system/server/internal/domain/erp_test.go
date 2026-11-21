// 有问题欢迎联系 hot_kun@hotmail.com
package domain

import "testing"

// TestBugCatalogCoversEveryDeclaredCode 目录必须覆盖所有缺陷常量。
//
// 漏一条的后果是「页面上根本没有这个开关」，而缺陷仍然能在业务里被打开；
// 反过来，目录里塞了不存在的码只会在业务里永远读不到。
func TestBugCatalogCoversEveryDeclaredCode(t *testing.T) {
	declared := []string{BugAmountError, BugNegativeStock, BugApproveSelf, BugEditCompleted, BugDuplicatePage, BugOrphanCustomer}
	seen := map[string]bool{}
	for _, b := range BugSwitches {
		if b.Code == "" || b.Name == "" || b.Area == "" {
			t.Errorf("缺陷条目字段不完整: %+v", b)
		}
		if seen[b.Code] {
			t.Errorf("缺陷码 %s 在目录里重复", b.Code)
		}
		seen[b.Code] = true
	}
	for _, code := range declared {
		if !BugDeclared(code) {
			t.Errorf("缺陷码 %s 没有出现在 BugSwitches 目录里", code)
		}
	}
	if len(seen) != len(declared) {
		t.Errorf("目录有 %d 条，声明的缺陷码有 %d 个", len(seen), len(declared))
	}
}

// TestWiredBugSwitchesAreSubset 已接线集合只包含目录里的已接线项。
func TestWiredBugSwitchesAreSubset(t *testing.T) {
	wired := WiredBugSwitches()
	if len(wired) == 0 {
		t.Fatal("已接线缺陷为空：平台开关面板会一个选项都没有")
	}
	for _, b := range wired {
		if !b.Wired {
			t.Errorf("%s 出现在已接线集合里但 Wired=false", b.Code)
		}
	}
	if BugDeclared("BUG-NOT-EXIST") {
		t.Error("未声明的缺陷码不应被判为已声明")
	}
}
