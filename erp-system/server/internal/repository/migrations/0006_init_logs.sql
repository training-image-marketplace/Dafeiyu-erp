-- 0006 操作日志与登录日志
--
-- 两张表都只做「追加 + 定期删旧」：
--   * 没有 updated_at —— 日志被改过就失去审计价值
--   * user_id 不设外键 —— 用户被删除后日志必须留下，否则删号就能抹掉痕迹
--     username 也冗余存一份，同理
--
-- 保留期由应用层控制（默认 30 天），因此 created_at 上需要索引支撑删除。

CREATE TABLE IF NOT EXISTS sys_login_logs (
    id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    created_at DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    username   VARCHAR(128)    NOT NULL,
    status     VARCHAR(16)     NOT NULL,
    reason     VARCHAR(64)     NOT NULL DEFAULT '',
    ip         VARCHAR(64)     NOT NULL DEFAULT '',
    user_agent VARCHAR(255)    NOT NULL DEFAULT '',
    PRIMARY KEY (id),
    KEY idx_login_logs_created (created_at),
    KEY idx_login_logs_user (username, created_at),
    CONSTRAINT ck_login_logs_status CHECK (status IN ('success','failed'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS sys_oper_logs (
    id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    created_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    user_id     BIGINT UNSIGNED NOT NULL DEFAULT 0,
    username    VARCHAR(128)    NOT NULL DEFAULT '',
    method      VARCHAR(10)     NOT NULL,
    path        VARCHAR(255)    NOT NULL,
    status      INT             NOT NULL,
    result      VARCHAR(64)     NOT NULL DEFAULT '',
    duration_ms INT             NOT NULL DEFAULT 0,
    ip          VARCHAR(64)     NOT NULL DEFAULT '',
    user_agent  VARCHAR(255)    NOT NULL DEFAULT '',
    PRIMARY KEY (id),
    KEY idx_oper_logs_created (created_at),
    KEY idx_oper_logs_user (user_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
