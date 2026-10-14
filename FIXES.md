# 修复说明

本项目是 `CSO2-Server`（KouKouChan，Go，v0.8.0）的维护分支，
上游原始实现与 TypeScript 参考项目保存在 `.ref/` 下供对照。
线协议、游戏数据与客户端行为均未改动，客户端无需任何调整。

三件事被彻底简化：

- **存储**：没有数据库、没有 Redis、没有 BloomFilter。每个账号是一份以 UUID 命名的
  JSON 文档，名字到 UUID 的映射放在内存里，位置由 `DATA_DIR` 决定。
- **日志**：不再落盘，全部是 stdout 上的 JSON 行。
- **部署**：配置、本地化、游戏数据全部编进二进制，**发布物就是单个可执行文件**。

---

## 1. 构建

```shell
go build              # 依赖 x/text 与 YAML v3，不需要 cgo
go test -race ./...
git ls-files -z '*.go' | xargs -0 gofmt -l   # 应当无输出
```

需要 **Go 1.27 或更高版本**。

### 构建到 `.dist`

```shell
mkdir -p .dist && go build -trimpath -ldflags="-s -w" -o .dist/CSO2-Server .
```

Windows PowerShell：

```powershell
New-Item -ItemType Directory -Force .dist | Out-Null
go build -trimpath -ldflags="-s -w" -o .dist/CSO2-Server.exe .
```

**就这些。** 资源已经编进二进制，`.dist` 里只有那一个文件（约 11 MB），
直接就能跑，不需要在旁边放任何目录。

---

## 2. 配置与内置资源

设置按**性质**分成两类，这是本节的核心：

| 类别 | 放哪 | 例子 |
| --- | --- | --- |
| **部署**：这台机器/这个容器怎么跑 | **环境变量** | 端口、语言、数据目录、管理开关 |
| **玩法**：服务器怎么表现 | **`$DATA_DIR/server.yaml`** | 人数上限、商店开关、日志级别 |
| **只读资源**：程序自带的数据 | **二进制内置**（可被 `<exe>/builtin/` 覆盖） | 本地化 ini、游戏数据 CSV |

理由：同一份 `server.yaml` 在任何机器上都应该产生同样的服务器行为，
而"这个实例监听哪个端口"是部署决定的；容器更应该靠改环境变量来重配，
而不是往镜像里塞文件。反过来，`server.yaml` 放在 `DATA_DIR` 里就能跟玩家数据
一起进 volume，改配置不需要重建镜像、也不需要重新挂载。

### 2.1 配置：环境变量 + 可选 YAML

**没有任何必需的配置文件**，开箱即用。

环境变量（`config.Environment`）：

| 变量 | 默认 | 说明 |
| --- | --- | --- |
| `DATA_DIR` | `/data` | 玩家数据 + 可选 `server.yaml`（容器里的挂载点；本地跑请显式设置） |
| `TCP_PORT` | 30001 | 游戏端口 |
| `UDP_PORT` | 30002 | UDP 打洞端口 |
| `ADMIN_PORT` | 1315 | 管理后台端口 |
| `ENABLE_ADMIN` | true | 是否启动管理后台 |
| `LOCALE` | zh-cn | 形如 `zh-cn`，不带 `.ini` |
| `CODE_PAGE` | 由 `LOCALE` 推导 | 客户端字符集，见 2.5 |
| `ADMIN_TOKEN` | 无 | 管理令牌，缺失则不启动管理后台 |

`$DATA_DIR/server.yaml`（可选，**增量覆盖**内置默认值）：

```yaml
server:
  max_users: 0            # 同时在线上限，0 = 不限
  enable_shop: true
  unlock_all_weapons: true

debug:
  level: 2                # 只影响 DebugInfo 的数量
```

配置从 INI 换成 YAML 的收获：

- **注释和嵌套**，不用再靠 `[Section]` 和 `key=value` 模拟结构。
- **未知键直接报错并退出**（`decoder.KnownFields(true)`）。
  原来的 INI 解析器会把拼错的键静默忽略 —— 那正是"我明明改了却没生效"的
  经典来源。现在会打印带行号的错误：

  ```
  cannot parse /data/server.yaml: yaml: unmarshal errors:
    line 2: field enable_shopp not found in type config.serverSection
  ```

- **增量覆盖**：先解析内置默认值，再把运营的文件叠加在上面，所以只写要改的键即可。
  内置与运营文件都为空/缺键时保留默认值。

> 依赖：`go.yaml.in/yaml/v3`（`gopkg.in/yaml.v3` 的新家）。这是本项目
> **唯一**新增的第三方依赖，通过本地 module cache 离线解析验证。

### 2.2 内置资源

```
builtin/
├── builtin.go              //go:embed server.yaml locales scripts
├── server.yaml             默认配置（同时也是运营文件的模板）
├── locales/                本地化文件 + motd.txt
│   ├── global.ini  zh-cn.ini  zh-tw.ini  en-us.ini  ko-kr.ini  tr-tur.ini
│   └── motd.txt
└── scripts/                从客户端解包出来的游戏数据
    ├── item_list.csv  item_unlock.csv  supplyList.csv  shop.csv
    └── exp_level.csv  vip_info.csv  defaultItemList.csv
```

