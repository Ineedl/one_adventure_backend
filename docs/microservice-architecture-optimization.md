# 微服务架构优化方案

## 一、现状判断

当前项目具备以下基础：

- Gateway、User、Item、Computing、Server Manager 等服务独立部署。
- 使用 etcd 进行服务注册与发现。
- 服务实例通过 lease/KeepAlive 自动摘除。
- gRPC 客户端使用 manual resolver，并配置了 `round_robin` 负载策略。
- Gateway 同时承载 HTTP 和 WebSocket。
- 当前 Kubernetes Deployment 默认只有 `replicas: 1`。
- 项目依赖中已经出现 OpenTelemetry，但目前没有看到完整的 Trace 注入、传递和采集闭环。
- 服务之间主要是基于 gRPC 的调用，Gateway 负责 HTTP 请求转发。

整体来看，服务发现和 gRPC 负载均衡已有雏形，但在链路追踪、故障隔离、入口负载均衡、限流熔断和 Kubernetes 高可用方面还需要完善。

---

# 1. 全链路 Trace ID 追踪方案

目标是保证一个请求从客户端进入 Gateway 开始，经过 HTTP、Gateway、gRPC、内部 HTTP 或 WebSocket 后，所有日志都可以通过同一个 `trace_id` 关联起来。

## 1.1 推荐使用 OpenTelemetry

建议统一采用：

- OpenTelemetry SDK
- OTLP 协议
- Jaeger、Tempo 或 SkyWalking 作为 Trace 存储和查询系统
- Prometheus 负责指标
- Loki、ELK 或其他日志系统负责日志

推荐链路：

```text
客户端
  |
  v
Gateway HTTP
  |
  |-- traceparent / x-trace-id
  v
Gateway gRPC Client
  |
  v
User / Item / Computing
  |
  v
数据库、Redis、第三方 HTTP 服务
```

## 1.2 Trace ID 生成规则

入口 Gateway 需要统一处理：

1. 如果请求携带符合 W3C 标准的 `traceparent`，继续使用原始 Trace。
2. 如果没有 Trace 信息，由 Gateway 创建新的 Trace。
3. 同时生成或提取业务侧的 `request_id`。
4. 将信息写入请求上下文。

建议区分两个概念：

| 字段 | 作用 |
|---|---|
| `trace_id` | 一次完整调用链的唯一标识 |
| `span_id` | 当前调用节点的唯一标识 |
| `request_id` | 业务请求编号，可用于幂等和问题定位 |
| `user_id` | 用户身份 |
| `service_name` | 当前服务名称 |
| `instance_id` | 当前服务实例编号 |

不要只依赖自定义 `x-trace-id`，应优先使用 W3C Trace Context：

```http
traceparent: 00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01
```

为了兼容旧系统，可以额外保留：

```http
x-trace-id: 4bf92f3577b34da6a3ce929d0e0e4736
x-request-id: req-20260812-000001
```

## 1.3 Gateway HTTP 层

Gateway 增加统一 Trace Middleware，建议记录：

- HTTP Method
- URL Path
- HTTP Status
- 请求耗时
- 客户端 IP
- 用户 ID
- Trace ID
- Gateway 实例 ID
- 下游服务名
- 错误类型

日志格式建议为 JSON：

```json
{
  "timestamp": "2026-08-12T10:00:00+08:00",
  "level": "INFO",
  "service": "gateway",
  "instance_id": "gateway-7d9c",
  "trace_id": "4bf92f3577b34da6a3ce929d0e0e4736",
  "span_id": "00f067aa0ba902b7",
  "request_id": "req-20260812-000001",
  "method": "GET",
  "path": "/api/item/123",
  "status": 200,
  "duration_ms": 35
}
```

注意：日志必须从 `context.Context` 中读取 Trace 信息，不能依赖全局变量。

## 1.4 Gateway 到 gRPC 服务

需要在 gRPC Client 端增加 Unary Interceptor 和 Stream Interceptor：

- 出站请求注入 Trace Context。
- 将 `traceparent` 放入 gRPC Metadata。
- 在 gRPC 服务端 Interceptor 中提取 Metadata。
- 为每次 RPC 创建独立 Span。

逻辑如下：

```text
Gateway Span
  |
  | gRPC Metadata 注入 traceparent
  v
User Server Span
```

每个 gRPC 方法建议记录：

- RPC Service
- RPC Method
- 请求耗时
- gRPC Status Code
- 下游实例地址
- 重试次数
- 是否触发熔断

