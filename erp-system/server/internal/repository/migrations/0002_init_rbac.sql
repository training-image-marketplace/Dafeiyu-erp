-- 0002 RBAC 基础表（MySQL 8.0.16+）
--
-- 与 mini-ruoyi 的 SQLite 版相比：
--   * 主键改 BIGINT UNSIGNED AUTO_INCREMENT
--   * 时间列用 DATETIME(3)，与 CURRENT_TIMESTAMP(3) 精度一致
--   * VARCHAR 必须给长度，MySQL 不接受裸 varchar
--   * CHECK 约束从 8.0.16 起才真正生效；约束名在全库必须唯一
--   * 索引直接写进 CREATE TABLE，减少迁移语句数

CREATE TABLE IF NOT EXISTS sys_users (
    id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    created_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    status      VARCHAR(16)     NOT NULL DEFAULT 'active',
    username    VARCHAR(128)    NOT NULL,
    password    VARCHAR(255)    NOT NULL,
    nickname    VARCHAR(128)    NOT NULL DEFAULT '',
    mobile      VARCHAR(20)     NOT NULL DEFAULT '',
    email       VARCHAR(64)     NOT NULL DEFAULT '',
    login_ip    VARCHAR(64)     NOT NULL DEFAULT '',
    login_at    DATETIME(3)     NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uk_users_username (username),
    CONSTRAINT ck_users_status CHECK (status IN ('active','inactive'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS sys_roles (
    id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    created_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    status      VARCHAR(16)     NOT NULL DEFAULT 'active',
    code        VARCHAR(64)     NOT NULL,
    name        VARCHAR(128)    NOT NULL,
    remark      VARCHAR(255)    NOT NULL DEFAULT '',
    PRIMARY KEY (id),
    UNIQUE KEY uk_roles_code (code),
    CONSTRAINT ck_roles_status CHECK (status IN ('active','inactive'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS sys_menus (
    id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    created_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    status      VARCHAR(16)     NOT NULL DEFAULT 'active',
    parent_id   BIGINT UNSIGNED NULL,
    sort        INT             NOT NULL DEFAULT 0,
    menu_type   VARCHAR(16)     NOT NULL DEFAULT 'directory',
    title_key   VARCHAR(128)    NOT NULL,
    path        VARCHAR(255)    NOT NULL DEFAULT '',
    component   VARCHAR(255)    NOT NULL DEFAULT '',
    icon        VARCHAR(64)     NOT NULL DEFAULT '',
    PRIMARY KEY (id),
    KEY idx_menus_parent_sort (parent_id, sort),
    CONSTRAINT fk_menus_parent FOREIGN KEY (parent_id) REFERENCES sys_menus (id) ON DELETE CASCADE,
    CONSTRAINT ck_menus_type CHECK (menu_type IN ('directory','menu'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS sys_user_roles (
    user_id BIGINT UNSIGNED NOT NULL,
    role_id BIGINT UNSIGNED NOT NULL,
    PRIMARY KEY (user_id, role_id),
    KEY idx_user_roles_role (role_id),
    CONSTRAINT fk_user_roles_user FOREIGN KEY (user_id) REFERENCES sys_users (id) ON DELETE CASCADE,
    CONSTRAINT fk_user_roles_role FOREIGN KEY (role_id) REFERENCES sys_roles (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS sys_role_menus (
    role_id BIGINT UNSIGNED NOT NULL,
    menu_id BIGINT UNSIGNED NOT NULL,
    PRIMARY KEY (role_id, menu_id),
    KEY idx_role_menus_menu (menu_id),
    CONSTRAINT fk_role_menus_role FOREIGN KEY (role_id) REFERENCES sys_roles (id) ON DELETE CASCADE,
    CONSTRAINT fk_role_menus_menu FOREIGN KEY (menu_id) REFERENCES sys_menus (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS sys_role_perms (
    role_id   BIGINT UNSIGNED NOT NULL,
    perm_code VARCHAR(64)     NOT NULL,
    PRIMARY KEY (role_id, perm_code),
    KEY idx_role_perms_code (perm_code),
    CONSTRAINT fk_role_perms_role FOREIGN KEY (role_id) REFERENCES sys_roles (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
