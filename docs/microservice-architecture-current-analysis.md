# 微服务架构现状复核与优化分析

## 1. 文档目的

本文依据《microservice-architecture-optimization.md》提出的问题，对当前仓库代码重新检查。原文可继续作为目标方案参考，但其中部分“尚未实现”的判断已经不符合现状，因此本文重点回答以下问题：

1. 哪些能力已经落地；
2. 哪些能力只完成了基础接入，尚未形成生产闭环；
3. 当前最可能引发数据泄露、请求堆积、单点故障和级联雪崩的问题是什么；
4. 后续改造的优先级和验收标准是什么。

本次分析以仓库中的实现和 Kubernetes 清单为准，不代表生产环境外部组件的实际部署状态。

## 2. 结论摘要

当前架构不需要整体推倒重构。服务发现、gRPC 客户端负载均衡、独立 WebSocket 服务以及 Trace/Log 基础设施都已经存在，架构已经从“缺少基础能力”进入“需要补齐治理闭环”的阶段。

最需要优先解决的不是继续拆服务，而是以下五类问题：

1. **调用缺少明确超时和故障隔离**：Gateway 调用 gRPC 时直接沿用请求 Context，没有为下游设置显式 deadline，也没有发现统一的限流、熔断、舱壁和受控重试策略。
2. **Kubernetes 仍是单实例模板**：六个服务的基础 Deployment 都是 `replicas: 1`，并且没有探针、资源配额、PDB、HPA 和跨节点调度约束。
3. **可观测性尚未闭环**：Trace 和 Loki 日志已经接入，但 Prometheus 指标未实现；Trace 默认全量采样，Loki 后台发送失败会被忽略。
4. **观测数据存在敏感信息风险**：GoFrame ORM 自动产生的数据库 Span 包含格式化 SQL，登录等接口可能把用户名、邮箱、Token 或其他参数发送到 Tempo。
5. **Gate Server 仍是单机连接模型**：连接只保存在进程内存，缺少跨实例消息路由、共享在线状态、连接配额、消息限速和完整的心跳治理，当前不能仅通过增加副本获得正确的多实例能力。

## 3. 当前架构

```text
HTTP Client
    |
    v
Gateway  ----------------------> User / Item / Computing
  HTTP Trace + Access Log          gRPC Trace Interceptor
    |                                      |
    |                                      v
    |                                Database / Redis
    |
WebSocket Client
    |
    v
Gate Server
  WebSocket Upgrade Trace
  Message-level Trace
  In-memory connections

各微服务 <---- etcd lease / KeepAlive ----> 服务注册与发现

Trace ----> Tempo
Log   ----> Loki
Metric     尚未实现
```

Gateway 已不再维护 WebSocket 长连接，WebSocket 代码已迁移到独立的 `gate_server`。因此原文中“Gateway 同时承载 HTTP 和 WebSocket”的描述已经过时。

## 4. 原优化项复核

| 原优化项 | 当前状态 | 复核结论 |
|---|---|---|
| Gateway HTTP Trace | 已完成 | 已有统一 Middleware，支持请求开始、完成、耗时和状态日志 |
| W3C Trace Context | 已完成 | 支持 `traceparent`、`baggage`，并兼容 `x-trace-id`、`x-request-id` |
| gRPC Trace 传递 | 已完成 | 已有 Unary/Stream Client/Server Interceptor |
| Tempo 上报 | 已完成基础接入 | 各服务通过 `trace.yaml` 和环境变量配置 OTLP Endpoint |
| Trace 与日志关联 | 已完成 | JSON 日志带有 `trace_id`、`span_id`、`request_id`、`service`、`instance_id` |
| Loki 日志上报 | 已完成基础接入 | 支持批量推送，但失败反馈和重试不完整 |
| Prometheus Metrics | 未完成 | `observability_kit/metric` 目前只有说明文件 |
| WebSocket 独立服务 | 已完成 | 已从 Gateway 拆分到 `gate_server` |
| WebSocket 多实例 | 未完成 | 连接状态和路由仍在单进程内存中 |
| etcd 注册和自动摘除 | 基本完成 | 使用 lease/KeepAlive，主要 gRPC 服务支持注销和优雅停止 |
| gRPC 负载均衡 | 已完成基础能力 | manual resolver 配合 `round_robin`，但缺少调用治理策略 |
| 请求超时分层 | 未完成 | 未发现统一的 Gateway、RPC、数据库超时预算 |
| 限流、熔断、舱壁 | 未完成 | 未发现公共实现或服务级配置 |
| Kubernetes 高可用 | 未完成 | 单副本且缺少探针、PDB、HPA、资源约束和分散调度 |
| 外部入口负载均衡 | 仓库内未体现 | 未发现 Ingress 或外部 LB 配置 |
| 服务依赖治理 | 部分完成 | 服务边界存在，但没有完整依赖矩阵和自动校验 |

