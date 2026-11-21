// 有问题欢迎联系 hot_kun@hotmail.com
package repository

import (
	"context"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"practical-training/erp-system/internal/domain"
	"practical-training/erp-system/internal/testutil"
)

// migrationCount 是 migrations/ 下的文件数。
//
// 这里写死而不是动态统计：新增迁移时必须显式改这个数字，
// 从而强制作者想一想「新迁移是否也该有对应的验证」。
const migrationCount = 29

func TestMigrateCreatesSchemaAndIsIdempotent(t *testing.T) {
	db := testutil.New(t)
	ctx := context.Background()

	for i := 1; i <= 2; i++ {
		if err := Migrate(ctx, db); err != nil {
			t.Fatalf("第 %d 次 Migrate: %v", i, err)
		}
	}

	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations`).Scan(&count); err != nil {
		t.Fatalf("读 schema_migrations: %v", err)
	}
	if count != migrationCount {
		t.Errorf("schema_migrations 有 %d 条记录，期望 %d（迁移不应重复执行）", count, migrationCount)
	}

	// 迁移建出来的表要真的可用
	if _, err := db.ExecContext(ctx,
		`INSERT INTO sys_menus (title_key) VALUES (?)`, "menu.test"); err != nil {
		t.Fatalf("写入 sys_menus: %v", err)
	}

	// 0012 建的任务执行日志：写一条真记录，并验证 CHECK 真的在拦。
	// trigger 是 MySQL 保留字，列名改成了 trigger_type；这里必须实测，
	// 写不进去的话，任务执行成功而历史记录静默丢失。
	if _, err := db.ExecContext(ctx, `
		INSERT INTO sys_job_logs (job_key, trigger_type, status, error, duration_ms)
		VALUES (?, ?, ?, ?, ?)`,
		"cleanup:old_logs", domain.JobTriggerCron, domain.JobStatusSuccess, "", 12); err != nil {
		t.Fatalf("写入 sys_job_logs: %v", err)
	}
	if _, err := db.ExecContext(ctx,
		`INSERT INTO sys_job_logs (job_key, trigger_type, status) VALUES (?, ?, ?)`,
		"cleanup:old_logs", "scheduler", domain.JobStatusSuccess); err == nil {
		t.Error("sys_job_logs.trigger_type 的 CHECK 没拦住非法触发方式")
	}

	// 0013 的登录失败锁定列：必须存在、初始为「未锁定」。
	var failedCount int
	var lockedUntil *time.Time
	if err := db.QueryRowContext(ctx,
		`SELECT failed_count, locked_until FROM sys_users WHERE username = 'admin'`).
		Scan(&failedCount, &lockedUntil); err != nil {
		t.Fatalf("读登录失败锁定列: %v", err)
	}
	if failedCount != 0 || lockedUntil != nil {
		t.Errorf("admin 初始状态 = failed_count %d / locked_until %v，期望 0 / NULL", failedCount, lockedUntil)
	}
}

// TestSeedRbac 验证 0003 种子数据的完整性与可用性。
func TestSeedRbac(t *testing.T) {
	db := testutil.New(t)
	ctx := context.Background()

	if err := Migrate(ctx, db); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	// 内置角色存在，且 code 唯一
	var roleID int64
	if err := db.QueryRowContext(ctx, `SELECT id FROM sys_roles WHERE code = 'admin'`).Scan(&roleID); err != nil {
		t.Fatalf("内置角色 admin 不存在: %v", err)
	}

	// 管理员账号存在，并且**种子哈希必须能验证通过**——
	// 哈希是手抄进 SQL 的，抄错一位就会导致谁也登不进去，必须由用例兜住
	var userID int64
	var hash string
	if err := db.QueryRowContext(ctx,
		`SELECT id, password FROM sys_users WHERE username = 'admin'`).Scan(&userID, &hash); err != nil {
		t.Fatalf("管理员账号不存在: %v", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte("admin123")); err != nil {
		t.Errorf("种子密码哈希无法通过校验（admin123）: %v", err)
	}

	// 管理员必须已经绑定内置角色，否则登录后没有任何权限
	var linkCount int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM sys_user_roles WHERE user_id = ? AND role_id = ?`, userID, roleID).Scan(&linkCount); err != nil {
		t.Fatal(err)
	}
	if linkCount != 1 {
		t.Errorf("admin 未绑定 admin 角色（%d 条关联）", linkCount)
	}

	// 菜单树：四个根目录（系统管理 / 系统监控 / 系统工具 / ERP 业务），各自带子菜单
	var rootCount int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM sys_menus WHERE menu_type = 'directory' AND parent_id IS NULL`).Scan(&rootCount); err != nil {
		t.Fatalf("读根目录: %v", err)
	}
	if rootCount != 4 {
		t.Errorf("有 %d 个根目录，期望 4（系统管理 + 系统监控 + 系统工具 + ERP 业务）", rootCount)
	}

	for _, tc := range []struct {
		dirKey string
		kids   int
	}{
		{"menu.system", 4},
		{"menu.monitor", 4},
		{"menu.tool", 2},
		// ERP 下：三组业务目录 + 缺陷开关 + Android 客户端
		{"menu.erp", 5},
		{"menu.erp.basic", 5},
		{"menu.erp.biz", 8}, // 库存/采购/销售/两类退货 + 调拨/盘点/报表
		{"menu.erp.audit", 2},
	} {
		var childCount int
		err := db.QueryRowContext(ctx, `
			SELECT COUNT(*) FROM sys_menus c
			JOIN sys_menus p ON p.id = c.parent_id
			WHERE p.title_key = ?`, tc.dirKey).Scan(&childCount)
		if err != nil {
			t.Fatal(err)
		}
		if childCount != tc.kids {
			t.Errorf("%s 下有 %d 个子菜单，期望 %d", tc.dirKey, childCount, tc.kids)
		}
	}

	// title_key 不得为空，且菜单必须按 sort 可排序
	var badCount int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM sys_menus WHERE title_key = '' OR menu_type = 'menu' AND component = ''`).Scan(&badCount); err != nil {
		t.Fatal(err)
	}
	if badCount != 0 {
		t.Errorf("有 %d 条菜单缺少 title_key 或 component", badCount)
	}
}

