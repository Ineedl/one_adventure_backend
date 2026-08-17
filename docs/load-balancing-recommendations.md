# 系统负载均衡方案

## 1. 目标

本方案适用于以下架构：

- Gateway 是多实例 HTTP 主入口；
- Gate Server 是多实例 WebSocket 主入口，负责长连接和长连接业务请求；
- 微服务之间使用 gRPC，并且可能互相调用；
- etcd 负责服务注册与发现；
- 同一个微服务可能部署多个实例。

负载均衡需要同时解决入口流量分配、服务间调用、WebSocket 长连接分配、故障实例摘除和滚动发布问题。

## 2. 分层架构

```text
                         Internet
                            |
                  DNS / 云负载均衡器
                            |
            +---------------+---------------+
            |                               |
      HTTP Ingress                     WebSocket Ingress
            |                               |
   Gateway-1 / Gateway-2          Gate-1 / Gate-2 / Gate-3
            |                               |
            +---------------+---------------+
                            |
                  etcd 服务注册与发现
                            |
                 gRPC 客户端负载均衡
                            |
          +-----------------+-----------------+
          |                 |                 |
       User 集群         Item 集群       Computing 集群
```

建议将负载均衡划分为三层：

1. 外部流量到 Gateway/Gate Server；
2. Gateway/Gate Server 到微服务；
3. 微服务到其他微服务。

## 3. HTTP 入口负载均衡

### 推荐方案

```text
Client -> 云负载均衡器 -> Kubernetes Ingress -> Gateway Pods
```

Gateway 应保持无状态，任意 HTTP 请求都可以由任意实例处理。认证信息使用 JWT、Token 或共享存储，不应依赖某个 Gateway 实例的本地内存。

推荐策略：

- 普通 HTTP API 使用 `round_robin`；
- 实例性能不同或请求耗时差异较大时使用 `least_request`；
- 配置 readiness，只向就绪实例发送请求；
- 配置连接、请求、空闲和响应超时；
- 配置入口限流和最大请求体；
- 滚动发布时先摘流，再等待在途请求完成。

### 推荐技术

| 场景 | 推荐技术 | 说明 |
|---|---|---|
| 公有云入口 | 云厂商 L4/L7 Load Balancer | 承担公网 IP、带宽和基础高可用 |
| Kubernetes HTTP 入口 | Envoy Gateway | 支持 Gateway API、gRPC、流量治理和可观测性 |
| 简单成熟方案 | NGINX Ingress Controller | 配置简单、生态成熟 |
| 已计划使用服务网格 | Istio Ingress Gateway | 与 Envoy Sidecar、mTLS 和流量策略统一 |

当前推荐：**云负载均衡器 + Envoy Gateway**。如果部署环境简单、团队更熟悉 NGINX，也可以先使用 NGINX Ingress。

## 4. WebSocket 入口负载均衡

### 连接分配

WebSocket 建立后是一条长期保持的 TCP 连接，负载均衡只在连接建立时选择 Gate Server，后续消息沿同一连接进入同一实例。

```text
WebSocket Client
       |
       v
L4/L7 Load Balancer
       |
       +--> Gate-1：10,000 条连接
       +--> Gate-2：9,800 条连接
       +--> Gate-3：10,200 条连接
```

不建议只使用普通请求轮询判断 Gate 负载。应关注当前连接数、CPU、内存和事件循环延迟。可优先选择 `least_conn`；如果入口组件无法使用连接数算法，则使用轮询并通过连接上限和 HPA 控制容量。

### 跨实例消息路由

负载均衡不能解决“用户连接在哪个 Gate 实例”的问题。必须额外维护：

```text
Redis: user_id/session_id -> gate_instance_id
消息总线: 业务服务 -> 目标 Gate 实例
Gate 本地内存: connection_id -> websocket connection
```

业务服务向用户推送消息时，应先通过共享路由或消息主题投递到目标 Gate 实例，再由该实例写入 WebSocket。

### Sticky Session

完成共享连接路由后，不需要依赖 Sticky Session。客户端断线重连可以连接到任意 Gate 实例，并重新注册连接。

在共享路由尚未完成前，可以临时使用基于 Cookie 或源 IP 的会话保持，但它不能解决实例故障、NAT 下流量倾斜和跨实例主动推送问题。

### 推荐技术

| 能力 | 推荐技术 |
|---|---|
| WebSocket 入口 | Envoy Gateway、NGINX Ingress 或云 L4 LB |
| 连接路由表 | Redis Cluster/Sentinel |
| 实例间消息路由 | NATS JetStream 或 Redis Streams |
| 大规模可靠事件流 | Kafka |
| 连接数和延迟指标 | Prometheus + Grafana |

当前推荐：**Envoy Gateway + Redis + NATS JetStream**。如果现阶段希望减少组件数量，可先使用 **Redis + Redis Streams**。

## 5. gRPC 服务间负载均衡

### 当前阶段方案

保留现有的：

```text
etcd -> manual resolver -> gRPC round_robin -> 多个服务实例
```

每个调用方，包括 Gateway、Gate Server 和普通微服务，都使用统一的 discovery/client 组件获取实例列表。

需要补充：

- 实例 Ready 后再注册；
- 停机时先从 etcd 摘除，再等待请求完成；
- gRPC Keepalive 和连接空闲配置；
- 每个下游独立的超时和并发上限；
- 只对幂等请求进行有限重试；
- 熔断、失败率和实例级调用指标。

### 负载均衡算法