## 5. 可观测性分析

### 5.1 已经具备的能力

公共能力已按职责放在 `observability_kit` 中，而不是继续堆入 `servicekit`：

- `observability_kit/trace`：OpenTelemetry 初始化、HTTP 上下文传播、gRPC 拦截器；
- `observability_kit/log`：标准 JSON 日志、Trace 关联、Loki Push；
- `observability_kit/config`：先读取 `trace.yaml`，再使用非空环境变量覆盖；
- `observability_kit/metric`：预留目录，尚无 Prometheus 实现。

Gateway 已有 HTTP 请求进入和返回日志，因而不需要再保留另一套重复的 HTTP access log。应以统一结构化日志为唯一入口日志，避免重复计数和字段标准不一致。

### 5.2 Trace 的风险

当前 TracerProvider 使用 `ParentBased(AlwaysSample())`。这适合本地调试，但生产环境全量采样会增加应用 CPU、网络、Tempo 存储和查询成本。建议把采样配置加入 `trace.yaml`，至少支持比例采样，并保持上游采样决定。

更重要的是数据库自动埋点会把格式化 SQL 放入 Span。`/user/login` 包含数据库查询时，Tempo 出现 SQL 并不是 Gateway 主动记录请求体，而是 ORM Instrumentation 记录数据库语句导致的。格式化 SQL 可能包含真实参数，必须在生产环境采取以下一种或多种措施：

- 只记录参数化 SQL 模板，不记录绑定值；
- 关闭数据库 Statement 属性；
- 对账号、邮箱、手机号、Token、密码等字段进行脱敏；
- 建立属性白名单，禁止把请求体和认证信息直接写入 Span。

### 5.3 日志的可靠性风险

Loki Logger 会批量缓存日志，但定时 Flush 和达到批量阈值时的错误被直接忽略。Loki 不可达、返回非 2xx 或网络超时时，业务仍继续运行，但远端日志可能静默丢失。

建议：

- Flush 失败至少写入 stderr，并增加失败计数；
- 失败批次进行有界重试，避免无限占用内存；
- 设置缓存条数和字节数上限，明确丢弃策略；
- 进程退出时设置有限时间完成 Flush；
- Loki 标签只保留低基数字段，`trace_id`、`request_id`、用户 ID 应保留在 JSON 内容中，不能作为标签。

### 5.4 Metrics 是当前最大观测缺口

只有 Trace 和 Log 无法可靠回答“错误率是否上升、哪个实例过载、是否需要扩容”。应补充 Prometheus，优先实现 RED 指标：

- HTTP/gRPC 请求数；
- HTTP/gRPC 错误数和状态码；
- 请求耗时直方图；
- 当前活跃请求数；
- WebSocket 当前连接数、消息数、消息错误数；
- etcd 注册/发现异常数；
- Loki/Tempo 上报成功、失败和丢弃数。

指标标签不得包含 Trace ID、Request ID、连接 ID、用户 ID等高基数值。

## 6. 服务调用与雪崩风险

### 6.1 超时预算缺失

Gateway 的路由处理直接把 HTTP 请求 Context 传给 gRPC 调用，没有为单次下游调用创建显式 deadline。如果客户端没有设置超时，下游阻塞会持续占用 Gateway 连接、Goroutine 和 gRPC 资源。

建议先定义请求总预算，再逐层缩短，例如：

```text
入口总预算 3s
  -> Gateway 路由与校验 200ms
  -> gRPC 下游调用 1s
      -> 数据库操作 500ms
```

具体数值应依据 P95/P99 数据调整，而不是直接把示例值作为统一生产配置。超时必须可按接口配置，且下游超时小于上游剩余预算。

### 6.2 重试必须受控

`round_robin` 解决的是多连接选择，不等于故障重试。建议只为明确幂等的查询接口配置最多 1～2 次重试，并使用指数退避和抖动。登录、创建、扣减等可能产生副作用的操作，未实现幂等键前不能自动重试。

重试必须受原请求 deadline 约束，还要统计尝试次数，否则重试会放大故障流量。

