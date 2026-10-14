# 维护说明

面向本仓库的日常维护：代码在哪里、怎么验证、怎么发版。
上游参考实现的只读副本放在 `.ref/`（`CSO2-Server` 与 TypeScript 版
`cso2-master-server`），**不属于本项目源码**，也不要修改。

## 清理与现代化

- 移除无调用的旧注册页类型、手写快速排序包、随机数包装器、Gob 编解码、文件读取、
  控制台输入和旧 IP 辅助函数。
- 字节比较改用 `bytes.Equal`，排序改用 `slices.Sort` / `slices.SortFunc`，
  机器人数量比较改用内置 `max`。
- 物品随机抽取统一使用 `math/rand/v2`，移除每次调用时的全局重新播种、
  无效种子计数器和对应互斥锁。抽奖范围和权重不变。
- 121 处游戏数据结构初始化改为显式字段名，并补齐嵌套的 `ItemOption` 初始化，
  避免类型字段变化导致静默错位。
- IPv4 转换使用 `net/netip` 和 `encoding/binary`，显式拒绝原生 IPv6，继续支持
  IPv4 映射地址。
- 修复访问不存在服务器或频道时的空指针，管理页关闭无效房间现在返回 404。
- 管理 API 的 JSON 解码拒绝第二个 JSON 值及尾随垃圾；收回物品接口拒绝 ID 0。
- 清理 README 的停更告示、数据库与占位功能描述，更新后台说明。
  CI 由 GitHub Actions 承担（构建并推送镜像），不再依赖 AppVeyor。
- 全树死代码清扫：删除约 150 行注释掉的代码、约 20 个零引用函数/类型
  （含 137 行的 `GetFlags`）、约 60 个零引用的协议与枚举常量、
  若干只写不读的结构体字段，以及 `UnlockList` / `item_unlock.csv` 这条
  从未被消费的数据通路。判据与逐项清单见 [FIXES.md](./FIXES.md) §11.1。
  删除后由 `go build` 兜底 —— 删到仍在使用的符号会直接报 `undefined`。
- 全仓库行尾统一为 LF（此前 6 个 locale ini 与 4 个 CSV 是 CRLF），
  并加 `.gitattributes`（`* text=auto eol=lf`）固定。解析结果经摘要比对未变。
- 模块路径改为 `cso2-server` —— 裸 `go build` 的产物名取模块路径最后一段，
  改这里默认产物名才真正变成 `cso2-server`；Dockerfile 同步为
  `/app/cso2-server`。版本号升至 `v0.9.0`。

## 模块职责

| 模块 | 职责 |
| --- | --- |
| `main.go` | 启动顺序、监听端口、后台服务、信号处理与关服存档 |
| `config/config.go` | 环境变量、内置 / `$DATA_DIR/server.yaml` 配置合并 |
| `config/csv.go` | 从内置资源装载武器、解锁、补给箱、商店等 CSV |
| `builtin/builtin.go` | `go:embed` 资源与可执行文件旁 `builtin/` 覆盖机制 |
| `logging/logging.go` | 单行 JSON 日志，stdout 输出 |
| `servermanager/storage.go` | 账号文档读写、UUID 命名、内存索引与原子写入 |
| `kerlong/hash.go` / `uuid.go` | PBKDF2 口令哈希（含旧 MD5 升级）、UUID v4 |
| `kerlong/ini.go` | 自建 INI 解析器（本地化文件用） |
| `blademaster/adminapi/server.go` | 管理服务启动、路由与认证 |
| `blademaster/adminapi/http.go` | JSON 编解码、错误映射与路径参数校验 |
| `blademaster/adminapi/handlers.go` | HTTP 请求与领域操作的连接 |
| `blademaster/adminapi/accounts.go` / `operations.go` | 账号和游戏管理操作 |
| `blademaster/adminapi/page.go` / `page.html` | 内嵌资源路由与语义化页面骨架 |
| `blademaster/adminapi/web/console.css` | 布局、主题、响应式与可访问样式 |
| `blademaster/adminapi/web/i18n.js` | 中英文文案、语言和主题偏好 |
| `blademaster/adminapi/web/api.js` | Bearer 认证、请求超时、会话取消与大整数传输 |
| `blademaster/adminapi/web/ui.js` | DOM 构造、表单工具、数值校验和增量字段比较 |
| `blademaster/adminapi/web/accounts.js` | 账号、物品及确认弹窗 |
| `blademaster/adminapi/web/app.js` | 页面导航、状态刷新、列表分页与操作记录 |
| `blademaster/adminapi/tests/console.test.mjs` | 前端回归测试（Node 内置测试运行器） |

