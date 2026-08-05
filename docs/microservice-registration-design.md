# 微服务注册与 Gateway 主动探活设计

## 1. 目标

微服务只负责向 Gateway 注册，不再循环调用 Gateway 的保活接口。Gateway 在注册时以及注册成功后，主动调用微服务统一的 `PingService.Ping` 接口确认实例可用。

公共 Ping 实现在 `servicekit/ping` 中，注册重试和 Ping 租约监控在 `servicekit/registration` 中。业务微服务只需要注册自身业务 Service、接入 `registration.Manager`，不需要重复实现 Ping Handler。

## 2. 职责边界

- `rpc`：定义 Gateway 注册协议和公共 Ping 协议。
- `servicekit/ping`：实现所有微服务共用的 `PingService`，校验服务类型、实例 ID 和本轮注册 Token。
- `servicekit/registration`：生成实例 ID 和注册 Token，执行注册、失败退避，以及监控 Gateway Ping 租约。
- `gateway`：验证注册参数，连接微服务，主动调用 Ping，并维护可用实例注册表。
- 业务微服务：先启动自身 gRPC Server，再运行注册管理器。

## 3. 协议

公共 Ping 协议：

```proto
message PingReq {
  string type = 1;
  string instance_id = 2;
  string registration_token = 3;
}

message PingResp {
  int32 code = 1;
  string message = 2;
}

service PingService {
  rpc Ping(PingReq) returns (PingResp);
}
```

Gateway 只保留注册 RPC，不再提供由微服务调用的 `Heart` RPC：

```proto
message RegisterReq {
  string type = 1;
  string ip = 2;
  int32 port = 3;
  int64 register_time = 4;
  string version = 5;
  int32 weight = 6;
  string instance_id = 7;
  string registration_token = 8;
}

message RegisterResp {
  int32 code = 1;
  string message = 2;
  int64 ping_interval_ms = 3;
  int64 lease_duration_ms = 4;
}

service GatewayService {
  rpc RegisterGateway(RegisterReq) returns (RegisterResp);
}
```

`registration_token` 每轮注册都重新生成。Gateway 的后台 Ping 和注册表删除操作必须匹配当前 Token，避免旧请求误删同一实例的新注册。

## 4. 注册与探活时序

```text
微服务                                      Gateway
  │                                           │
  ├─ 启动 gRPC Server                         │
  │    ├─ 注册业务 Service                    │
  │    └─ 注册 servicekit PingService         │
  │                                           │
  ├─ 生成 registration_token                  │
  ├─ RegisterGateway ────────────────────────►│
  │                                           ├─ 连接微服务 gRPC 地址
  │◄────────────────────────────── Ping(token)┤
  ├─ 校验当前注册 Token                       │
  ├──────────────────────── PingResp success ►│
  │                                           ├─ 写入/替换注册表
  │◄──────────────────── RegisterResp success ┤
  │                                           │
  │◄──────────────────── 周期性 Ping(token) ──┤
  ├──────────────────────── PingResp success ►│
  │                                           │
```

Gateway 只有在首次 Ping 成功后才接受注册。注册成功后，Gateway 按 `pingInterval` 周期并发探测所有实例；任何实例 Ping 超时、RPC 失败或返回非成功码时，从注册表删除该实例并关闭对应连接。

## 5. 微服务自动恢复

微服务不会向 Gateway 发送保活请求，但会监听公共 Ping Service 收到的有效请求：

- 注册 RPC 成功后直接进入 `Active`。Gateway 已在返回注册成功前完成首次反向 Ping。
- 每次收到有效 Ping 都刷新本地租约计时器。
- 在 `lease_duration_ms` 内未收到 Gateway Ping，则认为 Gateway 已重启、连接中断或注册已丢失，使用新 Token 重新注册。
- 注册失败或 Ping 租约过期后，按带抖动的指数退避重试。

状态机：

```text
Starting → Registering → Active
              │           │
              └───────────┴─► RetryWaiting → Registering
```

## 6. 配置

Gateway：

```yaml
rpc:
  gateway:
    port: 8081
    pingInterval: "10s"
    pingTimeout: "3s"
```

Gateway 返回的租约当前按 `3 * pingInterval + pingTimeout` 计算，允许微服务容忍短暂的调度延迟，同时能在 Gateway 重启后重新注册。

微服务：

```yaml
rpc:
  computing:
    port: 8082
    registration:
      gatewayIp: "127.0.0.1"
      gatewayPort: 8081
      serviceType: "computing"
      serviceIp: "127.0.0.1"
      version: "v1.0.0"
      weight: 10
      instanceId: ""
      registerTimeout: "8s"
      gatewayPingTimeout: "30s"
      retryInitialInterval: "1s"
      retryMaxInterval: "30s"
```

`gatewayPingTimeout` 只在注册成功、进入 `Active` 后开始计时；注册过程中不会因为该超时触发新的注册。`instanceId` 为空时，注册组件生成 `服务名_机器码`。容器环境可用 `ONE_ADVENTURE_MACHINE_ID` 覆盖机器标识来源。

## 7. 微服务接入

```go
manager, err := registration.New(registration.Config{
    GatewayIP:   "127.0.0.1",
    GatewayPort: 8081,
    Service: registration.ServiceInfo{
        Type:    "computing",
        IP:      "127.0.0.1",
        Port:    8082,
        Version: "v1.0.0",
        Weight:  10,
    },
})
if err != nil {
    return err
}

grpcServer := grpc.NewServer()
computingpb.RegisterComputingServiceServer(grpcServer, computingService)
manager.RegisterPingService(grpcServer)

// 必须先开始 Serve，再启动注册管理器。
go grpcServer.Serve(listener)
go manager.Run(ctx)
```

后续新增微服务使用相同接入方式，无需在 Gateway 中增加按业务类型区分的 Ping 客户端逻辑。

## 8. 并发与替换规则

Gateway 注册表使用 `(service_type, instance_id)` 作为联合键：

1. 新注册先建立新连接并完成首次 Ping。
2. Ping 成功后原子替换旧实例。
3. 替换完成后关闭旧连接。
4. 后台 Ping 的成功刷新和失败删除都携带 Token；如果实例已经重新注册，旧探测结果会被忽略。

## 9. 验收标准

- Gateway 协议中不存在 `Heart` RPC。
- 微服务中不存在循环调用 Gateway 保活接口的逻辑。
- 所有微服务通过 `servicekit` 注册公共 `PingService`。
- Gateway 注册时主动 Ping，失败的实例不会进入注册表。
- Gateway 注册后周期性主动 Ping，失败实例会被移除。
- Gateway 重启或长时间不再 Ping 时，微服务会自动重新注册。
- 同一服务类型支持多个不同 `instance_id` 的实例。
- 相同实例重新注册时，旧 Ping 结果不会影响新注册。