原来的目录名是 `assert/`（"assert" 是断言，和资源毫无关系，纯属命名错误）
嵌套 `cstrike/scripts/`（照抄客户端 mod 目录）。现在统一成 `builtin/scripts/`。

`builtin.go` 用 `//go:embed` 把整棵树编进二进制，对外只有两个入口：

```go
builtin.ReadFile("locales/zh-cn.ini")      // 读内置资源（磁盘覆盖优先）
builtin.Exists("scripts/item_list.csv")    // 判断是否存在
```

`config` 包不再拼接文件路径：配置、本地化、MOTD、CSV 都从 `builtin` 取字节再解析。
INI 解析器为此保留了 `Parse([]byte)`（本地化文件仍是 INI），`LoadIni(path)` 只在测试里用。

### 2.3 覆盖只读资源（不用重新编译）

本地化和游戏数据走一条规则：

> 可执行文件同级的 `builtin/` 目录里，**相对路径相同**的文件会盖掉内置副本。

```
<exe目录>/builtin/locales/zh-cn.ini        只换简体中文文案
<exe目录>/builtin/locales/ja-jp.ini        新增语种（配 LOCALE=ja-jp）
<exe目录>/builtin/scripts/item_list.csv    改武器数据
```

**没有放的文件继续用二进制里的那一份**，所以可以只覆盖一项。
启动日志会写明哪些文件走了覆盖（`Using override …`）。
如果覆盖文件存在但读不出来（权限问题），会打 `warn` 而不是静默忽略 ——
否则"我明明改了却没生效"会非常难查。

Docker 下对应 `/app/builtin/`（镜像里可执行文件是 `/app/app.bin`）：

```shell
docker run ... -v /host/zh-cn.ini:/app/builtin/locales/zh-cn.ini:ro
```

> 注意配置项**不**走这条规则：它固定读 `$DATA_DIR/server.yaml`（见 2.1）。
> 两者分开是有意的 —— 配置是运营状态，应该待在 volume 里跟数据一起备份；
> 只读资源是程序的一部分，跟二进制放一起。

### 2.4 运行期需要的东西

| 内容 | 来源 |
| --- | --- |
| 默认配置、本地化、MOTD、游戏数据、管理页 | **二进制内置** |
| 配置覆盖 | `$DATA_DIR/server.yaml`（可选） |
| 资源覆盖 | `<exe目录>/builtin/`（可选） |
| 账号文档、举报记录 | `$DATA_DIR`（默认 `/data`） |

所以运行一个服务器只需要：**一个 exe + `ADMIN_TOKEN` 环境变量**。

### 2.5 `LOCALE` 同时决定字符集（`CodePage` 的去留）

原配置里 `LocaleFile=zh-cn.ini` 和 `CodePage=gbk` 是两个独立的键，
但它们其实描述同一件事：**客户端说的是哪种语言/编码**。

`CodePage` 绝不是废键 —— 它决定服务端发出的玩家名、战队名、签名、频道名
用哪种字节编码（`typestruct/userInfo.go`、`servermanager.go`），
也决定解析客户端登录名的编码（`core/user/login.go`）。填错就是满屏乱码。

但让运营填两遍确实是设计问题，所以现在**从 `LOCALE` 推导**：

| `LOCALE` | 推导出的字符集 |
| --- | --- |
| `zh-cn` / `zh-hans` | gbk |
| `zh-tw` / `zh-hk` / `zh-hant` | big5 |
| `ko-kr` | euc-kr |
| `ja-jp` | shift_jis |
| 其余（`en-us`、`tr-tur`、`global`…） | utf-8 |

遇到不寻常的组合（比如韩文客户端配英文界面）仍然可以用 `CODE_PAGE` 显式覆盖。
不认识的 `CODE_PAGE` 依旧**启动即 panic 并列出可用值**，而不是悄悄按 utf-8 发字节。

---

## 3. 数据存储

### 3.1 位置：`DATA_DIR`

所有持久化数据都在 `DATA_DIR` 下：

```shell
DATA_DIR=/srv/cso2/data ./CSO2-Server
```

未设置时默认为 **`/data`**（容器里的挂载点）。相对路径按当前工作目录解析
（`InitDataDir` 会用 `filepath.Abs` 转成绝对路径）。
解析后的绝对路径会在启动时写进日志（`Persistent data directory is ...`），不需要猜。

```
$DATA_DIR/
├── users/
│   ├── b1437bd3-6fa1-4f15-ae45-7bfb55a1b40e.json    # 账号文档
│   ├── 0f3c1e88-…-9a2b.json
│   └── …
└── reports/
    └── <被举报者名>.txt                              # 举报记录，追加写入
```

程序资源（配置、本地化、游戏数据）已经编进二进制，见第 2 节，
所以它们不受 `DATA_DIR` 影响，也不需要放在磁盘上。

