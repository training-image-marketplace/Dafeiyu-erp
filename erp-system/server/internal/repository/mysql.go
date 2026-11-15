// 有问题欢迎联系 hot_kun@hotmail.com
package repository

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/go-sql-driver/mysql"
)

// MySQLConfig 是仓储层需要的连接参数。
//
// 刻意不直接依赖 config 包：repository 只关心「连哪个库、池开多大」，
// 不关心 YAML 长什么样；测试也能用一份最小的配置直接建库。
type MySQLConfig struct {
	Host            string
	Port            int
	User            string
	Password        string
	Name            string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// dsn 组装 go-sql-driver 的连接串。
//
// 用 mysql.Config 而不是 printf 拼字符串：密码里出现 @ : / 等字符时，
// 手拼会把连接串切错，而 FormatDSN 会按驱动自己的规则处理。
func (c MySQLConfig) dsn() string {
	mc := mysql.NewConfig()
	mc.User = c.User
	mc.Passwd = c.Password
	mc.Net = "tcp"
	mc.Addr = net.JoinHostPort(c.Host, strconv.Itoa(c.Port))
	mc.DBName = c.Name
	mc.Collation = "utf8mb4_0900_ai_ci"

	// ParseTime：DATETIME 直接扫进 time.Time。
	// Loc=UTC + 会话 time_zone=+00:00：两处都固定成 UTC，否则
	// CURRENT_TIMESTAMP（服务器时区）和 Go 写入的时间会差 8 小时，
	// 而且是静默的——排序、过期判断全都会错。
	mc.ParseTime = true
	// ClientFoundRows：UPDATE 没有实际改动时也返回「匹配行数」。
	// 否则 requireAffected 会把「保存了但值没变」误判成「记录不存在」。
	mc.ClientFoundRows = true
	mc.Loc = time.UTC
	mc.Params = map[string]string{
		"charset":   "utf8mb4",
		"time_zone": "'+00:00'",
	}

	mc.Timeout = 5 * time.Second
	mc.ReadTimeout = 30 * time.Second
	mc.WriteTimeout = 30 * time.Second

	return mc.FormatDSN()
}

// NewDB 打开 MySQL 连接并按服务端场景配置连接池。
func NewDB(c MySQLConfig) (*sql.DB, error) {
	db, err := sql.Open("mysql", c.dsn())
	if err != nil {
		return nil, fmt.Errorf("open mysql: %w", err)
	}

	db.SetMaxOpenConns(c.MaxOpenConns)
	db.SetMaxIdleConns(c.MaxIdleConns)
	db.SetConnMaxLifetime(c.ConnMaxLifetime)

	// 提前 Ping，让账号、库名、网络错误在启动时就暴露，
	// 而不是等到第一个请求才 500。
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping mysql %s:%d/%s: %w", c.Host, c.Port, c.Name, err)
	}
	return db, nil
}