### 6.3 限流、熔断与舱壁

当前没有发现统一治理实现。建议按调用方到下游服务的维度分别设置：

- 最大并发数和短队列；
- 超时；
- 熔断器；
- 连接池；
- 有界重试；
- 降级响应。

Gateway 入口还应提供按 IP、用户和接口的限流。过载时明确返回 `429` 或 `503`，不能无限排队。引入熔断前必须先有指标，否则无法校准阈值和判断误熔断。

## 7. etcd 服务发现分析

现有 discovery 已具备 lease、KeepAlive、注销、实例监听和 `round_robin` 客户端连接能力，也支持微服务注册后抢占式写入带租约的 KV。这些基础无需重新设计。

仍需注意：

- Gateway 当前是服务消费者，不注册自身；如果运维系统需要发现 Gateway，应明确是否由 Kubernetes Service/Ingress 管理，而不是默认要求它注册 etcd；
- Gate Server 尚未形成完整的服务注册和跨实例发现方案；
- lease 存活只能证明进程仍能续约，不能证明服务已经 ready；
- 服务启动应在监听成功且必要依赖就绪后注册，退出时应先停止接流量、注销，再完成在途请求；
- etcd 短暂不可用时，应验证已建立的 gRPC 连接是否继续工作、缓存地址如何过期以及恢复后能否重新 Watch。

## 8. Gateway 与 Gate Server 边界

当前拆分方向正确：

- Gateway 负责短连接 HTTP、认证、路由和协议转换；
- Gate Server 专门负责 WebSocket Upgrade、连接和消息生命周期；
- Gateway 不应重新引入 WebSocket 连接表或消息循环。

但 Gate Server 的 `Manager` 使用本地 `map` 保存连接。多副本后，任一实例只能找到连接在本实例上的客户端，因此“扩成三个副本”并不能自动支持服务端向任意用户推送。

多实例化至少需要：

1. 用 Redis 等共享存储维护 `user/session -> gate instance` 路由；
2. 使用 Redis Streams、NATS 或可靠消息队列把消息投递到连接所在实例；
3. 实例退出和异常断开时清理路由，使用 TTL 防止脏状态永久存在；
4. 客户端支持断线重连和会话恢复；
5. 增加 Ping/Pong、空闲超时、最大连接数、单用户连接数和单连接消息速率限制；
6. 发布期间先停止接收新连接，给现有连接发送重连提示，再在宽限期后退出。

在这些能力完成前，Gate Server 可以保持单副本，或通过 Sticky Session 作为临时方案，但必须接受实例故障会断开全部本机连接的事实。

## 9. Kubernetes 部署分析

Gateway、Gate Server、User、Item、Computing、Server Manager 的基础 Deployment 当前均为 `replicas: 1`，且未配置以下生产必需项：

- `readinessProbe`、`livenessProbe`、`startupProbe`；
- CPU/内存 `requests` 和 `limits`；
- `PodDisruptionBudget`；
- HPA；
- `topologySpreadConstraints` 或 Pod Anti-Affinity；
- 明确的 RollingUpdate 参数；
- Ingress/外部负载均衡。

不应直接把所有服务副本统一改成 3。应先完成 readiness、优雅退出和无状态验证，否则多个“不健康但仍接流量”的实例只会扩大问题。建议顺序为：

1. 暴露轻量的 startup/liveness/readiness 端点；
2. 配置资源 requests/limits，并通过压测校准；
3. 验证注销和优雅停止；
4. 对无状态 HTTP/gRPC 服务增加副本、PDB 和分散调度；
5. 接入 Metrics 后配置 HPA；
6. Gate Server 完成多实例路由后再横向扩容。

Liveness 只判断进程是否失去工作能力，不能因为数据库或 etcd 短暂不可用就重启所有 Pod。下游依赖状态应主要影响 Readiness，并结合业务是否可以降级决定。

## 10. 服务边界与依赖治理

当前没有证据表明必须进行全量服务重构，但依赖关系需要显式化。建议建立版本化依赖矩阵，至少包含：

| 调用方 | 被调用方 | RPC 方法 | 超时 | 幂等 | 允许重试 | 降级方式 | Owner |
|---|---|---|---|---|---|---|---|
| Gateway | User | Login | 待压测 | 否 | 否 | 返回明确错误 | 待填写 |
| Gateway | Item | 查询类接口 | 待压测 | 是 | 有界 | 缓存或失败 | 待填写 |

还应通过代码评审或 CI 保证：