### 3.2 为什么用 UUID 而不是用户名做文件名

账号名可以被**改名**，而且可能含有在路径里有歧义甚至危险的字符。
所以每个账号在创建时拿到一个随机的 UUID v4，它才是这份数据的永久身份：

- 文件名 = `<uuid>.json`
- 文档里的 `Uuid` 字段与文件名一致，可自校验
- `User.Userid`（1..MAXUSERNUM 的运行时槽位号，每次登录重新分配）
  **与 `Uuid` 无关**，且不落盘

副作用是账号名/昵称从此不再受文件名规则约束 —— 不需要再担心目录穿越，
因为名字根本不参与路径拼接。文件系统层面只剩下 UUID 格式校验。

### 3.3 账号查找：内存索引

启动时扫描 `$DATA_DIR/users/*.json`，构建三张表：

| 表 | 用途 |
| --- | --- |
| `username → uuid` | 登录、GM 按账号名操作 |
| `nickname → uuid` | 昵称占用判断 |
| `uuid → {username, nickname}` | 改名时回收旧键 |

之后所有查找走内存，不再碰文件系统。读不到或不合法的文件会被跳过并打 `warn`，
不影响启动；重复的账号名/昵称也会告警。

`RebuildUserIndex()` 可以随时重建索引，测试里就是靠它验证「索引能从磁盘完全恢复」。

### 3.4 存储格式

一个账号 = 一个 JSON 文档，即 `User` 结构体的序列化结果。持久化字段约 50 个：
`Uuid`、账号信息、等级与货币、战绩、称号与签名、已解锁的头像/称号/成就位图、
技能经验，以及嵌套的 `Inventory`（物品列表、8 个装扮槽、购买菜单、3 套配装）
和 `WeaponKills`。

`json:"-"` 的运行时字段（`Userid`、连接、当前频道/房间/队伍、局内计数、网络信息）
不落盘。

### 3.5 写入策略

| 时机 | 说明 |
| --- | --- |
| 每 60 秒 | 全量保存在线玩家 |
| 玩家断线 | 单人保存 |
| 第一次 Ctrl+C | 关服前全量保存 |
| GM `save` 命令 | 同上 |
| 密码哈希升级、GM 操作离线玩家 | 立即保存 |

每次都是**整条文档覆写**，但通过「写临时文件 + rename」完成，
因此进程写盘中途崩溃不会留下半截 JSON —— 一账号一文件的前提下，
半截文件等于该账号报废，所以这一步是必要的。

### 3.6 账号名唯一性现在是原子的

名字在索引里**先抢占、再写文件**，写失败会回滚。因此两个并发创建同名账号的请求
不可能都成功（这一点此前是「检查然后写入」，存在窗口）。
管理 API 会把 `ErrUsernameTaken` / `ErrNicknameTaken` 映射成 `409` 和明确的文案。

### 3.7 密码兼容性

旧的无盐 `md5(username+password)` 可以正常登录，并在首次成功登录时自动改写为
PBKDF2 格式。

---

## 4. 日志

日志**不再写文件**，全部以单行 JSON 输出到 stdout，一条记录一行：

```json
{"ts":"2026-10-13T15:23:23.722Z","level":"info","msg":"Server is running at [AnyAdapter]:30001","caller":"main.go:292"}
```

| 字段 | 说明 |
| --- | --- |
| `ts` | UTC，毫秒精度，RFC3339 |
| `level` | `debug` / `info` / `warn` / `error` |
| `msg` | 消息正文，多个操作数以空格连接（与原 `log.Println` 一致） |
| `caller` | 真实调用点 `文件:行号`，便于在大量点导入的代码里定位来源 |

日志包原来的名字是 `verbose`，但它其实承担全部日志职责（info/warn/error 都在里面），
`verbose` 只描述了 `DebugInfo` 那一部分，属于名不副实。现在叫 **`logging`**，
文件也从 `verbose.go` 改成 `logging.go`。因为它到处被点导入，调用点一行未改。

`server.yaml` 的 `debug.level` 控制**调试输出量**：

| 值 | 效果 |
| --- | --- |
| `0` | 不输出 `DebugInfo` |
| `1` | 额外输出 `DebugInfo(1, ...)`（41 处） |
| `2` | 全开，含 `DebugInfo(2, ...)`（318 处） |

**Info / Warn / Error 任何级别都输出**，不受这个值影响 —— 启动横幅（监听端口、
数据目录、生效的配置来源）必须始终可见，否则排查"服务器到底起没起来"会很难。
原来 424 处 `DebugInfo(...)` 调用一行未改。

整个记录先渲染进缓冲区再**一次 Write 写出**，所以多 goroutine 并发时不会串行。
需要保留日志就重定向 stdout：`./CSO2-Server >> server.jsonl 2>&1`。

日志不再有落盘选项（原 `LogFile` 键已删除）。
管理页面的输出走 HTTP 响应，不走 stdout；服务器进程的 stdout 现在**只有** JSON 日志。

---

