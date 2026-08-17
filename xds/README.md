# xDS 控制面

xDS 作为宿主机上的独立进程运行，不由 Docker Compose 管理。它读取 etcd 注册数据，并向 Envoy 提供 CDS/EDS 配置。

```bash
docker compose -f one_adventure/docker-compose.yml up -d etcd
cd xds && go run ./cmd/xds
docker compose -f one_adventure/docker-compose.yml up -d envoy
```

xDS 默认监听 `0.0.0.0:18000`，Envoy 通过 `host.docker.internal:18000` 连接宿主机上的 xDS。

默认配置文件为 `xds/config.yaml`，可通过 `XDS_CONFIG` 指定其他路径，`ETCD_ENDPOINTS` 可覆盖配置文件中的 etcd 地址。