保持已有游戏模块边界，未大规模移动协议处理文件。资源通过 `go:embed` 全部进入
可执行文件；部署仍不需要 npm、Node、CDN 或额外 Web 目录。Node
仅用于开发时前端测试。

## 后台体验

五个页面：概览、在线玩家、账号管理、运维工具、操作记录。

- 桌面侧栏导航、运行指标、实例信息和快捷操作；手机顶部导航与账号卡片。
- 中文 / English 以及亮色 / 暗色 / 跟随系统，偏好保存于 `localStorage`，
  不改变游戏的 `LOCALE`。
- 账号搜索和每页 20 项；新增、查看、修改、删除；离线账号同样支持物品管理。
- 显示账号战绩与仓库数量；玩家列表支持踢出、物品操作和 GM / VIP 设置。
- 删除、踢出、收回物品和关闭房间有确认弹窗。表单提交期间禁用重复操作。
- 使用打开表单时的数据作为修改基准，仅发送用户改动字段，
  避免覆盖游戏期间变化的余额。
- 点券以十进制字符串保留在前端，按 JSON 数值发给 Go，覆盖完整 uint64 范围。
- 所有服务端返回内容使用 `textContent` 构造，昵称里的 HTML 不会作为代码运行。
- 请求超时为 15 秒，退出会取消在途请求。401 清除会话并重新要求连接。
- 令牌只保存在当前标签页的 `sessionStorage`；退出即删除。页面内存操作记录最多
  100 条，刷新清空。
- 焦点轮廓、跳转链接、原生 dialog、表单标签、状态反馈和减少动态效果偏好。
- 同源资源和 CSP，不使用内联事件处理、内联脚本或外部字体。

## 验证

在项目根目录执行：

```bash
go build ./...
go test ./...                       # 74 个测试函数
go test -race ./...
go vet ./...                        # 无输出
git ls-files -z '*.go' | xargs -0 gofmt -l   # 无输出
node --test blademaster/adminapi/tests/console.test.mjs   # 7 个用例
```

> 不要直接 `gofmt -l .` —— `.ref/` 里的上游副本会被一起列出来；这里只检查
> 受版本控制的 Go 文件。`-z` / `xargs -0` 只是让文件名里的空格不至于把路径拆开。
> 若工作区有尚未提交的删除，`gofmt` 会报 `no such file or directory`，
属正常现象。

打包与容器：

```bash
mkdir -p .dist
go build -trimpath -ldflags="-s -w" -o .dist/cso2-server .
go build -tags timetzdata -ldflags '-s -w' -trimpath -o .dist/cso2-server .   # Dockerfile 的构建方式
docker build -t cso2:temp .
```

Go 测试覆盖管理认证、账号生命周期、存储与配置，以及内嵌资产路由、
严格 JSON 请求体、物品 ID 和不存在频道的处理。
管理路由测试直接执行完整 HTTP Handler，不依赖临时监听端口。

前端回归覆盖增量修改、零值和布尔值、整数边界、uint64 点券、
令牌失效、退出后的迟到响应、网络错误及翻译完整性。

浏览器验证使用隔离的 `$DATA_DIR` 和测试端口，
不读取现有玩家档案：连接与刷新、特殊字符昵称、账号增量保存、
桌面与 390px 手机布局、语言和主题切换。真实游戏客户端联机行为仍需用客户端验证。

## 启动

```bash
export ADMIN_TOKEN=ChangeMe_0123456789
export DATA_DIR=./data
go run .          # 或用 .dist/cso2-server
```

默认访问 `http://localhost:1315/`，点击「连接」填入令牌。
如果不设置 `DATA_DIR`，默认是 `/data`（容器里的挂载点）；
本地跑记得改成本地路径。 容器方式见 [README.md](./README.md)。
