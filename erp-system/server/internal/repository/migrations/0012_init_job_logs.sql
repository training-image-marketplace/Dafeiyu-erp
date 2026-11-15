-- 0012 定时任务执行日志
--
-- 只追加、不修改，所以没有 updated_at；job_key 不设外键，
-- 任务从代码里删掉后它跑过的记录必须留下。
--
-- 注意：trigger 是 MySQL 保留字，列名改为 trigger_type。
-- 沿用 SQLite 时的列名会让迁移直接语法错误，而且报错信息不会告诉你
-- 「是保留字」——只会指在 trigger 这个位置。

CREATE TABLE IF NOT EXISTS sys_job_logs (
    id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    created_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    job_key     VARCHAR(64)     NOT NULL,
    trigger_type VARCHAR(16)    NOT NULL,
    status      VARCHAR(16)     NOT NULL,
    error       VARCHAR(255)    NOT NULL DEFAULT '',
    duration_ms INT             NOT NULL DEFAULT 0,
    PRIMARY KEY (id),
    KEY idx_job_logs_key (job_key, id),
    KEY idx_job_logs_created (created_at),
    CONSTRAINT ck_job_logs_trigger CHECK (trigger_type IN ('cron','manual')),
    CONSTRAINT ck_job_logs_status CHECK (status IN ('success','failed','skipped'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