## 1.5 服务到服务的 HTTP 调用

如果微服务之间存在 HTTP 调用，应统一封装 HTTP Client，不允许业务代码直接创建裸 `http.Client`。

封装层需要自动完成：

- 创建 Client Span
- 注入 `traceparent`
- 注入 `x-request-id`
- 设置连接超时
- 设置请求超时
- 记录状态码和耗时
- 统一处理重试和错误

建议设置超时层级：

```text
Gateway 总超时：3 秒
  |
  |-- gRPC 调用：1 秒
  |-- HTTP 调用：800 毫秒
  |-- 数据库操作：500 毫秒
```

下游超时必须小于上游超时，防止请求在链路中无限堆积。

## 1.6 WebSocket Trace 处理

WebSocket 与普通 HTTP 不同，连接建立时可以创建一个连接级 Span：

```text
WebSocket Upgrade Span
```

连接建立后，每条消息建议携带：

```json
{
  "session_id": "xxx",
  "trace_id": "xxx",
  "request_id": "xxx",
  "message_type": "request",
  "data": {}
}
```

建议策略：

- WebSocket Upgrade 阶段继承 HTTP Trace。
- 每条消息创建独立 Span。
- 服务端向后端发送请求时重新注入 Trace Context。
- 日志中同时记录连接 ID、用户 ID、Trace ID 和 Session ID。

连接级 Span 不建议长期保持数小时，否则会造成 Trace 数据过大。可以将连接 Span 作为低采样率摘要，消息级 Span 单独上报。

## 1.7 采样策略

生产环境不建议 100% 采样，否则高流量下成本很高。

建议：

- 错误请求：100% 采样
- 慢请求：100% 采样
- 普通请求：5%～10%
- WebSocket 消息：按错误和慢请求采样
- 核心交易接口：提高采样率

## 1.8 Trace 落地优先级

建议按以下顺序实施：

1. Gateway HTTP Trace Middleware
2. gRPC Client/Server Interceptor
3. 统一 HTTP Client
4. 数据库和 Redis Instrumentation
5. WebSocket 消息级 Trace
6. 日志关联 Trace ID
7. 接入 Jaeger/Tempo/SkyWalking
8. 建立错误 Trace 和慢调用告警

---

# 2. 微服务之间网状连接是否需要优化

## 2.1 当前网状连接的问题

服务之间完全网状调用会带来以下问题：

- 服务依赖关系复杂。
- 任意服务都可能被大量其他服务调用。
- 故障影响范围难以预测。
- 容易形成循环依赖。
- 服务升级需要同时考虑大量调用方。
- 很难定义清晰的限流、熔断和权限边界。
- 出现故障时，容易发生级联雪崩。

## 2.2 是否需要完全中心化

不建议把所有调用都改成经过 Gateway。

Gateway 适合处理外部 HTTP/HTTPS 请求、身份认证、参数校验、API 路由、限流、协议转换和 WebSocket 接入；不适合承载所有内部服务调用，否则会成为单点瓶颈并额外增加调用跳数。

## 2.3 推荐的目标架构

建议采用“分层 + 有向依赖”的架构，而不是完全网状连接：

```text
                 ┌──────────────┐
                 │  外部客户端   │
                 └──────┬───────┘
                        v
              ┌──────────────────┐
              │ Gateway / WS LB  │
              └──────┬─────┬─────┘
                     v     v
                 ┌─────┐ ┌─────┐
                 │User │ │Item │
                 └──┬──┘ └──┬──┘
                    v       v
                 ┌────────────┐
                 │ Computing  │
                 └────────────┘
```

建议遵循：

- Gateway 只依赖业务服务。
- 业务服务只依赖下层服务。
- 禁止形成循环依赖。
- 公共能力抽取为基础服务或 SDK。
- 同一领域内的服务优先通过领域 API 协作。
- 跨领域复杂流程优先使用事件或异步消息。
- 数据库按服务边界隔离，避免共享表。

## 2.4 依赖治理

建议维护一份服务依赖矩阵：

| 调用方 | 被调用方 | 协议 | 是否同步 | 超时 | 是否允许重试 |
|---|---|---|---|---|---|
| Gateway | User | gRPC | 是 | 1 秒 | 是 |
| Gateway | Item | gRPC | 是 | 1 秒 | 是 |
| Item | Computing | gRPC | 是 | 500 毫秒 | 仅幂等接口 |
| Item | User | gRPC | 是 | 500 毫秒 | 仅查询接口 |

