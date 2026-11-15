# 部署文件（大肥鱼 ERP）

本目录只部署 ERP 自己：一个 MySQL + 一个前后端单镜像。**不包含实训平台**——
平台在另一个仓库里，它把这个镜像当「被测系统」登记即可。

## 一键启动

```bash
cd deploy
cp .env.example .env        # 改掉 MYSQL_ROOT_PASSWORD 与 INTERNAL_TOKEN
docker compose up -d --build
```

| 服务 | 地址 | 说明 |
| --- | --- | --- |
| MySQL | `localhost:23307` | 首次启动建 `pt_erp` |
| 大肥鱼 ERP | http://localhost:8889 | 默认账号 `admin / admin123`，演示账号见 [README](../README.md) |

> 端口默认避开平台的 23306/8888。宿主机已有 MySQL 时改 `.env` 的 `MYSQL_PORT`，
> 或删掉 `mysql` 服务、把 `APP_DB_HOST` 指向既有实例。

## 单独跑（复用已有 MySQL）

```bash
mysql -h127.0.0.1 -P23306 -uroot -p < init/01-databases.sql

cd ../erp-system/server
APP_DB_HOST=127.0.0.1 APP_DB_PORT=23306 APP_DB_USER=root APP_DB_PASSWORD='***' \
  go run ./cmd/server
```

## 作为平台镜像运行

平台会用自己的 `instance_db` 给每个学生建独立空库，并改写容器启动命令里的
`APP_DB_NAME`；镜像本身不用改。镜像广场里登记时：

- 端口：`8889`
- 启动命令：见 [doc/07](../doc/07-镜像与部署.md)
- 数据目录：`/app/uploads`

## 许可证

MIT，见 [LICENSE](../LICENSE)。
