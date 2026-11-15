-- 0010 文件管理
--
-- 元数据进库，文件本体落磁盘（upload.dir）。
-- 不存 BLOB：备份和迁移会变得非常笨重，而且对象存储才是文件该待的地方。
--
-- storage_path 是磁盘上的相对路径（由 UUID 组成），绝不使用用户给的文件名——
-- 直接拿用户输入当路径就是路径穿越。original_name 只用于展示与下载时的文件名。

CREATE TABLE IF NOT EXISTS sys_files (
    id            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    created_at    DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    group_name    VARCHAR(64)     NOT NULL DEFAULT '',
    original_name VARCHAR(255)    NOT NULL,
    storage_path  VARCHAR(255)    NOT NULL,
    size          BIGINT          NOT NULL,
    content_type  VARCHAR(128)    NOT NULL DEFAULT '',
    uploader_id   BIGINT UNSIGNED NOT NULL DEFAULT 0,
    uploader_name VARCHAR(128)    NOT NULL DEFAULT '',
    PRIMARY KEY (id),
    UNIQUE KEY uk_files_storage_path (storage_path),
    KEY idx_files_group (group_name, id),
    KEY idx_files_uploader (uploader_id, id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
