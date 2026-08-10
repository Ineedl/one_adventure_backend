# etcd service discovery

所有实例使用 servicekit 注册到 etcd：
`/one_adventure/<server_name>/<instance_id>`。值是 JSON：

```json
{"address":"127.0.0.1","grpc_port":"8083","http_port":"8000"}
```

注册使用 etcd lease 和 KeepAlive，进程退出或租约失效后实例自动删除。
消费者先读取关注前缀的完整快照，再从快照 revision 开始 watch。Gateway
watch `/one_adventure/`，微服务按 `watchServices` 配置 watch 指定服务。

servicekit/discovery 维护每个服务名的 gRPC manual resolver，并将当前所有
实例地址推送给 gRPC `round_robin`。生成 client 的工厂表也位于该包中，
服务名到 protobuf client 的映射集中维护。
