-- 0008 定时任务
--
-- 任务本体在代码里（internal/job），这张表只存可调参数：
-- 启用/禁用、cron 表达式，以及最近一次执行结果。
--
-- job_key 由代码声明，启动时按注册表 upsert，所以界面不需要「新建任务」。

CREATE TABLE IF NOT EXISTS sys_jobs (
    id               BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    created_at       DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at       DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    status           VARCHAR(16)     NOT NULL DEFAULT 'active',
    job_key          VARCHAR(64)     NOT NULL,
    cron             VARCHAR(64)     NOT NULL,
    remark           VARCHAR(255)    NOT NULL DEFAULT '',
    last_run_at      DATETIME(3)     NULL,
    last_status      VARCHAR(16)     NOT NULL DEFAULT '',
    last_error       VARCHAR(255)    NOT NULL DEFAULT '',
    last_duration_ms INT             NOT NULL DEFAULT 0,
    PRIMARY KEY (id),
    UNIQUE KEY uk_jobs_key (job_key),
    CONSTRAINT ck_jobs_status CHECK (status IN ('active','inactive')),
    CONSTRAINT ck_jobs_last_status CHECK (last_status IN ('','success','failed','skipped'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
