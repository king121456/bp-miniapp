# 血压小程序后端（Go + Gin + MongoDB）

统一返回 `{ "code": 0, "data": ... }`，出错为 `{ "code": <HTTP状态>, "message": "..." }`。
除登录外都需请求头 `Authorization: Bearer <token>`（30 天有效）。时间统一以 +08:00 的 RFC3339 返回，按北京时间计算「今天」。
备注、药名、剂量在库里用 AES-256-GCM 加密存储。

## 技术选型

| 层 | 选型 |
|---|---|
| HTTP | [github.com/gin-gonic/gin](https://github.com/gin-gonic/gin) |
| DB | [go.mongodb.org/mongo-driver/v2](https://pkg.go.dev/go.mongodb.org/mongo-driver/v2)（v2.8.x，兼容 Go 1.23+） |
| 日志 | 标准库 `log/slog`（JSON 到 stdout） |
| 配置 | 进程环境变量；启动时可选加载同目录 `.env`（不覆盖已有环境变量） |
| Go | `go 1.23` 起（`go.mod` 写 1.23.0） |

## 目录说明

```
server/                        # 本仓库根即 backend
  cmd/server/main.go           # 入口：load .env、连 Mongo、启 Gin + 分钟调度
  internal/
    config/                    # 环境变量 / .env
    crypto/                    # AES-GCM 字段加密
    auth/                      # HMAC token 签发与校验
    wechat/                    # code2session + 订阅消息（WxAPI 接口便于单测）
    model/                     # 领域结构体与时间辅助
    store/                     # Mongo 集合与索引
    service/                   # 业务逻辑（血压、用药、补记/跳过、复诊、家人、报告、账号）
    httpserver/                # Gin 引擎、中间件、路由适配、验收测试
    scheduler/                 # 每分钟 ticker 提醒
  Dockerfile, docker-compose.yml, .env.example, README.md, go.mod
```

## 如何运行

```bash
cp .env.example .env
# 填好 JWT_SECRET、DATA_KEY、WX_APPID、WX_SECRET 等
go run ./cmd/server
# 或
go build -o bp-server ./cmd/server && ./bp-server
```

默认监听 `:8080`。健康检查：`GET /healthz`。

测试（需本地 Mongo）：

```bash
MONGO_URI=mongodb://127.0.0.1:27027 go test ./...
```

Docker：`cp .env.example .env` 填好后 `docker compose up -d`。

## 调试登录失败（读日志）

服务以 **JSON slog** 打到 **stdout**（Docker 下即 `docker compose logs -f api`）。

与登录相关的字段：

| 场景 | level | 关键键 | 说明 |
|---|---|---|---|
| HTTP 访问 | info/warn/error | `method`, `path`, `status`, `latency_ms` | Gin 中间件；`POST /api/auth/login` 失败多为 `status=502/400/403` |
| code2session 失败 | error | `errcode`, `errmsg`, `appid` | **完整微信错误码与文案**；**从不打印** `WX_SECRET` |
| 登录业务拒绝 | warn | `reason` | 如 `invalid json` / `code required` / `consent required` |
| 登录微信失败（handler） | error | `err`, `code_len` | 与 wechat 包日志对应，便于串起来看 |
| 订阅消息发送失败 | error | `errcode`, `errmsg`, `tpl` | 推送失败时 |

示例排查：

```bash
# 看最近登录相关日志
docker compose logs api 2>&1 | grep -E 'login|code2session|http' | tail -50

# 本地前台跑时直接看终端 JSON
go run ./cmd/server
# 然后用错误的 code 打一次登录，应看到类似：
# {"time":"...","level":"ERROR","msg":"wechat code2session failed","errcode":40029,"errmsg":"invalid code","appid":"wx..."}
# {"time":"...","level":"ERROR","msg":"login wechat failed","err":"code2session 40029 invalid code","code_len":32}
# {"time":"...","level":"WARN","msg":"http","method":"POST","path":"/api/auth/login","status":502,...}
```

常见 `errcode`：`40029` invalid code（code 用过或过期）、`40163` code been used、`41008` missing code、AppID/Secret 配错多为 `40013` / `40125`。

## API 行为

与重构前扁平包布局 **路径、JSON 形状、鉴权、业务规则完全一致**（补记窗口、漏服窗口、软删药品、计划版本、复诊唯一 upcoming 索引等不变）。
CORS 仅在 `APP_ENV=dev` 且设置了 `CORS_ORIGIN` 时开启。

详细接口说明沿用原文档约定（登录、血压、用药、提醒、家人、报告、补记/跳过、复诊、账号导出与注销）。

## 部署注意

- `DATA_KEY` 一旦写入数据不可更换，否则旧密文解不开，请单独备份。
- 生产勿提交 `.env`、`*.exe`、`*.log`（已在 `.gitignore`）。
