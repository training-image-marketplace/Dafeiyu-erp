把 Android 安装包放在这个目录，文件名用 dafeiyu-erp.apk。

- 开发/构建：放好后再 npm run build，文件会被复制到 dist/apk/。
- 已部署：直接把 apk 丢进 <APP_WEB_DIR>/apk/ 即可，不用重新构建，也不用重启后端。
- 页面入口：ERP 侧边栏「Android 客户端」（/erp/android），下载地址 /apk/dafeiyu-erp.apk。
