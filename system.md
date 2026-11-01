# go_gateway_demo 系统功能说明

本项目是一个基于 gin 的微服务网关，由两个进程组成：

| 进程 | 启动方式 | 默认地址 | 作用 |
| --- | --- | --- | --- |
| 管理后台 dashboard | `go run main.go -config ./conf/dev/ -endpoint dashboard` | `127.0.0.1:8880` | 管理员登录，服务/租户的增删改查，流量统计 |
| 代理服务 server | `go run main.go -config ./conf/dev/ -endpoint server` | HTTP `127.0.0.1:8080`，HTTPS `127.0.0.1:4433` | 按服务规则把请求反向代理到下游 |

两个进程共用 MySQL（服务、租户配置）和 Redis（流量统计）。dashboard 写配置，server 定时热加载配置；server 写流量计数，dashboard 读取展示。

> `-config` 由 `golang_common/lib.InitModule` 定义，路径要用 `/` 分隔（反斜杠路径会在 `ParseConfPath` 中 panic）。

---

## 一、管理后台（dashboard）

### 1. 通用能力
- 请求链路日志：mysql / redis / request 日志带 trace
- 多语言参数校验错误提示（`TranslationMiddleware`）
- 统一响应格式 `{errno, errmsg, data, trace_id}`（`middleware/response.go`）
- panic 恢复、IP 白名单（`base.http.allow_ip`）
- 基于 cookie 的 session 登录态（`SessionAuthMiddleware`）
- swagger 文档：`/swagger/index.html`

### 2. 接口列表

**管理员 `/admin_login`**
| 方法 | 路径 | 说明 |
| --- | --- | --- |
| POST | `/login` | 管理员登录 |
| GET | `/logout` | 退出登录 |
| GET | `/login_info` | 当前管理员信息 |
| POST | `/change_pwd` | 修改密码 |

**服务管理 `/service`**
| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/service_list` | 服务列表（分页、关键字搜索） |
| GET | `/service_detail` | 服务详情 |
| POST | `/service_delete` | 删除服务（软删除） |
| GET | `/service_statistics` | 服务今日/昨日按小时的流量 |
| POST | `/service_add_http` / `/service_update_http` | 新增/修改 HTTP 服务 |
| POST | `/service_add_tcp` / `/service_update_tcp` | 新增/修改 TCP 服务 |
| POST | `/service_add_grpc` / `/service_update_grpc` | 新增/修改 GRPC 服务 |

**租户管理 `/app`**
| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/app_list` | 租户列表 |
| GET | `/app_detail` | 租户详情 |
| POST | `/app_add` / `/app_update` | 新增/修改租户 |
| GET | `/app_delete` | 删除租户 |
| GET | `/app_stat` | 租户今日/昨日按小时的流量 |

**大盘 `/dashboard`**
| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/panel_group_data` | 服务数、租户数、今日请求数等指标卡 |
| GET | `/flow_stat` | 全站今日/昨日按小时的流量 |

---

## 二、代理服务（server）

### 1. 启动流程（`main.go`）
1. `ServiceManagerHandler.LoadOnce()` 从 MySQL 加载所有 HTTP 服务（4 张表批量查询，无 http 规则的服务跳过）
2. `AppManagerHandler.LoadOnce()` 加载所有未删除租户
3. `proxy.service.reload_interval > 0` 时开启定时热加载（服务和租户）
4. 启动 HTTP 和 HTTPS 监听
5. 收到 SIGINT / SIGTERM / SIGQUIT 后优雅关闭

### 2. 不经过代理链的接口
| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/ping` | 健康检查 |
| POST | `/oauth/tokens` | 租户换取访问 token，见「租户认证」 |

### 3. 中间件链（按执行顺序，`http_proxy_router/route.go`）

> 下表中 2→5 的顺序有讲究：租户认证写入 `app` 后租户限流才能识别租户；租户限流排在流量统计之前，被租户限流拒绝的请求不计入租户当天请求量。

| # | 中间件 | 功能 | 错误码 |
| --- | --- | --- | --- |
| 1 | `HTTPAccessModeMiddleware` | 根据请求匹配服务 | 1001 未匹配到服务 |
| 2 | `HTTPProtocolCheckMiddleware` | 协议校验：https / websocket | 4001 服务不支持 https；4002 服务不支持 websocket |
| 3 | `HTTPJwtAuthTokenMiddleware` | 租户认证 | 2001 上下文无服务；2002 token 缺失/无效/租户不存在 |
| 4 | `HTTPJwtFlowLimitMiddleware` | 租户限流：qps、qpd | 5003 租户 QPS 限流；5004 租户日请求量限流 |
| 5 | `HTTPFlowCountMiddleware` | 流量统计：全站、服务、租户 | — |
| 6 | `HTTPFlowLimitMiddleware` | 限流：服务级、客户端 IP 级 | 5001 服务限流；5002 客户端 IP 限流 |
| 7 | `HTTPWhiteListMiddleware` | IP 白名单 | 3001 |
| 8 | `HTTPBlackListMiddleware` | IP 黑名单 | 3002 |
| 9 | `HTTPHeaderTransferMiddleware` | 请求头增/改/删 | — |
| 10 | `HTTPStripUriMiddleware` | 去掉服务前缀 | — |
| 11 | `HTTPUrlRewriteMiddleware` | URL 正则重写 | — |
| 12 | `HTTPReverseProxyMiddleware` | 负载均衡 + 反向代理 | 下游异常时返回错误响应 |