## 5. 管理后台：Web API + 页面

旧版是一个自定义 TCP 文本协议（签名 `0x66`），配套一个 `-console` 客户端模式，
默认口令 `cso2server123`、明文传输。**整套已被 Web API 取代并移除**：
`blademaster/GMconsole/`（服务端 + 客户端 + 协议）、`local-console.bat`、
以及旧的 `[Console]` 配置段（`EnableConsole`、`GMport`、`GMusername`/`GMpassword`）都没有了。

现在：`blademaster/adminapi/`，一个监听环境变量 `ADMIN_PORT`（默认 1315）的 HTTP 服务，
提供一组 JSON 接口和一个自带的管理页面。

### 5.1 认证：`ADMIN_TOKEN`

- token **只从环境变量读取**，不在配置文件里，也不落盘。
- 格式：**16–128 位，仅 `[A-Za-z0-9_]`**（字母、数字、下划线）。
- 请求用 `Authorization: Bearer <token>` 提交，比较用 `subtle.ConstantTimeCompare`。
- **失败关闭**：`ADMIN_TOKEN` 缺失或格式不合法时，管理服务**根本不启动**，
  日志写明原因；游戏服不受影响。宁可没有管理接口，也不要一个谁都能调的。
  `ENABLE_ADMIN=false` 可以主动关掉它。
- `GET /`（页面本身）不鉴权 —— 页面里没有任何机密，机密是 API 返回的数据。

### 5.2 接口

`players` 是**当前在线**的人，`accounts` 是**账号记录**。

| 方法 | 路径 | 作用 |
| --- | --- | --- |
| GET | `/` | 管理页面 |
| GET | `/api/status` | 版本、在线人数、账号总数、运行时长、各端口、数据目录 |
| GET | `/api/players` | 在线玩家列表 |
| POST | `/api/players/{账号}/kick` | 踢出 |
| GET | `/api/accounts` | 全部账号（账号名 / 昵称 / UUID，来自内存索引，不读盘） |
| POST | `/api/accounts` | 新建账号 |
| GET | `/api/accounts/{账号}` | 账号详情（在线取内存对象，离线读文档） |
| PATCH | `/api/accounts/{账号}` | 增量修改账号 |
| DELETE | `/api/accounts/{账号}` | 删除账号 |
| POST | `/api/accounts/{账号}/items` | 发放物品 `{"itemId":5042,"count":2}` |
| DELETE | `/api/accounts/{账号}/items/{物品id}` | 收回物品 |
| POST | `/api/accounts/{账号}/vip` | 设为 VIP |
| POST | `/api/accounts/{账号}/gm` | 设为 GM |
| DELETE | `/api/rooms/{服务器序号}/{频道序号}/{房间id}` | 强制关闭房间 |
| POST | `/api/save` | 立即存档全部在线账号 |

路由用 Go 1.22+ `http.ServeMux` 的 `"METHOD /path/{wildcard}"` 模式，
没有引入任何路由库。请求体限长 64 KB 且拒绝未知字段。

### 5.3 账号 CRUD

**这是账号的唯一来源** —— 公开注册页已删除（见 5.5）。

- **新建**：账号名、昵称、密码必填，邮箱/GM/VIP 可选。账号名与昵称各不超过 64 字符，
  且都必须唯一（由 3.6 的原子占位保证）。
- **修改**：`PATCH` 的每个字段都是**指针**，因此「不改」和「改成 0」是可区分的；
  只提交改动过的字段，未改字段不会被覆盖。改密码会重新走 PBKDF2。
  VIP 等级是直接赋值的整数（`SetVIP()` 那个便利接口则是按等级推导，
  和游戏内逻辑一致 —— 两者都保留了）。
- **删除**：如果玩家在线会先踢下线，否则 60 秒的周期存档会把文件写回来。

**在线 vs 离线**：玩家在线时改的是内存里的活对象（否则会被下次存档覆盖），
并立刻下发 `UserInfo` 包让客户端刷新；不在线时读文档、改、原子写回。
两种情况都会**立即落盘**，管理员的操作不会因为崩溃而丢。

物品发放 / 收回、改 VIP / GM 对离线账号同样有效。

### 5.4 页面

管理页以 `page.html` 为入口，`web/` 中分离样式、翻译、API 请求、账号表单和页面交互。
所有资源由 `//go:embed` 编进二进制，无 CDN、npm 或前端打包依赖，内网可用。
支持中文 / English、亮色 / 暗色 / 跟随系统；桌面使用侧栏，手机使用顶部导航和账号卡片。

令牌保存在当前标签页的 `sessionStorage`，退出时删除；语言与外观偏好保存在
`localStorage`。页面记录仅在内存中保留，刷新页面会清空。HTTP API 认证方式保持不变。

功能：状态卡片、搜索和分页、账号创建 / 增量编辑 / 删除、在线玩家管理、
物品发放 / 收回、VIP / GM 设置、强制关闭房间、立即存档和会话操作记录。
详情见 [MAINTENANCE.md](./MAINTENANCE.md)。