同时增加以下约束：

- 每个服务必须声明依赖服务。
- 禁止未登记的服务调用。
- 禁止服务直接访问其他服务数据库。
- 禁止同步调用链超过 3～4 层。
- 禁止出现循环依赖。
- 关键流程拆分同步和异步部分。

## 2.5 什么时候使用消息队列

以下场景建议改为异步消息：

- 日志和审计
- 通知推送
- 统计计算
- 非实时数据同步
- 批量任务
- 订单后置处理
- 失败后可重试的任务

同步请求只保留用户必须立即得到结果的部分。

---

# 3. 单实例故障后的级联雪崩防护

典型解决方案是健康检查、负载均衡、限流、熔断、舱壁隔离、自动扩容、优雅下线和过载保护的组合，而不是只增加实例数量。

## 3.1 Kubernetes 副本和反亲和

当前 Deployment 默认 `replicas: 1`，生产环境至少设置为 3，并增加 Pod Anti-Affinity、多节点部署、PodDisruptionBudget、RollingUpdate 以及资源 requests/limits。

```yaml
spec:
  replicas: 3
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 0
      maxSurge: 1
```

使用 `topologySpreadConstraints` 避免多个实例调度到同一节点。

## 3.2 Readiness、Liveness 和 Startup Probe

### Readiness Probe

表示“是否可以接收新流量”。数据库未连接、依赖服务不可用、当前实例过载、正在优雅下线、线程池或连接池耗尽时应返回失败。

### Liveness Probe

表示“进程是否还活着”。不应把所有下游依赖都放入 Liveness，否则一个数据库短暂故障会导致所有 Pod 被重启。

### Startup Probe

用于启动时间较长的服务，避免服务刚启动时被误判为故障。

## 3.3 服务发现剔除异常实例

etcd lease 可以自动移除失联实例，但还应补充：

- 注册前先确认服务已可以提供请求。
- 健康状态变化时主动更新注册状态。
- 优雅退出时先取消注册，再停止接收请求。
- 增加 `ready`、`load`、`version` 等实例状态信息。

## 3.4 客户端负载均衡和故障转移

当前 gRPC 已使用 `round_robin`，建议进一步完善：

- 每个服务至少 3 个实例。
- 对异常连接快速摘除。
- 配置连接状态监测。
- 只对幂等请求进行有限重试。
- 使用指数退避和随机抖动。
- 设置最大重试次数，通常 1～2 次。
- 禁止无限重试。

仅建议对 `UNAVAILABLE`、`RESOURCE_EXHAUSTED` 和短暂网络错误重试；参数、权限、业务校验错误以及非幂等写请求不应重试。

## 3.5 熔断器

每个调用方到每个下游服务都应独立配置熔断器。当下游持续失败时快速失败、返回降级结果或明确错误，冷却后通过半开探测逐步恢复。

参考配置：

```text
连续失败次数：5
失败率阈值：50%
熔断窗口：10 秒
半开探测：每秒 1 个请求
```

## 3.6 舱壁隔离

为每个下游服务设置独立的最大并发数、请求队列、超时时间、连接池、Goroutine/线程池、熔断器和限流器，避免一个下游服务耗尽整个 Gateway 的资源。

## 3.7 限流和过载保护

建议按“客户端/IP → Gateway 全局 → Gateway 接口 → 服务实例 → 数据库连接池”多层限流。限流维度可包括 IP、用户 ID、API Key、租户、接口、服务和实例。过载时主动返回 `429` 或 `503`，不要无限排队。

## 3.8 自动扩缩容

HPA 指标不要只看 CPU，还应考虑 QPS、P95/P99 延迟、活跃请求数、gRPC 并发数、队列长度和内存。Computing 等计算型服务可使用基于队列长度的扩容策略。

---

# 4. Gateway 和 WebSocket 层前增加负载均衡

## 4.1 Gateway 前的负载均衡

推荐部署结构：

```text
Internet
   |
   v
云负载均衡 / Nginx / Envoy / Ingress
   |
   +--> Gateway Pod 1
   +--> Gateway Pod 2
   +--> Gateway Pod 3
```

可使用云厂商 LoadBalancer、Nginx Ingress、Traefik、Envoy Gateway 或 HAProxy。入口负载均衡负责 TLS 终止、HTTP/2、Keep-Alive、连接数限制、基础限流、黑白名单、健康检查、灰度发布和访问日志。