### 4. 服务匹配（接入方式）
- `rule_type=1` 域名接入：`Host`（去掉端口）与规则完全相等
- `rule_type=0` 前缀接入：URL 路径以规则开头
- 优先级：域名精确匹配 > 最长前缀匹配（例如 `/test_strip_uri` 不会被 `/test` 抢走）

### 5. 协议支持
- **HTTPS**：`[https]` 配置 addr、cert_file、key_file；addr 为空或证书文件缺失时跳过 HTTPS 启动。`need_https=0` 的服务拒绝 https 请求。
  - 本地自签证书生成：`go run "$(go env GOROOT)/src/crypto/tls/generate_cert.go" --host 127.0.0.1,localhost`，放到 `conf/dev/cert/`（已在 .gitignore 中）
- **WebSocket**：带 `Upgrade: websocket` 的请求仅在 `need_websocket=1` 时放行，由 ReverseProxy 透传长连接，不受 WriteTimeout 影响
- 下游协议默认 `http://`，与 `need_https` 无关；如需 https 下游，在 `ip_list` 中写明 `https://ip:port`

### 6. 租户认证（JWT）
1. 租户用 HTTP Basic（`app_id:secret`）调用 `POST /oauth/tokens`：
   ```
   curl -u app_id:secret -X POST http://127.0.0.1:8080/oauth/tokens
   ```
   返回 `{access_token, expires_in, token_type:"Bearer", scope:"read_write"}`。
   - 未带 Basic 认证 → 2001；app_id 不存在或 secret 错误统一返回 2002 `app_id or secret error`（防止探测租户），secret 常量时间比较
   - 签名失败（如未配置密钥）→ 2003
2. 访问服务时带 `Authorization: Bearer <token>`：
   - token 用 HS256 签名，`iss` = app_id，有效期 `proxy.jwt.expire_seconds`（默认 7200 秒）
   - 只接受 HMAC 签名算法，拒绝 `alg=none` 等；密钥为空时拒绝签发和校验
   - 校验通过后租户写入上下文 `app`，并删除 `Authorization` 头，不透传给下游
   - `open_auth=1` 的服务必须带有效 token；`open_auth=0` 的服务带有效 token 时也会识别租户，用于租户流量统计

### 7. 流量统计
- 每个请求计入：全站计数、服务计数，识别到租户时再计入租户计数
- 计数先在内存原子累加，每秒批量写入 Redis（MULTI + HINCRBY + EXPIRE 2 天），写失败会把计数加回，下次重试
- Redis 结构：hash，field 为 `YYYYMMDDHH`

| 维度 | Redis key |
| --- | --- |
| 全站 | `flow_count_total` |
| 服务 | `flow_count_service_<service_name>` |
| 租户 | `flow_count_app_<app_id>` |

dashboard 的 `flow_stat`、`service_statistics`、`app_stat`、`panel_group_data` 读取这些 key 展示今日/昨日按小时的流量。

### 8. 限流
**服务级（`HTTPFlowLimitMiddleware`）**
- 令牌桶（`golang.org/x/time/rate`）
- `service_flow_limit`：服务总 QPS，超出返回 5001
- `clientip_flow_limit`：单个客户端 IP 的 QPS，超出返回 5002
- 值为 0 表示不限流；修改 QPS 后热加载会重建限流器；10 分钟未使用的限流器自动清理
- 客户端 IP 取 TCP 连接的 `RemoteAddr`，不信任 `X-Forwarded-For`，防止伪造绕过

**租户级（`HTTPJwtFlowLimitMiddleware`，`gateway_app` 表的 `qps` / `qpd`）**
- 只对带有效 token、已识别出租户的请求生效；值为 0 表示不限
- `qps`：租户每秒请求数，令牌桶，所有服务共用一个桶，超出返回 5003
- `qpd`：租户每天请求数，当天请求量 ≥ qpd 时返回 5004，次日 0 点自动恢复
  - 当天请求量 = Redis `flow_count_app_<app_id>` 中当天各小时之和 + 本实例尚未写入 Redis 的计数
  - 租户计数器每秒从 Redis 读回当天总量，多个网关实例共享同一份额度；计数器创建时同步读取，网关重启后限流立即生效
  - 多实例部署时各实例最多有约 1 秒的计数尚未同步，可能略微超出 qpd
