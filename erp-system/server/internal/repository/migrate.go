// 有问题欢迎联系 hot_kun@hotmail.com
package repository

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

// migrateLockName 是 MySQL 命名锁，防止多个实例同时启动时并发执行迁移。
//
// GET_LOCK 是**连接级**的：必须拿同一条连接加锁和释放，所以 Migrate
// 全程持有一个 *sql.Conn，而不是从连接池里随手取。
const migrateLockName = "practical_training_migrate"

// createSchemaMigrations 建迁移记录表。dirty=1 表示「这个版本的脚本执行到一半」。
//
// MySQL 的 DDL 会隐式提交，没有 SQLite 那种「整个迁移在一个事务里」的保证。
// 所以不能只记录「跑过/没跑过」，还要能区分「跑了一半」：否则崩溃后重启会
// 认为该版本已完成，而库结构其实缺了一半，之后每个请求都以各种形态报错。
const createSchemaMigrations = `
CREATE TABLE IF NOT EXISTS schema_migrations (
	version    BIGINT       NOT NULL,
	name       VARCHAR(255) NOT NULL,
	dirty      TINYINT(1)   NOT NULL DEFAULT 0,
	applied_at DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
	PRIMARY KEY (version)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci`

// Migrate 按文件名数字前缀顺序执行 migrations/ 下的 SQL，已执行版本记录在
// schema_migrations 表里。
//
// 只做向前补齐，不提供回滚：改错了一个迁移的正确做法是再加一个修复迁移。
func Migrate(ctx context.Context, db *sql.DB) error {
	// 迁移全程用同一条连接：GET_LOCK/RELEASE_LOCK 必须落在同一会话上
	conn, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("migrate 获取连接: %w", err)
	}
	defer conn.Close()

	var locked sql.NullInt64
	if err := conn.QueryRowContext(ctx, `SELECT GET_LOCK(?, 30)`, migrateLockName).Scan(&locked); err != nil {
		return fmt.Errorf("获取迁移锁: %w", err)
	}
	if !locked.Valid || locked.Int64 != 1 {
		return fmt.Errorf("获取迁移锁超时：另一个实例正在执行迁移")
	}
	defer func() {
		var released sql.NullInt64
		_ = conn.QueryRowContext(context.Background(),
			`SELECT RELEASE_LOCK(?)`, migrateLockName).Scan(&released)
	}()

	if _, err := conn.ExecContext(ctx, createSchemaMigrations); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	applied, dirty, err := appliedVersions(ctx, conn)
	if err != nil {
		return err
	}
	if len(dirty) > 0 {
		return fmt.Errorf(
			"检测到未完成的迁移：%v\n"+
				"  上一次迁移执行到一半（DDL 非事务，MySQL 无法自动回滚）。\n"+
				"  处理方式：人工核对库结构，补齐或删除残留对象后，\n"+
				"  删除 schema_migrations 中对应 version 的记录再重启",
			dirty)
	}

	names, err := fs.Glob(migrationFS, "migrations/*.sql")
	if err != nil {
		return err
	}
	sort.Strings(names) // 文件名用零填充数字前缀，字典序即版本序

	var bodies []string
	for _, name := range names {
		base := path.Base(name)
		version, err := parseMigrationVersion(base)
		if err != nil {
			return err
		}

		body, err := migrationFS.ReadFile(name)
		if err != nil {
			return err
		}
		bodies = append(bodies, string(body))

		if applied[version] {
			continue
		}
		if err := applyMigration(ctx, conn, version, base, string(body)); err != nil {
			return err
		}
	}

	// 结构校验放在最后：只信 schema_migrations 是不够的，见 verifySchema 的说明
	return verifySchema(ctx, conn, expectedTables(bodies))
}

func appliedVersions(ctx context.Context, conn *sql.Conn) (map[int64]bool, []int64, error) {
	rows, err := conn.QueryContext(ctx, `SELECT version, dirty FROM schema_migrations`)
	if err != nil {
		return nil, nil, fmt.Errorf("read schema_migrations: %w", err)
	}
	defer rows.Close()

	applied := map[int64]bool{}
	var dirty []int64
	for rows.Next() {
		var (
			v   int64
			bad bool
		)
		if err := rows.Scan(&v, &bad); err != nil {
			return nil, nil, err
		}
		applied[v] = true
		if bad {
			dirty = append(dirty, v)
		}
	}
	return applied, dirty, rows.Err()
}

// parseMigrationVersion 解析 "<版本号>_<描述>.sql" 里的版本号。
func parseMigrationVersion(filename string) (int64, error) {
	prefix, _, ok := strings.Cut(filename, "_")
	if !ok {
		return 0, fmt.Errorf("迁移文件 %q 不符合 <版本号>_<描述>.sql 命名", filename)
	}
	v, err := strconv.ParseInt(prefix, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("迁移文件 %q 的版本号不是数字: %w", filename, err)
	}
	return v, nil
}

