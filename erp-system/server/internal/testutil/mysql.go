// Package testutil 提供 MySQL 集成测试的公共设施。
//
// 每个测试用例都会拿到一个**独立的数据库**（随机库名），结束时 DROP。
// 这样测试之间不共享状态，可以并行，也不会因为残留数据互相影响。
//
// 通过 TEST_MYSQL_DSN 指定实例级连接串（不带库名），例如：
//
//	TEST_MYSQL_DSN='root:root@tcp(127.0.0.1:3307)/' go test ./...
//
// 未设置时相关测试自动 skip，保证没有 MySQL 的环境里 `go test ./...` 仍然能过。
package testutil

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
)

// EnvDSN 是测试用 MySQL 实例连接串的环境变量名。
const EnvDSN = "TEST_MYSQL_DSN"

// New 创建一个独立数据库并返回指向它的连接。
//
// 用 testing.TB 而不是 *testing.T：以后想在 Benchmark 或 Fuzz 里复用也成立。
func New(t testing.TB) *sql.DB {
	t.Helper()

	base := os.Getenv(EnvDSN)
	if base == "" {
		t.Skipf("未设置 %s，跳过 MySQL 集成测试（示例：root:root@tcp(127.0.0.1:3307)/）", EnvDSN)
	}

	cfg, err := mysql.ParseDSN(base)
	if err != nil {
		t.Fatalf("解析 %s: %v", EnvDSN, err)
	}
	// 测试连接与生产连接保持同样的时间语义，否则「本地测试全绿、线上差 8 小时」。
	cfg.ParseTime = true
	cfg.ClientFoundRows = true
	cfg.Loc = time.UTC
	if cfg.Params == nil {
		cfg.Params = map[string]string{}
	}
	cfg.Params["charset"] = "utf8mb4"
	cfg.Params["time_zone"] = "'+00:00'"

	admin, err := sql.Open("mysql", base)
	if err != nil {
		t.Fatalf("连接 MySQL 实例: %v", err)
	}
	if err := admin.Ping(); err != nil {
		admin.Close()
		t.Fatalf("Ping MySQL 实例（检查容器是否已启动）: %v", err)
	}

	var buf [6]byte
	if _, err := rand.Read(buf[:]); err != nil {
		admin.Close()
		t.Fatalf("生成测试库名: %v", err)
	}
	name := "pt_test_" + hex.EncodeToString(buf[:])

	if _, err := admin.Exec(fmt.Sprintf(
		"CREATE DATABASE `%s` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci", name)); err != nil {
		admin.Close()
		t.Fatalf("创建测试库 %s: %v", name, err)
	}

	cfg.DBName = name
	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		admin.Exec("DROP DATABASE `" + name + "`")
		admin.Close()
		t.Fatalf("连接测试库 %s: %v", name, err)
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)

	t.Cleanup(func() {
		db.Close()
		if _, err := admin.Exec("DROP DATABASE `" + name + "`"); err != nil {
			t.Logf("清理测试库 %s 失败（可忽略，下次会重新建）: %v", name, err)
		}
		admin.Close()
	})
	return db
}