### 5.5 被移除的公开注册页

`web/register/`（HTTP 服务、`html/template` 渲染、验证码、SMTP 发信）、
`assert/web/`（7.8 MB 前端资源，97 个文件；`assert` 这个名字本身就是错的，见 2.1）、`kerlong/mail.go`（`SendEmailTO`），
以及 `[Register]` 配置段和 `CSO2Conf` / `CSO2Locales` 里对应的
13 个字段全部删除。端口 **1314 不再监听**。

---

## 6. 安全修复

### 6.1 密码存储（严重）

原实现 `md5(username + password)`：无盐、可被彩虹表秒破、同密码用户哈希相同，
比较还是逐字节循环（非常量时间）。

现在：**PBKDF2-HMAC-SHA512，10 万次迭代，16 字节随机盐，64 字节派生密钥，
常量时间比较**（`kerlong/hash.go`）。

```
新格式：2:<salt hex>:<iterations>:<key hex>
旧格式：32 位十六进制（md5(username + password)）
```

### 6.2 SQL 注入（随数据库一起消失）

原先 `servermanager/database.go` 有三处把客户端输入拼进 SQL，且 DSN 开着
`multiStatements=true`。查询先改为参数化，随后整个 SQL 层被删除，攻击面不复存在。

### 6.3 路径穿越（随 UUID 化一起消失）

旧实现把账号名直接当文件名，账号名又来自登录包 —— 本身就是目录穿越面。
现在文件名是服务端生成的 UUID，名字不参与路径拼接；存储层只校验 UUID 格式。

### 6.4 GM 控制台并发崩溃（严重）

`clients map[string]bool` 被每条控制台连接读写却没有任何锁。
Go 的并发 map 写入是 `recover()` 抓不到的 `fatal error`，会直接结束进程。

现在用 `sync.RWMutex` + `isGMLoggedIn`/`setGMLoggedIn`/`clearGMLogin` 访问。
口令比较改为 `subtle.ConstantTimeCompare`；空命令不再 `req[0]` 越界 panic。

> 更正：先前分析称「GM 控制台在登录前可执行任意命令」是**错的**，
> 每个 handler 开头本来就有登录态检查。

### 6.5 注册站空指针 panic

（该代码已随注册页一起删除，此处仅作记录。）
`web/register/register.go` 在检查 `err` **之前**就 `defer f.Close()`，
任何缺失的静态资源都会触发空指针 panic。删除前已改为先判错返回。

---

## 7. 并发与正确性修复

- `UManager.Lock`：`sync.Mutex` → `sync.RWMutex`；`AddUser`/`DelUser` 原来
  **先检查 map 再上锁**（TOCTOU），改为全部在锁内完成。
- 删除全局 `usersManagerLock`：它被复制进 `UsersManager` 结构体，
  `go vet` 报 `literal copies lock value` —— 即结构体内的锁与全局变量是**两把不同的锁**，
  互斥完全失效。
- 新增 `SnapshotUsers()`/`OnlineCount()`/`DelUserByID()`，
  把 **12 处**无锁遍历 `UsersManager.Users` 改为遍历快照
  （原写法会触发 `fatal error: concurrent map iteration and map write`）。
- `DelUserWithConn`、`DelRoomFromManager` 的「锁外检查、锁内删除」已修正；
  持久化 I/O 移到锁外，不再阻塞其他 goroutine。
- 存储索引的所有读写都在 `RWMutex` 下进行。
- `main.go` 信号通道改为带缓冲并补 `signal.Stop`。
- `config/config.go` 两处格式串拼接改为格式参数。
- **`GetNewUser()` 的默认值现在有回归测试锁定**（`user_defaults_test.go`）。
  它用的是**无键字面量**，往 `User` 里插字段会静默错位 —— 加 `Uuid` 时就踩到了，
  测试把这类错误变成了明确失败。

---

## 8. 依赖移除

默认构建**只剩一个**第三方依赖：

| 原依赖 | 替换为 | 理由 |
| --- | --- | --- |
| `djimenez/iconv-go` | `golang.org/x/text/encoding` | 需要 cgo 与系统 libiconv |
| `garyburd/redigo` | **删除**（Redis 支持整体移除） | 上游已归档；内网小规模不需要缓存层 |
| `go-sql-driver/mysql` | **删除**（MySQL 支持整体移除） | 同上 |
| `willf/bitset` | **删除**（BloomFilter 整体移除） | 查重改用内存索引 |
| `gomail.v2` + `quotedprintable.v3` | **删除**（随注册页的邮件验证一起移除；不引入替代品） | 唯一的用途是注册验证码，功能已删 |
| `gopkg.in/ini.v1` | `kerlong/ini.go` 自建解析器 | 只用到 `[Section]` / `key=value` / `#` |
| `smartystreets/goconvey` | — | 未使用的间接测试依赖 |

UUID 生成也用标准库 `crypto/rand` 手写（`kerlong/uuid.go`），没有引入新依赖。

