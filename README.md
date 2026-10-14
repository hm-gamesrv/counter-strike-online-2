# CSO2 Server

Counter-Strike Online 2 服务端。用于 **2017 年国服** 和 **2018 年韩服** 客户端。

> *Counter-Strike Online 2 归 NEXON 所有，本项目仅用于学习之用。*

## 1. 简述

本项目是 [LoveBeforT/CSO2-Server](https://github.com/LoveBeforT/CSO2-Server) 的改进版本，主要改进：

- **纯 Go 静态编译**：移除 cgo/iconv 依赖，编码转换使用 `golang.org/x/text` 纯 Go 实现
- **单文件发布**：配置、本地化、游戏数据与管理页面全部编进二进制（`go:embed`），不需要额外资源目录
- **Web 管理后台**：取代旧的明文 TCP 控制台，账号增删改、发物品、踢人、关房间都在浏览器里完成
- **优化的容器化运行**：支持 Docker / Podman 一行命令启动
- **GitHub Actions 自动构建**：推送镜像到 GHCR，无需本地编译

**镜像：**

| 仓库 | tag |
| ---- | --- |
| `ghcr.io/hm-gamesrv/counter-strike-online-2` | `latest`、`yyyy-MM-dd` |

## 2. 资源占用信息

### 2.1. 端口

| 端口号 | 协议 | 说明           |
| ------ | ---- | -------------- |
| 1315   | TCP  | 管理后台       |
| 30001  | TCP  | 游戏主连接     |
| 30002  | UDP  | 联机通信       |

### 2.2. 持久卷

| 容器路径 | 说明                                      |
| -------- | ----------------------------------------- |
| `/data`  | 玩家数据、举报记录与可选的 `server.yaml`  |

### 2.3. 环境变量

| 变量           | 默认                  | 说明                                          |
| -------------- | --------------------- | --------------------------------------------- |
| `ADMIN_TOKEN`  | 无（**必填**）        | 管理令牌，16–128 位字母 / 数字 / 下划线       |
| `DATA_DIR`     | `/data`               | 数据目录，相对路径按工作目录解析              |
| `TCP_PORT`     | `30001`               | 游戏端口                                      |
| `UDP_PORT`     | `30002`               | UDP 打洞端口                                  |
| `ADMIN_PORT`   | `1315`                | 管理后台端口                                  |
| `ENABLE_ADMIN` | `true`                | 设为 `false` 则不启动管理后台                 |
| `LOCALE`       | `zh-cn`               | 形如 `zh-cn`，不带 `.ini`                     |
| `CODE_PAGE`    | 由 `LOCALE` 推导      | 客户端字符集，一般不用设置                    |

没有合法的 `ADMIN_TOKEN` 时管理后台**不会启动**（游戏服不受影响），这是有意的：
宁可没有管理接口，也不要一个谁都能调的。

## 3. 快速开始（Docker / Podman）

### 3.1. 构建并运行（Docker）

```bash
docker build -t cso2:temp . && \
    docker run --rm -it \
        -e ADMIN_TOKEN=ChangeMe_0123456789 \
        -p 1315:1315 \
        -p 30001:30001 \
        -p 30002:30002/udp \
        -v ./data:/data \
        cso2:temp
```

### 3.2. 运行服务器（Podman）

```bash
IMAGE=ghcr.io/hm-gamesrv/counter-strike-online-2:latest

if ! podman pull "$IMAGE"; then
    exit 1
fi

podman run --rm -it \
    --name cso2 \
    --userns keep-id \
    --network pasta \
    -e ADMIN_TOKEN=ChangeMe_0123456789 \
    -p 1315:1315 \
    -p 30001:30001 \
    -p 30002:30002/udp \
    -v ./data:/data \
    "$IMAGE"
```

### 3.3. 使用 Compose

```bash
export ADMIN_TOKEN=ChangeMe_0123456789
docker compose up -d
```

`docker-compose.yml` 会把宿主机的 `./data` 挂到容器的 `/data`。

> **首次运行前先建好数据目录：`mkdir -p data`。**
> 容器内以 UID/GID 1000 运行，而目录不存在时 Docker 会以 root 创建它，
> 容器随即启动失败（`mkdir /data/users: permission denied`）。
> 自己建出来的目录属于当前用户；只有宿主 uid 不是 1000 时才需要
> `sudo chown 1000:1000 data`。

## 4. 服务器设置

设置分两类：

- **部署**（这个实例怎么跑）——环境变量，见 2.3。
- **玩法**（服务器怎么表现）——`$DATA_DIR/server.yaml`，可选，**增量覆盖**内置默认值：

```yaml
server:
  max_users: 0            # 同时在线上限，0 = 不限
  enable_shop: true
  unlock_all_weapons: true

debug:
  level: 2                # 0/1/2，只影响调试日志数量
```

改了配置不用重建镜像，重启进程即可。写错键名会**启动失败并打印行号**，
而不是静默忽略。

> 本地化文案与游戏数据（`items`、商店、补给箱等 CSV）已经编进二进制，
> 如需替换，把同名文件放到**可执行文件旁边的 `builtin/` 目录**下即可覆盖，
> 容器内对应 `/app/builtin/`。

## 5. 管理后台与账号

1. 启动服务端后，浏览器访问 `http://<服务器IP>:1315`
2. 点击「连接」，填入 `ADMIN_TOKEN`
3. 在「账号管理」中创建账号，再把账号名和密码线下交给玩家

**公开注册页已移除，端口 1314 不再监听**，账号必须由管理员创建。

玩家数据是 `$DATA_DIR/users/<uuid>.json`，一个账号一份文档；举报记录在
`$DATA_DIR/reports/`。密码使用 PBKDF2-HMAC-SHA512 存储，旧的无盐 MD5
密码在首次登录成功时自动升级。

## 6. 原项目

README 与更多细节见原始项目：[LoveBeforT/CSO2-Server](https://github.com/LoveBeforT/CSO2-Server)
