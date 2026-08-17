# 微服务间 Envoy + etcd 架构

## 1. 架构目标

使用 etcd 保存微服务实例注册信息，由 xDS Control Plane 将 etcd 中的实例信息转换为 Envoy 的服务发现和负载均衡配置，再由 Envoy 将请求转发到目标微服务实例。

核心链路如下：

```text
etcd
  |
  | 服务实例注册、租约、Watch
  v
xDS Control Plane
  |
  | CDS：服务集群配置
  | EDS：实例端点配置
  | LDS/RDS：监听器和路由配置（可选）
  v
Envoy
  |
  | 负载均衡、健康检查、超时、熔断、重试、TLS
  v
目标微服务实例
```

## 2. 总体架构图

```text
                                      +----------------------+
                                      |      Prometheus       |
                                      +----------^-----------+
                                                 |
                                      metrics / health
                                                 |
+-------------+       Watch       +-------------+--------------+
|             | -----------------> |                            |
|    etcd     |                    |      xDS Control Plane      |
|             | <----------------- |                            |
+------+------+   lease/status    +-------------+--------------+
       |                                         |
       | service registration                    | xDS gRPC stream
       |                                         | CDS / EDS / LDS / RDS
       v                                         v
+------+-----------------------------------------+------+
|                                                       |
|                    Envoy Proxy                        |
|                                                       |
|  Listener -> Route -> Cluster -> Endpoint             |
|  timeout / retry / circuit breaker / load balancing   |
|                                                       |
+------+----------------------+-------------------------+
       |                      |
       | gRPC/HTTP            | gRPC/HTTP
       v                      v
+------+-------+       +------+-------+       +----------+
| User-1       |       | User-2       |  ...  | Item-1   |
| 10.0.1.10    |       | 10.0.1.11    |       | 10.0.2.10|
+--------------+       +--------------+       +----------+
```

## 3. 在 Gateway 和 Gate Server 中的部署方式

推荐先为 Gateway、Gate Server 和核心微服务部署 Envoy sidecar：

```text
+---------------------- Pod ----------------------+
|                                                  |
|  Gateway / Gate / Business Service               |
|        |                                         |
|        | localhost gRPC/HTTP                     |
|        v                                         |
|      Envoy Sidecar                               |
|        |                                         |
|        | xDS 配置、负载均衡和转发                 |
|        v                                         |
|      其他服务的 Envoy                             |
|        |                                         |
|        v                                         |
|      目标业务进程                                |
|                                                  |
+--------------------------------------------------+
                         ^
                         |
                  xDS Control Plane
                         ^
                         |
                        etcd
```

应用只连接本地 Envoy，例如：

```text
Gateway -> localhost:15001 -> Envoy -> User Service
Gate    -> localhost:15001 -> Envoy -> Item Service
```

这样可以把连接池、负载均衡、超时、重试、熔断和 TLS 等能力从业务代码中统一抽离。

## 4. etcd 数据模型

建议使用稳定、可版本化的 Key 前缀：

```text
/one_adventure/services/user/instances/user-1
/one_adventure/services/user/instances/user-2
/one_adventure/services/item/instances/item-1
```

Value 示例：

```json
{
  "service": "user",
  "instance_id": "user-1",
  "address": "10.0.1.10",
  "port": 8083,
  "protocol": "grpc",
  "version": "v1",
  "weight": 100,
  "zone": "az-a",
  "ready": true
}
```

每个实例必须绑定 etcd lease。实例退出、进程异常或租约过期后，Key 自动删除，Control Plane 通过 Watch 感知变化。

## 5. xDS 配置职责

### CDS：Cluster Discovery Service

CDS 描述逻辑服务集群，例如：

```text
user-grpc
item-grpc
computing-grpc
```

CDS 主要配置：

- 集群名称；
- 使用 EDS 获取实例端点；
- 连接超时；
- HTTP/2/gRPC 协议；
- 熔断阈值；
- 连接池限制。

### EDS：Endpoint Discovery Service

EDS 将 etcd 中的实例转换为 Envoy 可使用的端点：

```text
user-grpc:
  - 10.0.1.10:8083 weight=100 zone=az-a
  - 10.0.1.11:8083 weight=100 zone=az-b
```

实例新增、删除、Ready 状态或权重变化时，Control Plane 推送新的 EDS 版本。

### LDS/RDS：监听器和路由

LDS 和 RDS 是可选能力：

- LDS 配置 Envoy 的监听端口和 Filter Chain；
- RDS 配置 HTTP/gRPC 路由、域名和路径匹配。

如果只使用 Envoy sidecar 做服务间 gRPC 代理，可以先实现 CDS + EDS，暂时不实现 LDS/RDS 动态下发。

## 6. 请求调用流程

```text
1. Gateway 调用 user 服务
2. 请求发送到本地 Envoy
3. Envoy 根据 CDS 找到 user-grpc 集群
4. Envoy 根据 EDS 获取 user 实例列表
5. Envoy 执行健康检查和负载均衡
6. Envoy 将请求转发给 User-1 或 User-2
7. 返回响应，并记录访问日志、指标和 Trace
```

请求路径：

```text
Gateway -> Gateway Envoy -> User Envoy -> User Application
```