整棵源码树已规范为 **LF 行尾**并通过 `gofmt`，
`git ls-files -z '*.go' | xargs -0 gofmt -l` 无输出。
不要直接 `gofmt -l .` —— `.ref/` 里的上游副本会被一起列出来。

---

## 9. 行为变化（升级需注意）

1. **需要 Go 1.27+**（原为 1.15）。
2. **MySQL / Redis / BloomFilter 相关代码与配置键全部移除。**
   旧的 `[Database]` 配置段（`EnableDataBase`、`DBUserName`、`DBpassword`、
   `DBaddress`、`DBport`、`RedisIP`、`RedisPort`）和 `[Debug]` 的 `LogFile` 都已删除；
   配置换成 YAML 后这些键自然不存在了（旧 INI 文件不会再被读取）。
3. **数据目录与命名都变了**，现在是 `$DATA_DIR/users/<uuid>.json`。
   项目此前没有正式发布过 JSON 布局（MySQL 是唯一真实后端，且已确认属于项目残留、
   不需要保留），因此没有提供迁移工具。
4. **日志不再落盘**，`CSO2-Server.log` 不会再生成。
5. **管理接口从 TCP 控制台换成了 Web API。**
   旧的 `[Console]` 配置段（`EnableConsole`、`GMport`、`GMusername`、
   `GMpassword`）已删除，端口改用环境变量 `ADMIN_PORT`（默认仍是 1315），
   但现在跑的是 HTTP。**必须在环境变量里提供 `ADMIN_TOKEN`**，
   否则管理服务不会启动。旧的 `-console` 命令行模式和 `local-console.bat` 已删除。
6. **公开注册页整个删除，端口 1314 不再监听。**
   旧的 `[Register]` 配置段（`EnableRegister`、`EnableMail`、`REGPort`、
   `REGEmail`、`REGPassWord`、`REGSMTPaddr`）已移除，`assert/web/` 的
   7.8 MB 前端资源也随之删除。**账号必须由管理员在管理页创建**，
   再线下把账号名和密码交给玩家。
7. **关服改成一次信号。** 以前是「按一次 Ctrl+C 存档，再按一次才退出」，
   现在按一次（SIGINT 或 SIGTERM）就会先存档再退出。同时**新增了 SIGTERM**
   —— 之前只监听 SIGINT，`docker stop` / systemd 发的 SIGTERM 会被忽略，
   进程在超时后被强杀，那段时间的存档就丢了。
8. **配置从 `server.conf`（INI）换成 `$DATA_DIR/server.yaml`（YAML），
   而且配置文件变成可选的。** 旧键的去向：

   | 旧键 | 现在 |
   | --- | --- |
   | `TCPPort` | 环境变量 `TCP_PORT` |
   | `UDPPort` | 环境变量 `UDP_PORT` |
   | `AdminPort` | 环境变量 `ADMIN_PORT` |
   | `EnableAdmin` | 环境变量 `ENABLE_ADMIN` |
   | `LocaleFile` | 环境变量 `LOCALE`（取值形如 `zh-cn`，不带 `.ini`） |
   | `CodePage` | 由 `LOCALE` 推导，`CODE_PAGE` 可覆盖 |
   | `MaxUsers` | `server.yaml` 的 `server.max_users` |
   | `EnableShop` | `server.yaml` 的 `server.enable_shop`（`true`/`false`） |
   | `UnlockAllWeapons` | `server.yaml` 的 `server.unlock_all_weapons` |
   | `DebugLevel` | `server.yaml` 的 `debug.level` |

   旧的 `server.conf` **不会再被读取**（既不解析也不报错），所以升级后
   必须按上表迁移，否则服务器会按默认值启动（默认端口恰好相同，
   但人数上限、商店开关、语言都会回到默认）。
9. **启动时的致命错误现在是非零退出码。** 之前 `main` 里 recover 掉 panic 后
   进程以 `0` 退出，systemd / Docker / Kubernetes 会把它当成正常结束，
   既不重启也不告警。现在是 `1`。
10. **`CodePage` 取值仍然只接受**
    `gbk`/`gb18030`/`big5`/`euc-kr`/`shift_jis`/`utf-8` 及若干别名（大小写不敏感）。
    填了不认识的值会在**启动时 panic 并列出可用值**，而不是静默按错误码页处理
    （那会让所有玩家名和聊天变乱码且极难排查）。
11. 非法字节现在替换为 U+FFFD（`x/text` 行为），iconv 原来返回错误。对正常客户端无影响。
12. 管理页的账号名/昵称不再受文件名规则限制（因为不再是文件名），
    但长度上限 64 与唯一性仍然强制。
13. **发布物从「exe + `CSO2-Server/` 资源目录」变成一个 exe。**
    配置、本地化、游戏数据、管理页全部编进二进制；`assert/`、`locales/`
    这些磁盘目录不再需要，也不再随发布包分发。
14. **两个包改名**：`configure` → `config`（包名原本是动词），
    `verbose` → `logging`（它承担全部日志职责，不只是 verbose）。
    文件名同步为 `config/config.go`、`logging/logging.go`。