// applyMigration 先登记 dirty，再逐条执行语句，成功后清除 dirty。
//
// 逐条执行（而不是依赖 multiStatements）有两个好处：
//  1. 不需要在全局 DSN 上打开 multiStatements——那会让任何一处拼接 SQL
//     的疏忽都变成可执行多语句的放大器；
//  2. 失败时能报出是第几条语句，定位比「整个文件语法错误」精确。
func applyMigration(ctx context.Context, conn *sql.Conn, version int64, name, body string) error {
	stmts := splitStatements(body)
	if len(stmts) == 0 {
		return fmt.Errorf("迁移 %s 没有任何语句", name)
	}

	if _, err := conn.ExecContext(ctx,
		`INSERT INTO schema_migrations (version, name, dirty) VALUES (?, ?, 1)`,
		version, name); err != nil {
		return fmt.Errorf("登记迁移 %s: %w", name, err)
	}

	for i, stmt := range stmts {
		if _, err := conn.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf(
				"apply migration %s 第 %d/%d 条语句失败（已标记 dirty）: %w\n语句: %.200s",
				name, i+1, len(stmts), err, stmt)
		}
	}

	if _, err := conn.ExecContext(ctx,
		`UPDATE schema_migrations SET dirty = 0, applied_at = CURRENT_TIMESTAMP(3) WHERE version = ?`,
		version); err != nil {
		return fmt.Errorf("完成迁移 %s: %w", name, err)
	}
	return nil
}

// splitStatements 把迁移脚本切成单条语句。
//
// 需要识别引号与注释，否则注释里的分号、字符串里的分号都会切错。
// 只覆盖 MySQL 迁移脚本会用到的语法：-- 与 # 行注释、/* */ 块注释、
// 单引号/双引号/反引号，以及反斜杠转义。
func splitStatements(script string) []string {
	var (
		out   []string
		b     strings.Builder
		quote byte // 0 表示不在引号内
	)

	flush := func() {
		if s := strings.TrimSpace(b.String()); s != "" {
			out = append(out, s)
		}
		b.Reset()
	}

	for i := 0; i < len(script); i++ {
		c := script[i]

		if quote != 0 {
			b.WriteByte(c)
			if c == '\\' && quote != '`' && i+1 < len(script) {
				i++
				b.WriteByte(script[i])
				continue
			}
			if c == quote {
				quote = 0
			}
			continue
		}

		switch {
		case c == '\'' || c == '"' || c == '`':
			quote = c
			b.WriteByte(c)

		case c == '-' && i+1 < len(script) && script[i+1] == '-':
			for i < len(script) && script[i] != '\n' {
				i++
			}
			b.WriteByte('\n')

		case c == '#':
			for i < len(script) && script[i] != '\n' {
				i++
			}
			b.WriteByte('\n')

		case c == '/' && i+1 < len(script) && script[i+1] == '*':
			i += 2
			for i+1 < len(script) && !(script[i] == '*' && script[i+1] == '/') {
				i++
			}
			i++ // 跳过收尾的 '/'

		case c == ';':
			flush()

		default:
			b.WriteByte(c)
		}
	}
	flush()
	return out
}

var createTableRe = regexp.MustCompile(`(?is)CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?([a-zA-Z_][a-zA-Z0-9_]*)`)

// expectedTables 从所有迁移文件里提取应该存在的表名。
func expectedTables(bodies []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, body := range bodies {
		for _, m := range createTableRe.FindAllStringSubmatch(body, -1) {
			name := strings.ToLower(m[1])
			if name == "schema_migrations" || seen[name] {
				continue
			}
			seen[name] = true
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

// verifySchema 确认迁移文件声明过的表都真实存在，缺失则报错。
//
// schema_migrations 只说明「迁移跑过」，不说明「表还在」。库被手工改过、
// 从半截备份恢复、迁移执行到一半崩溃，都会出现版本号说已应用但表缺失的情况。
// 这时代码不会报「缺表」，而是每个接口各自 500，排查成本极高。
func verifySchema(ctx context.Context, conn *sql.Conn, want []string) error {
	var missing []string
	for _, table := range want {
		var n int
		err := conn.QueryRowContext(ctx, `
			SELECT COUNT(*) FROM information_schema.tables
			WHERE table_schema = DATABASE() AND table_name = ?`, table).Scan(&n)
		if err != nil {
			return fmt.Errorf("检查表 %s 是否存在: %w", table, err)
		}
		if n == 0 {
			missing = append(missing, table)
		}
	}
	if len(missing) == 0 {
		return nil
	}

	return fmt.Errorf(
		"数据库结构不完整：迁移记录显示已应用，但以下表不存在：%s\n"+
			"  可能原因：库被手工改过、从半途中断的备份恢复、或迁移执行到一半。\n"+
			"  修复方式：\n"+
			"    - 全新部署（没有要保留的数据）：DROP DATABASE 后重建，再启动\n"+
			"    - 已有数据：从备份恢复，或手工补齐缺失的表",
		strings.Join(missing, ", "))
}
