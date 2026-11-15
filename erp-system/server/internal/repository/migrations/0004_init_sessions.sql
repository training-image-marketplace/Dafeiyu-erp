-- 0004 会话表
--
-- 认证方案是 Cookie + 服务端 session，因此需要服务端表存会话状态，
-- 这样「登出 / 踢人 / 禁用用户」才能立即生效。
--
-- token 存 SHA-256 而不是明文：数据库一旦泄露，明文 token 等于把所有人的
-- 会话直接送出去；哈希值无法反推。

CREATE TABLE IF NOT EXISTS sys_sessions (
    token_hash   VARCHAR(64)     NOT NULL,
    user_id      BIGINT UNSIGNED NOT NULL,
    created_at   DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    expires_at   DATETIME(3)     NOT NULL,
    last_seen_at DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    csrf_token   VARCHAR(64)     NOT NULL,
    ip           VARCHAR(64)     NOT NULL DEFAULT '',
    user_agent   VARCHAR(255)    NOT NULL DEFAULT '',
    PRIMARY KEY (token_hash),
    KEY idx_sessions_user (user_id),
    KEY idx_sessions_expires (expires_at),
    CONSTRAINT fk_sessions_user FOREIGN KEY (user_id) REFERENCES sys_users (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
