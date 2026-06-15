-- 路由去 /po 前缀：更新已有环境的菜单 path（install.sql 新装环境无需执行）
UPDATE `zt_menus` SET `path` = '/home' WHERE `path` = '/po/home' AND `deletedAt` IS NULL;
UPDATE `zt_menus` SET `path` = '/schedule' WHERE `path` = '/po/schedule' AND `deletedAt` IS NULL;
