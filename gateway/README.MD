# GoFrame Template For SingleRepo

Quick Start: 
- https://goframe.org/quick

## WebSocket

WebSocket 服务由 `internal/websocket` 独立包提供，与 HTTP 服务运行在同一
GoFrame 进程中。端口和路由等参数位于 `manifest/config/config.yaml` 的
`websocket` 节点。

启动项目：

```bash
go run .
```

默认连接地址为 `ws://localhost:8080/ws`。握手响应头
`X-WebSocket-Connection-ID` 是服务端分配的连接 ID，可通过
`Server.Manager()` 查询连接、关闭连接或主动发起请求。

请求格式：

```json
{"type":"PING_REQ","session_id":"会话 ID","data":{"message":"hello"}}
```

响应格式：

```json
{"type":"PING_RESP","session_id":"同一个会话 ID","code":0,"data":{"message":"hello"}}
```

`data` 在 Go 中使用 `json.RawMessage`。`type` 以 `_REQ` 结尾时作为请求处理，
以 `_RESP` 结尾时作为响应处理；响应通过 `session_id` 与待处理请求关联。
默认请求处理器会原样返回 `data`，业务可通过
`wsServer.Manager().SetRequestHandler(...)` 替换。

## Gateway RPC

Gateway gRPC 服务端口位于 `manifest/config/config.yaml` 的 `rpc.gateway.port`，
默认监听 `8081`。服务启动时会注册 `gateway.GatewayService`，当前提供
`RegisterGateway` RPC。注册成功后由 Gateway 主动周期调用微服务统一的
`ping.PingService`，微服务不再调用 Gateway 心跳接口。