15. **`assert/` 目录消失**（名字本身就是错的），内容进了 `builtin/scripts/`。

---

## 10. 验证

| 项目 | 结果 |
| --- | --- |
| `go build ./...` | 通过 |
| `go test -count=1 ./...` | 通过（**74** 个测试函数，含子测试 **133** 个用例） |
| `go test -race -count=1 ./...` | 通过 |
| `go test -race -count=5 ./kerlong/ ./config/` | 通过（排除偶发失败） |
| `go vet ./...` | **无输出**（原先的点导入风格提示已随显式字段名改造清除） |
| `git ls-files -z '*.go' \| xargs -0 gofmt -l` | 无输出 |

测试覆盖：

- `config/config_test.go` —— **12 个内置资源全部存在且非空**（并断言已退休的
  `server.conf` 确实不在二进制里）、内置 `server.yaml` 的默认值、
  **`$DATA_DIR/server.yaml` 覆盖生效**、**部分覆盖保留其余内置默认值**、
  **拼错的键会终止启动**、内置本地化加载、**全部 6 个内置 locale 逐个解析**、
  内置 MOTD、**内置 CSV 装载出 1132 件武器 / 84 条解锁 / 10 个箱子 /
  107 件默认物品 / 103 条商店记录**、locale 磁盘覆盖优先于内置且不影响其他文件；
  环境变量部分：默认值、全部覆盖、**非法端口回退默认**、
  **非法布尔回退默认**（含 5 种真值 / 5 种假值写法）、
  **`LOCALE`→字符集推导**（11 种输入）、`CODE_PAGE` 显式覆盖、
  `LOCALE` 文件名规整（大小写/空白/带 `.ini`/空值）。
- `blademaster/adminapi/console_test.go` —— 6 个内嵌前端资产的 MIME 与内容、
  页面 CSP、缺失资产 404、请求体尾随 JSON 与物品 ID 0 的拒绝、
  不存在频道的房间关闭返回 404。
- `blademaster/adminapi/admin_test.go` —— **token 格式校验（含 10 种非法输入）**、
  **14 条 API 路由**在无 token / 错误 token 下全部 401、Bearer 大小写不敏感、
  裸 token 不被接受、页面公开且不泄露 token、status/players 响应结构、
  **完整账号生命周期**（新建 → 列表 → 详情 → 增量修改 → 改名冲突 → 删除
  → 昵称被释放可复用）、新建/修改的字段校验（非 JSON / 未知字段 / 缺字段 /
  空白值 / 超长值 / 空 patch）、未知账号 404、踢人需在线 409、房间路径参数校验。
- `main_test.go` —— **一次信号即存档并退出**（先造出与磁盘不一致的内存状态，
  发一个信号，断言磁盘被刷新且函数返回），以及无人在线时也能及时返回。
- `kerlong/uuid_test.go` —— UUID v4 格式、1000 次无重复、16 种非法输入。
- `kerlong/hash_test.go` —— 新哈希往返、随机盐、错误密码拒绝、**旧 MD5 兼容与升级标记**、
  9 种畸形哈希输入。
- `kerlong/ini_test.go` —— 缺文件、BOM、`#`/`;` 注释、大小写不敏感、值内含 `=`、
  **内存解析 `Parse` 与文件解析 `LoadIni` 结果一致**。
- `blademaster/typestruct/user_defaults_test.go` —— **锁定 `GetNewUser()` 的全部默认值**
  （含三个位图长度与首字节），以及默认状态不跨用户共享。
- `servermanager/storage_test.go` —— 目录创建、UUID 命名、账号与昵称往返、原子覆写、
  文档确为可读 JSON 且不含运行时字段、**并发创建唯一性**、写失败回滚、
  **索引可从磁盘重建**、跳过损坏文件、改名回收旧昵称、输入校验、
  **UUID 校验与非 ASCII 名字**、密码校验、**旧 MD5 升级并落盘**。

端到端冒烟：**把 exe 单独放进一个空目录**（旁边什么都没有），跑四种场景

```
目录内容：CSO2-Server.exe          （8.1 MB，-s -w 之后）

A) 全部默认
   Environment: tcp=30001 udp=30002 admin=1315 enableAdmin=true locale=zh-cn codePage=gbk
   Reading configuration from built-in defaults
   Reading locale <zh-cn.ini> ...
   监听 30001 ✓ 1315 ✓
   GET /api/status → tcp=30001 udp=30002 admin=1315 shop=true allWeapons=true

B) 全部用环境变量覆盖
   TCP_PORT=31001 UDP_PORT=31002 ADMIN_PORT=1316 ENABLE_ADMIN=false LOCALE=en-us
   Environment: tcp=31001 udp=31002 admin=1316 enableAdmin=false locale=en-us codePage=utf-8
   Admin web API is disabled (ENABLE_ADMIN=false)
   监听 31001 ✓，30001 ✗，1316 ✗，1315 ✗

C) $DATA_DIR/server.yaml 增量覆盖（只改 max_users / enable_shop / debug.level）
   Reading configuration from /…/data/server.yaml
   GET /api/status → maxUsers=7 shop=false allWeapons=true
                                     ↑ 没写的键保留了内置默认值

D) 覆盖文件里写错一个键 → 进程以退出码 1 结束，并打印带行号的错误
   line 2: field enable_shopp not found in type config.serverSection
```

