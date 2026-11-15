// 有问题欢迎联系 hot_kun@hotmail.com
package repository

import (
	"context"
	"testing"
	"time"

	"practical-training/erp-system/internal/domain"
	"practical-training/erp-system/internal/testutil"
)

func newSessionRepo(t *testing.T) (*SessionRepository, *UserRepository, context.Context) {
	t.Helper()

	db := testutil.New(t)

	ctx := context.Background()
	if err := Migrate(ctx, db); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	return NewSessionRepository(db), NewUserRepository(db), ctx
}

func adminID(t *testing.T, users *UserRepository, ctx context.Context) int64 {
	t.Helper()
	u, err := users.GetByUsername(ctx, "admin")
	if err != nil {
		t.Fatalf("取 admin 用户: %v", err)
	}
	return u.ID
}

// TestSessionTimesAreStoredAndComparedInUTC 覆盖时间语义。
//
// MySQL 的 DATETIME 不带时区，驱动返回 time.Time 时按连接 loc 解释。
// 如果 DSN 的 loc 与会话 time_zone 不一致，同一条记录会被读成相差 8 小时的
// 时刻，过期判断会静默出错。这里断言：写入的 UTC 时刻能原样读回，
// 数据库默认值（CURRENT_TIMESTAMP）与 Go 绑定的值也可以直接比较。
func TestSessionTimesAreStoredAndComparedInUTC(t *testing.T) {
	sessions, users, ctx := newSessionRepo(t)

	expires := time.Now().UTC().Add(time.Hour).Truncate(time.Second)
	if err := sessions.Create(ctx, domain.Session{
		TokenHash: "hash-format", UserID: adminID(t, users, ctx),
		ExpiresAt: expires, CSRFToken: "csrf",
	}); err != nil {
		t.Fatalf("Create: %v", err)
	}

	var (
		createdAt time.Time
		expiresAt time.Time
		lastSeen  time.Time
	)
	if err := sessions.db.QueryRowContext(ctx,
		`SELECT created_at, expires_at, last_seen_at FROM sys_sessions WHERE token_hash = ?`,
		"hash-format").Scan(&createdAt, &expiresAt, &lastSeen); err != nil {
		t.Fatal(err)
	}

	if !expiresAt.Equal(expires) {
		t.Errorf("expires_at 读回 %v，写入 %v（时区偏移？）", expiresAt, expires)
	}
	if createdAt.Location() != time.UTC || expiresAt.Location() != time.UTC || lastSeen.Location() != time.UTC {
		t.Errorf("时间列没有按 UTC 解释: created=%v expires=%v last_seen=%v",
			createdAt.Location(), expiresAt.Location(), lastSeen.Location())
	}
	if !createdAt.Before(expiresAt) {
		t.Error("created_at 应当早于 expires_at")
	}
}

// queryOne 是只取一列的最小查询辅助，避免为几个断言引入更多结构体。
func queryOne(ctx context.Context, r *SessionRepository, q string, dest any) error {
	return r.db.QueryRowContext(ctx, q).Scan(dest)
}

// TestSessionTimeRoundTrip 确认写进去再读出来是同一个时刻。
func TestSessionTimeRoundTrip(t *testing.T) {
	sessions, users, ctx := newSessionRepo(t)

	expires := time.Now().UTC().Add(7 * 24 * time.Hour).Truncate(time.Second)
	if err := sessions.Create(ctx, domain.Session{
		TokenHash: "hash-roundtrip", UserID: adminID(t, users, ctx),
		ExpiresAt: expires, CSRFToken: "csrf",
	}); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := sessions.GetByTokenHash(ctx, "hash-roundtrip")
	if err != nil {
		t.Fatalf("GetByTokenHash: %v", err)
	}
	if !got.ExpiresAt.Equal(expires) {
		t.Errorf("ExpiresAt 往返后 = %v，期望 %v", got.ExpiresAt, expires)
	}
}

// TestDeleteExpired 覆盖过期清理。它同时是时间格式的一道防线：
// 如果 expires_at 与传入的 now 时区不一致，比较会得出错误结论。
func TestDeleteExpired(t *testing.T) {
	sessions, users, ctx := newSessionRepo(t)
	uid := adminID(t, users, ctx)
	now := time.Now().UTC()

	mk := func(hash string, expiresAt time.Time) {
		t.Helper()
		if err := sessions.Create(ctx, domain.Session{
			TokenHash: hash, UserID: uid, ExpiresAt: expiresAt, CSRFToken: "csrf",
		}); err != nil {
			t.Fatalf("Create %s: %v", hash, err)
		}
	}
	mk("expired-1h", now.Add(-time.Hour))
	mk("expired-8h", now.Add(-8*time.Hour)) // 跨时区偏移的边界量级
	mk("valid-1h", now.Add(time.Hour))
	mk("valid-7d", now.Add(7*24*time.Hour))

	removed, err := sessions.DeleteExpired(ctx, now)
	if err != nil {
		t.Fatalf("DeleteExpired: %v", err)
	}
	if removed != 2 {
		t.Errorf("DeleteExpired 删除了 %d 条，期望 2（两条已过期）", removed)
	}

	for _, tc := range []struct {
		hash string
		want bool // 期望仍存在
	}{
		{"expired-1h", false},
		{"expired-8h", false},
		{"valid-1h", true},
		{"valid-7d", true},
	} {
		_, err := sessions.GetByTokenHash(ctx, tc.hash)
		exists := err == nil
		if exists != tc.want {
			t.Errorf("%s 存在性 = %v，期望 %v", tc.hash, exists, tc.want)
		}
	}
}

// TestDeleteByUserRevokesAllSessions 覆盖「踢人」：停用或改密码后要一次清干净。
func TestDeleteByUserRevokesAllSessions(t *testing.T) {
	sessions, users, ctx := newSessionRepo(t)
	uid := adminID(t, users, ctx)
	expires := time.Now().UTC().Add(time.Hour)

	for _, h := range []string{"s1", "s2", "s3"} {
		if err := sessions.Create(ctx, domain.Session{
			TokenHash: h, UserID: uid, ExpiresAt: expires, CSRFToken: "csrf",
		}); err != nil {
			t.Fatal(err)
		}
	}

	if err := sessions.DeleteByUser(ctx, uid); err != nil {
		t.Fatalf("DeleteByUser: %v", err)
	}
	for _, h := range []string{"s1", "s2", "s3"} {
		if _, err := sessions.GetByTokenHash(ctx, h); err == nil {
			t.Errorf("会话 %s 未被清除", h)
		}
	}
}

// TestDeletingUserCascadesSessions 确认外键级联真的生效——
// 删掉用户后会话必须跟着消失，否则会留下引用空用户的「幽灵会话」。
func TestDeletingUserCascadesSessions(t *testing.T) {
	sessions, users, ctx := newSessionRepo(t)

	u, err := users.Create(ctx, domain.User{
		Status: domain.StatusActive, Username: "temp", Password: "x",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := sessions.Create(ctx, domain.Session{
		TokenHash: "cascade", UserID: u.ID,
		ExpiresAt: time.Now().UTC().Add(time.Hour), CSRFToken: "csrf",
	}); err != nil {
		t.Fatal(err)
	}

	if err := users.Delete(ctx, u.ID); err != nil {
		t.Fatalf("Delete user: %v", err)
	}
	if _, err := sessions.GetByTokenHash(ctx, "cascade"); err == nil {
		t.Error("删除用户后会话仍然存在，外键级联没有生效")
	}
}