## 4.2 WebSocket 前的负载均衡

WebSocket 需要处理 TCP 长连接、Upgrade 请求透传、连接超时、心跳、断线重连和多实例会话状态。Ingress 需要启用 HTTP/1.1 Upgrade，并设置合理的读写超时时间。

## 4.3 是否需要 Sticky Session

优先将 WebSocket 设计为无状态接入层，连接信息、房间信息和在线状态放入 Redis、Redis Cluster、共享会话服务、消息队列或事件总线。Sticky Session（Cookie 或 IP Hash）只能作为短期过渡方案。

## 4.4 WebSocket 高可用设计

建议客户端自动重连，服务端实现 Ping/Pong 心跳，设置最大连接数、单用户最大连接数、单连接消息速率限制和空闲超时；Pod 下线前停止接收新连接，并将消息路由放到 Redis Streams 或可靠消息队列。重要消息不建议只使用 Redis Pub/Sub，以免发生消息丢失。

---

# 5. 是否有必要全部重构

## 5.1 不建议一次性全部重构

当前项目已经具备 etcd 服务发现、gRPC 调用、gRPC `round_robin`、Gateway、WebSocket 和业务服务边界。主要问题更偏向于治理能力不足，而不是服务划分完全错误。全量重构会同时引入业务回归、数据迁移、上线风险和排障成本。

## 5.2 推荐渐进式改造

### 第一阶段：可观测性

完成 Trace ID 全链路传递、结构化日志、Prometheus 指标、错误率和延迟监控，并接入 Jaeger、Tempo 或 SkyWalking。

### 第二阶段：高可用

所有核心服务至少 3 副本，增加探针、优雅下线、HPA、PDB、Pod 反亲和，以及 Gateway/WebSocket 前置负载均衡。

### 第三阶段：流量治理

增加 Gateway 限流、服务级限流、gRPC 超时、重试、熔断、舱壁隔离、降级策略和异步化处理。

### 第四阶段：服务依赖治理

建立服务依赖矩阵，清理循环依赖，限制跨服务调用范围，禁止跨服务访问数据库，并将高频、强耦合调用重新设计为领域服务或事件。

## 5.3 何时考虑大规模重构

只有在服务已形成大量循环依赖、共享数据库导致无法独立发布、Gateway 承担大量业务逻辑、同步调用链经常超过 5 层、单次故障会大范围级联或现有技术栈无法满足性能和稳定性要求时，才建议进行较大范围的架构重构。

---

# 推荐目标架构

```text
                         ┌─────────────────┐
                         │  Jaeger/Tempo   │
                         │  Prometheus     │
                         │  Loki/ELK       │
                         └────────▲────────┘
                                  │
Client
  │
  v
┌──────────────────────────────────────────┐
│ Cloud LB / Nginx / Envoy / Ingress       │
│ TLS、限流、健康检查、HTTP/WS Upgrade     │
└───────────────┬──────────────────────────┘
                │
        ┌───────┴────────┐
        │                │
        v                v
┌──────────────┐  ┌──────────────┐
│ Gateway Pods  │  │ WebSocket    │
│ 3+ replicas   │  │ Gateway Pods │
└──────┬───────┘  └──────┬───────┘
       │                 │
       v                 v
┌──────────────┐   ┌──────────────┐
│ User Service │   │ Redis/Stream │
└──────┬───────┘   └──────────────┘
       │
       v
┌──────────────┐
│ Item Service │
└──────┬───────┘
       │
       v
┌──────────────┐
│ Computing    │
│ 限流/熔断/隔离 │
└──────────────┘
```

# 最终建议

1. 立即补充 OpenTelemetry Trace 注入和日志关联。
2. 将所有核心服务副本数从 1 提升到至少 3。
3. 增加 Readiness、Liveness、Startup Probe。
4. 在 Gateway 和 WebSocket 前部署 Ingress 或云负载均衡。
5. 增加服务级超时、限流、熔断和舱壁隔离。
6. 保留 etcd 服务发现和 gRPC `round_robin`，但完善实例健康状态。
7. 对 WebSocket 做无状态化，连接状态放入 Redis 或共享存储。
8. 建立服务依赖矩阵，逐步消除循环依赖。
9. 暂时不要全量重构，采用渐进式演进。
10. 通过故障演练验证单实例宕机、整服务不可用、流量突增、etcd 短暂不可用、Redis 故障和 Gateway 重启等场景。