运行结束后目录里只有：exe、data/users/、data/reports/（以及可选的 data/server.yaml）。

账号生命周期（全部走 HTTP）：
  POST   /api/accounts            → alice / AliceInGame，uuid 88a930fb-…
                                    磁盘上是 88a930fb-….json，密码前缀 "2:"（PBKDF2 v2）
  GET    /api/accounts            → count=1 -> alice/AliceInGame
  PATCH  /api/accounts/alice      → nickname=Renamed level=42 points=777 gm=true
                                    未提交的 mail 保持 a@b.c；磁盘同步
  GET    /api/status              → version=v0.8.0 accounts=1
  DELETE /api/accounts/alice      → deleted；users/ 下 0 个文件
  GET    /api/accounts/alice      → 404

关服：单次信号 → 存档 → 退出（见 main_test.go，Windows 上无法用 taskkill 模拟 Ctrl+C）
```

管理后台冒烟（三种 token 场景）：

```
A) 不设 ADMIN_TOKEN     →「ADMIN_TOKEN is not set」，1315 不监听；30001 正常
B) ADMIN_TOKEN 只有 5 位 →「must be between 16 and 128 characters」，1315 不监听
C) 合法 token           → GET / 返回 14 KB HTML（公开）
                          /api/status 无 token → 401，错误 token → 401
                          正确 token → version=v0.8.0 adminPort=1315 dataDir=…
                          对离线账号发 2 个物品：itemCount 107 → 108 且已写入磁盘
                          设 VIP、立即存档均成功
                          向 1315 发旧的 0x66 控制台帧 → HTTP/1.1 400 Bad Request
```

---

## 11. 仍未处理（有意保留）

1. **反作弊为零** —— 击杀/助攻/死亡/胜负仍全部来自房主客户端上报，
   房主可以给自己开任意奖励（`blademaster/core/host/hostGameEnd.go`）。
2. **JSON 文档无法用 SQL 查询** —— 这是「内网小规模、用户可信」前提下的自觉取舍。
   代价是：排行榜、查「谁有这把枪」、批量发补偿都只能把 `$DATA_DIR/users/`
   整个读一遍在内存里解 JSON。
3. **索引在内存里，多进程部署会不一致** —— 单个服务器进程没问题，
   但同时跑两个进程指向同一个 `DATA_DIR` 时各自的内存索引不会同步。
4. **`blademaster/typestruct` 上帝包 + 全局点导入（`import . "..."`）** ——
   1346 行的 `packet.go` 与遍布全项目的点导入让符号来源不可追踪。
5. **死代码** —— `blademaster/core/version/version.go` 里 120 余行被注释掉的旧登录逻辑，
   同一段又被复制进 `login.go`。
6. **房间 ID 全局分配**（`RoomsManager`），无法按频道独立编号。
   房间查找本身走频道内的表，所以不存在数据损坏，只是可扩展性限制。
7. **管理接口是明文 HTTP，也没有失败锁定** —— token 本身强度足够（16–128 位随机串），
   传输层靠内网。如果要跨公网用，请放在反向代理后面加 TLS；也可以顺手加一层
   失败次数限制。默认仍监听 `0.0.0.0:1315`，建议防火墙限源。
8. **没有密码强度策略** —— 管理 API 只要求密码非空，不检查长度或复杂度。
   账号是管理员手工创建的，强度由管理员把关；如果需要可以加最小长度与
   弱口令校验。
9. **账号创建后没有「首次登录改密」流程** —— 管理员把账号名和密码线下交给玩家，
   玩家无法自行修改密码（没有自助入口）。若需要，可以加一个「玩家自助改密」
   的接口或用改名卡那样的一次性道具。
10. **两处注释与代码不一致**（不影响行为，改注释即可）：
    - `builtin/builtin.go` 的包注释与 `FS` 变量注释仍写 `server.conf`，
      实际内嵌的是 `server.yaml`。
    - `config.ReadEnv(exeDir)` 的 `exeDir` 参数未被使用，注释与内置
      `builtin/server.yaml` 里 "default: `<exe dir>/data`" 的说法都还停留在
      旧的默认值上；代码里 `DATA_DIR` 默认是 `/data`。
11. **两个内嵌 CSV 没有读取方**：`scripts/exp_level.csv` 与
    `scripts/vip_info.csv` 被打包并有常量声明，但没有任何代码读取它们
    （`LevelExp` 硬编码在 `blademaster/Exp/exp.go`）。保留是为了将来接回。
12. **本地化 ini 里残留 `[Register]` 段** —— 注册页删除后已无人引用，
    `[System]` 才是实际生效的部分。属于历史遗留，删掉不影响功能。
