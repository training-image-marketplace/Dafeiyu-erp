-- 0013 登录失败锁定
--
-- 状态落在用户行上，而不是单独建一张逐次记录的失败表：
-- 要回答的问题只有「这个账号现在能不能登录」，两个列就够了。
--
-- failed_count 是连续失败次数（成功登录即清零）；
-- locked_until 为 NULL 表示未锁定，到点后由下一次登录尝试自动解锁并清零计数。

ALTER TABLE sys_users
    ADD COLUMN failed_count INT NOT NULL DEFAULT 0,
    ADD COLUMN locked_until DATETIME(3) NULL;