- 禁止跨服务直接访问数据库；
- 禁止循环依赖；
- 同步调用链原则上不超过 3～4 层；
- 通知、统计、审计、批处理等非实时工作优先异步化；
- Server Manager 当前 RPC 业务能力较少，应先明确职责和使用方，避免为了“微服务完整”而增加空服务。

## 11. 配置与安全

各服务的 `trace.yaml` 支持文件配置和环境变量覆盖，适合本地开发，但部署时应注意：

- 配置文件中的 `127.0.0.1` 只在采集器与进程位于同一网络命名空间时成立；Kubernetes 中通常应使用 Service DNS；
- 敏感配置应来自 Secret，不应提交到普通 ConfigMap 或仓库；
- 配置加载应在启动日志中输出“最终生效的非敏感配置”，便于排查环境变量覆盖；
- OTLP gRPC/HTTP 协议与端口必须匹配，4317 通常是 OTLP gRPC，4318 通常是 OTLP HTTP；
- 日志、Span 和错误消息都不能记录密码、Token、Cookie、Authorization 和完整请求体。

## 12. 实施优先级

### P0：先降低直接生产风险

1. 为 Gateway 和服务间调用建立显式超时预算；
2. 禁止数据库 Span 上报格式化参数，完成敏感字段脱敏；
3. 增加 readiness/liveness/startup，校验优雅退出顺序；
4. 让 Loki Flush 失败可见，并增加有界缓存和丢弃计数；
5. 将生产 Trace 从 AlwaysSample 改成可配置比例采样。

### P1：建立治理数据和基础容错

1. 实现 `observability_kit/metric` 和 Prometheus Endpoint；
2. 为 HTTP/gRPC、数据库、etcd、Loki/Tempo、WebSocket 建立核心指标；
3. 增加入口限流和下游并发隔离；
4. 只对幂等 RPC 增加有界重试；
5. 对无状态服务增加多副本、PDB、资源配置和分散调度。

### P2：解决多实例和自动扩缩容

1. Gate Server 连接路由外置，建立跨实例消息通道；
2. 完成 WebSocket 心跳、连接配额、限速和排空；
3. 增加 Ingress 或云负载均衡；
4. 基于 CPU、内存、请求延迟、并发或队列长度配置 HPA；
5. 建立服务级熔断和降级策略。

### P3：长期架构治理

1. 维护服务依赖矩阵并在 CI 中检查；
2. 将非实时流程事件化；
3. 建立 SLO、告警、容量模型和故障演练；
4. 根据实际调用关系判断是否合并过细服务或拆分过载领域，而不是预先全量重构。

## 13. 可验收清单

完成以下检查后，才能认为本轮优化形成闭环：

- [ ] 任意 HTTP 请求可通过 `trace_id` 关联 Gateway、gRPC 服务、数据库 Span 和 Loki 日志；
- [ ] Tempo 中不出现密码、Token、Cookie、Authorization 和带真实敏感参数的 SQL；
- [ ] Loki/Tempo 不可用时业务不会被无限阻塞，同时存在明确失败与丢弃指标；
- [ ] 所有外部请求和服务间调用都有可验证的 deadline；
- [ ] 自动重试仅覆盖已声明幂等的方法，且总耗时不超过原 deadline；
- [ ] Prometheus 能按服务和实例查看 QPS、错误率、P95/P99 和活跃请求；
- [ ] 单个无状态服务实例退出后，新请求不会继续发送到该实例，在途请求能够完成或明确失败；
- [ ] Pod 缺少依赖时 Readiness 失败，但不会因短暂下游故障触发全体 Liveness 重启；
- [ ] 核心无状态服务具备多副本、PDB、资源约束和跨节点分散能力；
- [ ] Gate Server 多副本场景下可以把消息准确投递到连接所在实例；
- [ ] etcd、Tempo、Loki、数据库短暂不可用及流量突增均完成故障演练；
- [ ] 服务依赖不存在循环，跨服务数据库访问已被禁止。

## 14. 最终判断

原方案的方向仍然成立，但优先级应根据当前实现调整：全链路 Trace、Loki 日志和 WebSocket 服务拆分已经不再是“待实现项”；下一阶段应集中处理超时与隔离、Metrics、观测数据安全、Kubernetes 高可用和 Gate Server 多实例化。

因此建议继续渐进式演进，不做全量重构。先用 P0/P1 项建立可测量、可止损的运行基础，再依据真实指标决定副本数、熔断阈值、采样比例和服务边界调整。
