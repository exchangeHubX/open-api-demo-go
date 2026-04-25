# open-api-demo-go

exchangeHubX Open API 的 Go 语言示例代码，覆盖 **REST 签名调用** 与 **WebSocket 鉴权订阅**。

签名规则与协议细节见 [api-docs](../api-docs)：

- [docs/common/sign-code.md](../api-docs/docs/common/sign-code.md)
- [docs/common/websocket.md](../api-docs/docs/common/websocket.md)

## 目录结构

```
.
├── signer/      # 纯签名逻辑（REST 和 WS 共用）
├── client/      # REST HTTP 客户端封装
├── wsclient/    # WebSocket 客户端封装（auth / subscribe / 心跳）
└── examples/
    ├── rest/        # REST 下单 + 查询 demo
    ├── ws/          # 现货 WebSocket demo
    └── future-ws/   # 合约 WebSocket demo
```

## 快速开始

### 1. 准备凭证

把示例代码里的 `appKey` / `secretKey` 换成你自己的：

```go
const (
    appKey    = "ak_xxx"
    secretKey = "sk_xxx"
)
```

### 2. 运行示例

```bash
go run ./examples/rest         # REST 下单 + 查询
go run ./examples/ws           # 现货行情 + 订单推送
go run ./examples/future-ws    # 合约行情 + 订单推送
```

### 3. 运行测试

```bash
go test ./...
```

测试用例覆盖了文档样例的签名结果（`017097d75f9506e2c6e6a074dd5a5556d4aefa8def40a455fe1240a9cd4e5ae9`），可作为签名实现的对照基准。

## REST 用法

```go
import "github.com/exchangeHubX/open-api-demo-go/client"

c := client.New("https://api.j2coin.com", appKey, secretKey)
c.RecvWindow = 60000 // 可选，默认 5000ms

status, body, err := c.Do(client.Request{
    Method: "POST",
    Path:   "/api/v1/orders",
    JSONBody: map[string]any{
        "symbol":      "BTC_USDT",
        "side":        "BUY",
        "type":        "LIMIT",
        "timeInForce": "GTC",
        "bizType":     "SPOT",
        "price":       "0.1",
        "quantity":    "10",
    },
})
```

支持的请求体形式：

| 字段 | 类型 | Content-Type |
|------|------|--------------|
| `JSONBody` | `any`（自动 `json.Marshal`） | `application/json` |
| `FormBody` | `url.Values`（按 key 字典序拼接） | `application/x-www-form-urlencoded` |
| `Query`    | `url.Values`（与 body 互不冲突，可同时使用） | — |

`client.Do` 内部会自动：

1. 生成 4 个签名请求头：`validate-algorithms` / `validate-appkey` / `validate-recvwindow` / `validate-timestamp`。
2. 调用 [`signer.Sign`](signer/signer.go) 计算 `validate-signature`。
3. 拼接最终 URL 和 query 后发起请求。

## WebSocket 用法

```go
import "github.com/exchangeHubX/open-api-demo-go/wsclient"

c := wsclient.New("wss://ws.j2coin.com/ws", appKey, secretKey)
c.Debug = true // 可选：打印每条出站消息（auth / subscribe / ping）

c.OnMessage(func(raw []byte) {
    fmt.Printf("<< %s\n", raw)
})

if err := c.Dial(ctx); err != nil { return err }
defer c.Close()

// 私有频道前先鉴权；订阅纯公有频道（如 ticker/depth/kline）可跳过
if err := c.Auth(); err != nil { return err }

if err := c.Subscribe(
    "ticker@BTC_USDT",
    "depth@BTC_USDT,20",
    "kline@BTC_USDT,1m",
); err != nil { return err }

// 阻塞：内部维护心跳和读循环，直到 ctx 取消或连接断开
return c.Run(ctx)
```

| 端点 | URL |
|------|-----|
| 现货 | `wss://ws.j2coin.com/ws` |
| 合约 | `wss://fws.j2coin.com/ws` |

`Run` 内部按文档要求：

- 每 20s 发送文本帧 `"ping"`。
- 30s 内未收到任意服务端帧则判定断连。
- `"pong"` 文本帧自动过滤，不投递给 `OnMessage`。

## 签名规则速记

最终待签字符串：

```
original = X + Y
X = 按 key 升序的 validate-* 头，"&" 拼接
Y = #METHOD#path[#query][#body]
```

- `query` / 表单 body：按 key 字典序排序，`k=v&k=v`
- JSON body：原样，不重排
- WebSocket auth：`method=GET`，`path=/ws/auth`，无 query/body

`signature = HmacSHA256Hex(secretKey, original)`，放入请求头 `validate-signature`。

## 依赖

- Go 1.21+
- [`github.com/gorilla/websocket`](https://github.com/gorilla/websocket)（仅 `wsclient` 使用）