- 在 dashboard 修改租户 qps/qpd 后，随租户热加载生效

### 9. IP 黑白名单
- 仅在 `open_auth=1` 时生效
- `white_list` 非空时只允许名单内 IP（3001）
- `white_list` 为空时 `black_list` 才生效，名单内 IP 拒绝（3002）
- 多个 IP 用逗号或换行分隔

### 10. 请求改写
执行顺序：header 改写 → strip_uri → url_rewrite。
- **header_transfor**：每条 `add name value` / `edit name value` / `del name`，多条用逗号或换行分隔
- **need_strip_uri=1**（仅前缀接入）：去掉服务前缀，如规则 `/test_strip_uri` 时 `/test_strip_uri/abc` → `/abc`
- **url_rewrite**：每条 `正则 替换`，如 `^/abb/(.*) /bba/$1`；在 strip_uri **之后**执行，所以正则要按去掉前缀后的路径写；正则按规则缓存

### 11. 负载均衡与反向代理
- 下游地址取自 `ip_list`（逗号或换行分隔）
- 目前实现**随机**负载均衡（`reverse_proxy/load_balance`），`round_type` 暂不区分
- 每个服务缓存一个负载均衡器和一个连接池（`http.Transport`），`ip_list`/`round_type` 或上游参数变化时自动重建，旧连接池只关闭空闲连接
- 上游参数（0 表示用默认值）：

| 字段 | 含义 | 默认 |
| --- | --- | --- |
| `upstream_connect_timeout` | 建连超时（秒） | 30 |
| `upstream_header_timeout` | 等待响应头超时（秒） | 30 |
| `upstream_idle_timeout` | 空闲连接超时（秒） | 90 |
| `upstream_max_idle` | 最大空闲连接数 | 100 |

### 12. 配置热加载
- `proxy.service.reload_interval` 秒定时从 MySQL 重新加载服务和租户
- 加载成功后整体原子替换，加载失败保留旧配置
- dashboard 中修改服务/租户后，最多一个周期内在代理生效，无需重启

---

## 三、配置文件（`conf/dev/`）

| 文件 | 内容 |
| --- | --- |
| `base.toml` | 日志、dashboard 监听地址、swagger、session、集群信息 |
| `mysql_map.toml` | MySQL 连接 |
| `redis_map.toml` | Redis 连接 |
| `proxy.toml` | 代理服务配置，见下 |

`proxy.toml`：
```toml
[http]
    addr = "127.0.0.1:8080"
    read_timeout = 10
    write_timeout = 10
    max_header_bytes = 20

[https]
    addr = "127.0.0.1:4433"
    cert_file = "./conf/dev/cert/cert.pem"
    key_file = "./conf/dev/cert/key.pem"

[service]
    reload_interval = 10     # 服务/租户热加载间隔（秒），0 表示不热加载

[jwt]
    secret = "..."           # JWT 签名密钥，生产环境请单独配置，不要提交到仓库
    expire_seconds = 7200    # token 有效期（秒）
```

> 监听地址绑定 `127.0.0.1`，避免 Windows 防火墙每次 `go run` 都弹窗。

---

## 四、错误码汇总（代理服务）

| 错误码 | 含义 |
| --- | --- |
| 1001 | 未匹配到服务 |
| 2001 | 上下文无服务 / `/oauth/tokens` 缺少 Basic 认证 |
| 2002 | 租户认证失败（token 缺失、无效、过期，租户不存在；或 app_id/secret 错误） |
| 2003 | token 签发失败 |
| 3001 | 不在 IP 白名单 |
| 3002 | 在 IP 黑名单 |
| 4001 | 服务不支持 https |
| 4002 | 服务不支持 websocket |
| 5001 | 服务限流 |
| 5002 | 客户端 IP 限流 |
| 5003 | 租户 QPS 限流 |
| 5004 | 租户日请求量（QPD）限流 |

---

## 五、目录结构

| 目录 | 说明 |
| --- | --- |
| `controller/` | dashboard 接口 |
| `dto/` | 接口入参/出参 |
| `dao/` | 数据表模型；服务/租户管理器、负载均衡/连接池缓存、流量计数器、限流器 |
| `middleware/` | dashboard 中间件及统一响应 |
| `router/` | dashboard 路由与 HTTP 服务 |
| `http_proxy_router/` | 代理服务路由、HTTP/HTTPS 服务、`/oauth/tokens` |
| `http_proxy_middleware/` | 代理中间件链 |
| `reverse_proxy/` | 反向代理与负载均衡 |
| `public/` | 公共方法（JWT 编解码等） |
| `docs/` | swagger 文档 |

---

## 六、待实现

- 租户 `white_ips` 校验
- 轮询、加权轮询、一致性哈希负载均衡（按 `round_type`）
- 服务 `white_host_name` 校验
- 下游健康检查（`check_method` / `check_timeout` / `check_interval`）
- TCP、GRPC 代理（目前只有管理后台的配置接口）