若使用 Kubernetes，也可以让 Envoy 直接把 EDS 端点指向目标 Pod IP，而不是再经过 Kubernetes Service。

## 7. Control Plane 工作流程

```text
实例启动
   |
   v
注册 etcd + Lease
   |
   v
Control Plane Watch etcd
   |
   v
更新内部服务快照和版本号
   |
   v
生成 CDS/EDS 资源
   |
   v
通过 xDS gRPC Stream 推送 Envoy
   |
   v
Envoy 原子切换端点
```

Control Plane 应满足：

- 支持多实例部署；
- 保证配置版本单调递增；
- 支持 Envoy 断线重连和全量快照；
- etcd Watch 中断后能重新获取完整快照；
- 不向 Envoy 推送未 Ready 的实例；
- 保留最近版本并支持配置回滚；
- 对 xDS 推送失败和资源拒绝提供指标和日志。

## 8. 负载均衡和健康检查

推荐默认使用：

```text
普通 gRPC：round_robin
请求耗时差异大：least_request
实例规格不同：weighted_round_robin
需要分片：consistent_hash（谨慎使用）
```

Envoy 层应配置：

- 连接超时；
- 请求超时；
- HTTP/2 Keepalive；
- 主动健康检查；
- 最大连接数；
- 最大并发请求数；
- 每个集群的熔断阈值；
- 有界重试和重试预算。

应用侧和 Envoy 侧不要重复配置互相冲突的重试策略。建议只保留一层负责自动重试，并由原始 Deadline 限制总耗时。

## 9. 故障处理流程

### 实例故障

```text
实例无法续租或健康检查失败
  -> etcd Key 删除或标记 NotReady
  -> Control Plane 更新 EDS
  -> Envoy 停止向该实例发送新请求
  -> 其他健康实例继续接收请求
```

### etcd 暂时不可用

```text
etcd Watch 中断
  -> Control Plane 保留最后一次有效快照
  -> Envoy 继续使用当前端点
  -> Control Plane 重连 etcd
  -> 读取完整快照并恢复 Watch
```

不能因为 etcd 短暂不可用就立即清空所有 EDS 端点，否则可能造成全链路不可用。

### Control Plane 故障

Envoy 应继续使用最后一次有效的 xDS 配置。Control Plane 需要多副本部署，并使用 Kubernetes Service 或内部负载均衡器提供 xDS 地址。

## 10. 安全和可观测性

### 安全

- Envoy 与 Control Plane 的 xDS 通道使用 mTLS；
- Envoy 与业务服务之间根据安全等级启用 mTLS；
- etcd 开启 TLS、认证和最小权限；
- Control Plane 只允许读取服务注册前缀；
- 不在 xDS 元数据中放置密码、Token 和业务敏感数据。

### 指标

至少采集：

- Envoy 请求总数、错误率和 P95/P99 延迟；
- 每个 Cluster 和 Endpoint 的请求量；
- 健康检查失败数；
- 熔断、重试和连接池拒绝数；
- xDS 推送成功、失败和拒绝数；
- etcd Watch、Lease 和重新同步异常数。

## 11. 推荐技术选型

| 能力 | 推荐技术 |
|---|---|
| 服务注册 | etcd v3 Lease + Watch |
| xDS Control Plane | Go 自研控制面，或基于 go-control-plane |
| 数据面代理 | Envoy |
| 配置协议 | ADS/xDS，核心使用 CDS + EDS |
| Gateway 入口 | Envoy Gateway |
| Gate Server 入口 | Envoy Gateway 或云 L4 Load Balancer |
| 指标 | Prometheus + Grafana |
| Trace | OpenTelemetry + Tempo |
| 日志 | Loki 或集中式日志系统 |
| Gate 连接路由 | Redis |
| Gate 消息投递 | NATS JetStream 或 Redis Streams |

## 12. 推荐部署顺序

### 第一阶段：实现最小闭环

1. 规范 etcd 服务注册 Key 和 Value；
2. 实现 Control Plane 的 etcd Watch；
3. 实现 CDS + EDS；
4. 部署一个 Envoy sidecar；
5. 验证 User 服务实例新增、删除和故障摘除；
6. 验证 gRPC 超时、健康检查和 `round_robin`。

### 第二阶段：生产治理

1. Control Plane 多副本；
2. xDS mTLS；
3. Envoy 熔断、连接池和有限重试；
4. Prometheus 指标和告警；
5. 配置版本、回滚和故障演练；
6. 覆盖 Gateway、Gate Server 和核心微服务。

### 第三阶段：统一入口和服务网格

当服务数量和治理需求进一步增长时，再引入 Envoy Gateway 或 Istio 统一管理入口和 sidecar。不要在应用、Envoy 和服务网格中重复配置多套重试、熔断规则。

## 13. 结论

推荐架构为：

```text
etcd
  -> xDS Control Plane
  -> Envoy CDS/EDS
  -> 目标微服务实例
```

该方案可以保留当前 etcd 服务注册能力，同时将实例发现、负载均衡、健康检查、超时、熔断和安全策略集中到 Envoy 数据面。实施时应优先完成 CDS + EDS 的最小闭环，再逐步增加 LDS/RDS、mTLS、灰度和服务网格能力。