// TestMigrateOnLegacyDatabase 覆盖升级路径：库里已有表与数据，但没有任何迁移记录。
func TestMigrateOnLegacyDatabase(t *testing.T) {
	db := testutil.New(t)
	ctx := context.Background()

	if _, err := db.ExecContext(ctx, `
		CREATE TABLE legacy_table (
			id      BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			payload VARCHAR(255) NOT NULL,
			PRIMARY KEY (id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`); err != nil {
		t.Fatalf("构造旧库: %v", err)
	}
	if _, err := db.ExecContext(ctx,
		`INSERT INTO legacy_table (payload) VALUES (?)`, "legacy-row"); err != nil {
		t.Fatalf("写入旧数据: %v", err)
	}

	if err := Migrate(ctx, db); err != nil {
		t.Fatalf("Migrate 旧库: %v", err)
	}

	// 已有数据必须还在（迁移只能在库上追加，不能重建）
	var payload string
	if err := db.QueryRowContext(ctx, `SELECT payload FROM legacy_table`).Scan(&payload); err != nil {
		t.Fatalf("读旧数据: %v", err)
	}
	if payload != "legacy-row" {
		t.Errorf("旧数据被破坏，payload = %q", payload)
	}
}

// TestMigrateDetectsMissingTables 覆盖「迁移记录说已应用、表却不在」。
//
// 这里能成立的前提是 MySQL 允许 DROP TABLE 后保留 schema_migrations 记录：
// 手工改库、从半截备份恢复都可能造成这种状态。必须在启动阶段拦下并给出修法。
func TestMigrateDetectsMissingTables(t *testing.T) {
	db := testutil.New(t)
	ctx := context.Background()

	if err := Migrate(ctx, db); err != nil {
		t.Fatalf("首次 Migrate: %v", err)
	}

	// 模拟外部破坏：删表但保留迁移记录
	if _, err := db.ExecContext(ctx, `DROP TABLE sys_sessions`); err != nil {
		t.Fatalf("删表: %v", err)
	}

	err := Migrate(ctx, db)
	if err == nil {
		t.Fatal("表缺失时 Migrate 应当报错，而不是静默跳过")
	}
	// 错误信息要指出是哪张表、以及怎么修
	for _, want := range []string{"sys_sessions", "DROP DATABASE 后重建"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("错误信息缺少 %q:\n%s", want, err)
		}
	}
}

// TestMigrateAcceptsLegacyDatabase 确认结构校验不会误伤正常的历史库。
func TestMigrateAcceptsLegacyDatabase(t *testing.T) {
	db := testutil.New(t)
	ctx := context.Background()

	// 旧库：有一张与本项目无关的表，没有任何迁移记录
	if _, err := db.ExecContext(ctx,
		`CREATE TABLE devices (id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT, name VARCHAR(64), PRIMARY KEY (id))`); err != nil {
		t.Fatal(err)
	}

	if err := Migrate(ctx, db); err != nil {
		t.Fatalf("正常的历史库不该被结构校验拦下: %v", err)
	}
	// 再跑一次应当幂等
	if err := Migrate(ctx, db); err != nil {
		t.Fatalf("第二次 Migrate: %v", err)
	}
}

// TestMigrateDetectsDirtyVersion 覆盖「迁移执行到一半」的识别。
//
// MySQL 的 DDL 隐式提交，进程在迁移中途被杀会留下 dirty=1 的记录。
// 此时重启必须拒绝，而不是把半截结构当成已完成。
func TestMigrateDetectsDirtyVersion(t *testing.T) {
	db := testutil.New(t)
	ctx := context.Background()

	if err := Migrate(ctx, db); err != nil {
		t.Fatalf("首次 Migrate: %v", err)
	}
	if _, err := db.ExecContext(ctx,
		`UPDATE schema_migrations SET dirty = 1 WHERE version = 2`); err != nil {
		t.Fatal(err)
	}

	err := Migrate(ctx, db)
	if err == nil {
		t.Fatal("dirty 状态下 Migrate 应当拒绝启动")
	}
	if !strings.Contains(err.Error(), "未完成的迁移") {
		t.Errorf("错误信息没有说明 dirty 状态:\n%s", err)
	}
}

func TestSplitStatements(t *testing.T) {
	script := `
-- 行注释里的分号; 不该切
CREATE TABLE a (id INT); # 井号注释
INSERT INTO a VALUES (1);

/* 块注释; 也含分号 */
INSERT INTO a VALUES (2);
`
	got := splitStatements(script)
	if len(got) != 3 {
		t.Fatalf("切出 %d 条语句，期望 3: %#v", len(got), got)
	}
	for i, want := range []string{"CREATE TABLE a", "INSERT INTO a", "INSERT INTO a"} {
		if !strings.HasPrefix(got[i], want) {
			t.Errorf("第 %d 条 = %q，期望以 %q 开头", i, got[i], want)
		}
	}

	// 引号内的分号不能切
	quoted := splitStatements(`INSERT INTO t VALUES ('a;b', "c;d");`)
	if len(quoted) != 1 {
		t.Fatalf("引号内分号被误切: %#v", quoted)
	}
}