| 算法 | 适用场景 | 建议 |
|---|---|---|
| `round_robin` | 实例规格一致、请求耗时接近 | 当前默认选择 |
| 加权轮询 | 实例规格或版本容量不同 | 灰度和混合规格时使用 |
| 最少请求 | 请求耗时差异明显 | 引入 Envoy 后优先考虑 |
| 一致性哈希 | 请求需要稳定落到同一分片 | 只用于明确的分片/缓存场景 |
| 随机/Power of Two Choices | 大规模实例集 | 当前规模通常不需要 |

不要为了“用户固定到某实例”对普通业务 RPC 使用一致性哈希。服务应尽量无状态，共享状态放入数据库或 Redis。

## 6. Envoy 引入方案

Envoy 可以承担入口和服务间代理负载均衡，但不能直接读取当前 etcd 的自定义注册数据。可选择以下路线：

### 路线 A：保留应用内负载均衡

```text
etcd -> gRPC Resolver -> round_robin
```

优点是改造小、性能路径短；缺点是超时、熔断等策略需要在 Go 公共库中统一实现。

### 路线 B：Kubernetes Service + Envoy

```text
服务 -> 本地 Envoy -> Kubernetes Service/Endpoint -> 目标实例
```

适合需要统一 mTLS、灰度、熔断和跨语言治理的阶段。

### 路线 C：etcd + 自建 xDS 控制面

```text
etcd -> xDS Control Plane -> Envoy EDS/CDS -> 目标实例
```

该方案能够保留 etcd，但需要自行维护控制面、高可用、配置版本和故障恢复。除非有明确需求，不建议当前阶段采用。

## 7. 健康检查和故障摘除

每个实例至少提供：

- Startup：进程和必要初始化是否完成；
- Readiness：是否可以接收新流量；
- Liveness：进程是否已经无法继续工作。

推荐退出顺序：

```text
标记 NotReady
  -> 从 etcd 注销
  -> 停止接收新请求/新连接
  -> 等待 HTTP/gRPC 在途请求完成
  -> Gate 通知客户端重连并等待宽限期
  -> 关闭进程
```

数据库或 etcd 短暂异常通常只应影响 Readiness，不应立即触发所有实例重启。

## 8. 容量和扩缩容

### Gateway

根据 CPU、请求并发、P95/P99 延迟和每秒请求数扩容。

### Gate Server

根据以下指标综合扩容：

- 当前 WebSocket 连接数；
- 每秒收发消息数；
- CPU 和内存；
- 事件循环/消息处理延迟；
- 单实例连接上限。

不建议只按 CPU 对 Gate Server 扩容，因为空闲长连接可能占用大量内存但 CPU 很低。

### 普通微服务

根据 CPU、活跃请求数、队列长度和 RPC 延迟扩容。HPA 依赖 Prometheus Adapter 或其他自定义指标组件。

## 9. 推荐技术清单

| 层级 | 首选技术 | 可选技术 |
|---|---|---|
| 公网入口 | 云 L4/L7 Load Balancer | MetalLB（自建 Kubernetes） |
| Kubernetes 入口 | Envoy Gateway | NGINX Ingress、Istio Gateway |
| HTTP Gateway 负载均衡 | Envoy `round_robin`/`least_request` | NGINX `least_conn` |
| WebSocket Gate 负载均衡 | Envoy/云 L4 LB `least_conn` | NGINX Ingress |
| gRPC 服务发现 | 现有 etcd + gRPC Resolver | Kubernetes DNS/EndpointSlice |
| gRPC 实例均衡 | gRPC `round_robin` | Envoy `least_request` |
| Gate 连接路由 | Redis Cluster/Sentinel | 其他高可用 KV |
| Gate 消息路由 | NATS JetStream | Redis Streams、Kafka |
| 指标监控 | Prometheus + Grafana | 云监控平台 |
| 分布式追踪 | OpenTelemetry + Tempo | Jaeger |
| 自动扩缩容 | Kubernetes HPA + Prometheus Adapter | KEDA |

## 10. 推荐落地顺序

### 第一阶段

1. Gateway 和 Gate Server 配置多副本、readiness 和优雅退出；
2. 外部入口使用云 LB + Envoy Gateway 或 NGINX Ingress；
3. 服务间继续使用 etcd + gRPC `round_robin`；
4. 补齐 gRPC 超时、有限重试、熔断和并发隔离；
5. 建立 Gateway、Gate 和各微服务的实例级指标。

### 第二阶段

1. 使用 Redis 建立 Gate 连接路由表；
2. 使用 NATS JetStream 或 Redis Streams 实现跨实例消息投递；
3. Gate 根据连接数和消息延迟扩缩容；
4. Gateway 和无状态微服务接入 HPA；
5. 完成多实例故障和滚动发布演练。

### 第三阶段

当服务数量、语言种类或灰度发布需求明显增加时，再使用 Envoy Sidecar 或 Istio 统一服务间流量治理。不要同时长期维护应用内治理和服务网格两套相互冲突的重试、熔断策略。

## 11. 最终推荐

当前最适合该系统的组合是：

```text
外部入口：云负载均衡器
HTTP/WebSocket 路由：Envoy Gateway
Gateway：无状态多实例
Gate Server：多实例 + Redis 连接路由 + NATS JetStream 消息路由
服务发现：保留 etcd
服务间负载均衡：保留 gRPC round_robin
健康检查与扩容：Kubernetes Probe + HPA/KEDA
监控：Prometheus + Grafana + OpenTelemetry
```

该组合能够在控制改造成本的同时解决 HTTP 多入口、WebSocket 长连接、多实例服务调用和故障摘除问题。Envoy Sidecar 或完整服务网格应作为后续演进，而不是当前实施的前置条件。
