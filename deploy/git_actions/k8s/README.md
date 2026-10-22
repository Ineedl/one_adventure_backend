# GitHub Actions + Kubernetes

## Kubernetes 服务范围

## GitHub Actions CI/CD

`.github/workflows/k8s-deploy.yml` 使用已运行的 self-hosted runner 执行完整链路：
checkout 所选分支的当前提交、构建指定服务、推送到 `15342537816/one_adventure_backend`，再通过
kubectl 部署 Kubernetes。runner 工作目录由 GitHub Actions 设置为 `GITHUB_WORKSPACE`，
`deploy-runner.sh` 会从该 checkout 调用独立副本 `deploy-actions.sh`（原 `deploy.sh` 保留），不使用 runner 安装目录中的旧源码。

在仓库 Settings -> Secrets and variables -> Actions 中配置：

- Secret `REGISTRY_USERNAME`、`REGISTRY_PASSWORD`：Docker Hub 登录凭据
- Secret `MYSQL_ROOT_PASSWORD`、`REDIS_PASSWORD`：可选；不配置时首次部署自动生成并复用集群 Secret
- Variable `KUBE_NAMESPACE`：可选，默认 `one-adventure`

工作流仅支持手动触发；提交或推送代码不会自动部署。

手动发布：Actions → Build and Deploy to Kubernetes → Run workflow，选择代码分支后填写：

- `service`：选择单个服务（如 `user`）或 `all`。
- `image_tag`：必填版本号，如 `0.0.2`。
- `deploy_only`：默认不勾选，执行编译、镜像构建、推送和部署；勾选后直接部署仓库中指定版本的已有镜像。

仅部署模式仍会应用所选分支中的 Kubernetes 清单和服务配置，保留静态 PV 和部署超时逻辑。
该模式跳过 Go/Compose 构建工具检查及 GoFrame CLI 安装，仍登录 Docker Hub 以配置 Kubernetes 私有镜像拉取凭据。
所选服务（或 `all` 对应全部服务）的指定版本镜像必须已存在。

xds 与其他服务统一使用 `image_tag` 版本号。

例如 `service=user`、`image_tag=0.0.2` 只编译当前 checkout 的 user，构建并推送
`15342537816/one_adventure_backend:user-0.0.2`，更新 user 配置并等待其部署完成。
单服务发布要求集群已完成首次全量部署，复用 `middleware-auth` 中的密码。
选择 `all` 则发布全部 11 个应用并应用中间件清单。重复使用相同版本号也会重新拉取镜像并更新 Pod。
同一集群的发布串行执行，防止单服务与全量发布相互覆盖。
runner 必须已配置 Docker、kubectl、Go、Python 3/PyYAML，并且 kubeconfig 当前身份有目标集群部署权限。

部署脚本发布全部 11 个应用镜像对应的服务：gateway、gate-server、computing、
user、item、commerce、ws-gateway、order、pay、promotion、xds，以及 etcd、Kafka、Envoy、
MySQL 8.0、Redis 7.4、Loki、Tempo 和 Grafana。gateway 默认两个副本，其余各一个，稳定后预期有 20 个 Pod。

无需手动输入密码。首次部署会为 MySQL 和 Redis 分别自动生成随机默认密码，保存到
集群的 `middleware-auth` Secret；后续部署自动复用已有密码。执行账号需有读取该 Secret
的权限。需要自定义时可通过环境变量覆盖（重复部署时使用相同密码）：

```sh
export MYSQL_ROOT_PASSWORD=your_mysql_password
export REDIS_PASSWORD=your_redis_password
```

MySQL 密码支持字母、数字、下划线、点和短横线，以兼容应用的 GoFrame DSN 格式。
脚本需要 Python 3 和 PyYAML（`python3 -m pip install PyYAML`）。

```sh
bash deploy/git_actions/k8s/deploy.sh 15342537816/one_adventure_backend 0.0.1 --deploy-only
kubectl get deployments,statefulsets,pods -n one-adventure
```

`--deploy-only` 跳过编译、镜像构建和推送。所有对应版本镜像必须已存在。
此模式需要 kubectl、Python 3 和 PyYAML，无需 Docker 和 gf。

当前 xds 镜像使用绝对路径启动 `/app/main`，并基于 Alpine 3.20 构建。重新构建后建议使用
新的 xds 版本标签；其他应用仍使用 `0.0.1` 时，部署需指定：

```sh
bash deploy/git_actions/k8s/deploy.sh 15342537816/one_adventure_backend 0.0.1 \
  --deploy-only --xds-tag 0.0.2
```

`--xds-tag`（或环境变量 `XDS_IMAGE_TAG`）单独指定 xds 版本，默认使用公共
IMAGE_TAG；完整构建发布时，xds 的推送标签也使用该版本。
不要省略这个覆盖参数后再次部署旧的 `0.0.1`，否则会覆盖集群中的 xds 修复版本。

可以覆盖推送已有的 `xds-0.0.1` 标签。xds Deployment 已设置
`imagePullPolicy: Always`，覆盖推送后执行滚动重启即可拉取新镜像：

```sh
docker tag one-adventure/xds:latest 15342537816/one_adventure_backend:xds-0.0.1
docker push 15342537816/one_adventure_backend:xds-0.0.1
kubectl -n one-adventure rollout restart deployment/xds
kubectl -n one-adventure rollout status deployment/xds --timeout=5m
```

也可以重新执行部署脚本并指定 `--xds-tag 0.0.1`；相同镜像标签不会自动触发更新，
因此覆盖后仍建议执行 `rollout restart`。

Deployment 和 StatefulSet 会自动补齐期望副本，直接删除失败 Pod 会使其重建。
xds 滚动更新卡住时可能同时保留新旧 Pod，应修正 Deployment 的镜像版本。
副本数为 0 的旧 ReplicaSet 是回滚历史，不代表正在运行的 Pod。
Kafka、MySQL、Redis 和 etcd 使用单副本 StatefulSet，并有各自的 headless governing Service；
Envoy 使用 Deployment，通过 xDS 获取动态配置。Kafka、MySQL 和 Redis 客户端仍通过原有
`kafka`、`mysql`、`redis` Service 访问。迁移时部署脚本会删除旧 workload，再应用对应的新类型，期间服务会短暂中断。
Kafka、MySQL 和 Redis 分别使用 `kafka-data`、`mysql-data`、`redis-data` PVC，已有数据卷会保留。
旧版 Kafka 曾使用 `emptyDir` 的数据不会迁移；etcd 目前仍使用临时存储，首次替换旧
etcd Deployment 会丢失其 Pod 内数据，应用需重新向 etcd 注册服务。Envoy 从旧 StatefulSet
改回 Deployment 时会移除旧 headless Service，Envoy 动态配置会在 Pod 启动后从 xDS 重新获取。
Redis 从旧 Deployment 迁移时会保留 `redis-data` PVC。

应用配置从 `<CONFIG_ROOT>/<服务源码目录>/manifest/config-docker` 生成 Secret，
保留 config.yaml、rpc.yaml、trace.yaml 等完整文件；CONFIG_ROOT 默认仓库根目录。
脚本在临时目录中将 config.yaml 的 MySQL 地址改为 `mysql:3306`，统一使用 root
和 MYSQL_ROOT_PASSWORD，将 Redis 地址改为 `redis:6379` 并设置 REDIS_PASSWORD。
源配置文件不会被修改，其他配置文件原样复制。
Secret 用于避免额外将数据库密码复制到部署清单，但不代表集群已启用静态加密。
脚本为应用 Pod 模板写入配置校验值，配置修改后再次部署会自动滚动更新对应应用。

MySQL、Redis 使用 namespace 内的 ClusterIP Service，不开放节点端口。Kubernetes
发布不会再注入 Docker 专用的 host.docker.internal 主机别名，也不会把 Pod 固定到
某个节点。配置中的集群内依赖必须使用 Kubernetes Service 名称；集群外依赖必须使用
稳定的 DNS、VIP 或其他可路由地址。渲染配置时如果仍发现 host.docker.internal，部署会
直接失败，避免首次部署后服务使用错误地址。

私有镜像可通过 `--docker-config` 导入当前用户的 Docker 登录凭据：

```sh
bash deploy/git_actions/k8s/deploy.sh 15342537816/one_adventure_backend 0.0.1 \
  --deploy-only --docker-config "$HOME/.docker/config.json"
```

也支持环境变量 `DOCKER_CONFIG_FILE`。脚本在目标 namespace 创建或更新
`dockerhub-auth` Secret，并为全部 11 个应用 Deployment 设置 imagePullSecrets。
已有 Deployment 首次添加引用时会自动滚动更新 Pod，保留其他镜像拉取 Secret。
文件必须包含实际 `auths` 凭据；仅有 `credsStore`/`credHelpers` 的文件不能直接用于集群拉取。
此参数仅配置 Kubernetes 拉取认证；完整发布的 Docker 推送仍使用当前用户的登录状态。
全量部署会创建 Loki、Tempo 和 Grafana；Compose 的 Nginx 入口和 Prometheus 尚未迁移。
trace.yaml 中的 tempo:4317、loki:3100 由同 namespace 的 Service 提供。
etcd 已使用 StatefulSet 获取稳定 Pod 身份，但仍使用临时数据目录；Pod 替换会重置 etcd
数据，生产使用前应添加持久卷。

## Kafka、MySQL 与 Redis

`middleware.yaml` 为 Kafka、MySQL 和 Redis 分别创建 10Gi、10Gi、1Gi 的 PVC；
Redis 开启 AOF。集群需有默认 StorageClass，或预先提供匹配的 PV。
若出现 `pod has unbound immediate PersistentVolumeClaims`，表示 PVC 尚未绑定存储，
可通过 `kubectl get storageclass`、`kubectl describe pvc mysql-data -n one-adventure` 查看原因。
没有动态存储的单节点集群可直接重新部署并添加 `--local-storage`：

```sh
bash deploy/git_actions/k8s/deploy.sh 15342537816/one_adventure_backend 0.0.1 \
  --deploy-only --xds-tag 0.0.2 --local-storage
```

该参数为尚未绑定的 Kafka/MySQL/Redis/Loki/Tempo/Grafana PVC 创建静态 hostPath PV，数据目录为节点上的
`/var/lib/one-adventure/<namespace>/<组件名>-data`，并设置节点亲和性。
已绑定的 PVC 保持原存储，现有 PVC 无需删除；多节点集群必须通过 STORAGE_NODE_NAME
指定存储节点。此模式需要创建集群级 PV 的权限，节点磁盘需有足够空间，PVC 容量声明
不会限制 hostPath 实际用量。节点故障时数据不会自动迁移。
PV 使用 Retain 策略。普通清理保留 PVC 和 PV；`cleanup.sh --delete-data` 会删除 PVC 及经过归属校验的项目静态 PV，节点数据目录仍保留。部署脚本可在归属和存储配置一致时复用同名 PVC 的保留卷。
Kafka Pod 的初始化容器会将数据目录属主设为运行 Kafka 的 UID 1000；它不会更改目录内已有文件的属主。

Kafka、MySQL 和 Redis 各运行一个 StatefulSet 副本，使用稳定身份和 PVC；StatefulSet 默认按序更新，
避免新旧实例同时写入同一个数据卷。
MySQL 8 首次初始化可能需要几十秒，启动探针会等待 30 秒后使用 `mysqladmin ping`
检查服务，最多允许 10 分钟完成初始化；启动期间出现少量 `ERROR 2003` 探针事件属于正常现象。
Actions 部署脚本应用中间件清单后会等待 PVC 绑定，并等待 MySQL、Redis StatefulSet
健康检查通过后再提交应用清单。

只需重新应用 Kafka/MySQL/Redis 配置并重建这些 Pod 时，可在仓库根目录运行：

```sh
bash deploy/git_actions/k8s/recreate-middleware.sh
```

脚本默认使用单节点静态 hostPath PV，仅应用 Kafka/MySQL/Redis 相关资源并保留 PVC；
`--no-local-storage` 可用于已有 StorageClass 或预先创建 PV 的集群。旧 Kafka `emptyDir`
数据不会自动迁移到新 PVC。

密码存储在 `middleware-auth` Secret。MySQL 首次启动自动创建 `test`、
`one_adventure_user`、`one_adventure_item`、`one_adventure_commerce`、
`one_adventure_server` 五个数据库。初始化仅在空数据卷执行，不包含业务表结构或旧数据，
需自行导入；CONFIG_ROOT 指向的自定义配置若使用其他库名，也需自行创建。

已有 MySQL 数据卷不会因修改 MYSQL_ROOT_PASSWORD 自动更新账号密码，密码变更需先在
MySQL 内执行对应的 ALTER USER，再同步部署配置。Redis 密码变更后需手动重启 Redis
（`kubectl rollout restart statefulset/redis -n one-adventure`），使其加载新 Secret；
应用配置变更由脚本触发滚动更新。删除 StatefulSet 会保留 PVC，删除 PVC 可能丢失数据。

```sh
kubectl get deployments,statefulsets,pods,svc,pvc -n one-adventure
kubectl logs statefulset/mysql -n one-adventure
kubectl logs statefulset/redis -n one-adventure
```

## 本地端口转发

运行脚本后，MySQL 和 Redis 会临时暴露到本机回环地址；脚本保持前台运行，按 `Ctrl-C`
会自动结束两个转发进程，不创建或修改 Kubernetes 资源：

```sh
bash deploy/git_actions/k8s/port-forward-middleware.sh
```

默认连接地址：MySQL `127.0.0.1:13306`，Redis `127.0.0.1:16379`。可自定义本地端口：

```sh
bash deploy/git_actions/k8s/port-forward-middleware.sh \
  --mysql-port 3307 --redis-port 6380
```

也可以使用 `MYSQL_LOCAL_PORT`、`REDIS_LOCAL_PORT` 和 `KUBE_NAMESPACE` 环境变量。
脚本要求目标 namespace 中已存在 `mysql` 和 `redis` Service，并使用当前 kubectl context。

## 清理部署

清理全部 11 个应用及 etcd、Kafka、Envoy、MySQL、Redis 的 Deployment/StatefulSet、Service，
同时删除应用配置 Secret、镜像认证 Secret 和项目 ConfigMap。工作负载下属的 Pod 随之删除。
执行时不询问确认。

```sh
# 预览当前 kubectl context 下将删除的资源
bash deploy/git_actions/k8s/cleanup.sh --dry-run

# 清理默认 one-adventure namespace 中的项目服务
bash deploy/git_actions/k8s/cleanup.sh

# 清理自定义部署 namespace（也支持 KUBE_NAMESPACE 环境变量）
bash deploy/git_actions/k8s/cleanup.sh my-one-adventure

# 预览包含 Kafka/MySQL/Redis 数据卷的清理范围
bash deploy/git_actions/k8s/cleanup.sh --delete-data --dry-run

# 同时删除 Kafka/MySQL/Redis 数据卷及密码 Secret
bash deploy/git_actions/k8s/cleanup.sh --delete-data
```

默认保留 namespace、Kafka/MySQL/Redis PVC 和 `middleware-auth` Secret，重新部署时复用已有
数据和密码。指定 `--delete-data` 后，先清理服务，再删除 `kafka-data`、`mysql-data`、`redis-data` PVC
及 `middleware-auth` Secret；下次部署会重新生成密码。PVC 的删除可能永久清除数据，
项目静态 PV（`<namespace>-kafka-data`、`<namespace>-mysql-data`、`<namespace>-redis-data`）通过归属和目录校验后也会删除；其他 PV 按 reclaimPolicy 回收。节点 hostPath 文件不会删除，旧 MySQL 数据仍需原密码，不能靠删除 PV 重置数据库。
namespace 始终保留。etcd 当前使用临时存储，清理 Pod 会丢失其中数据。
脚本仅允许清理带有 `app.kubernetes.io/part-of=one-adventure` 标签的项目 namespace，
该标签由部署清单自动设置。namespace 或资源不存在时跳过，支持重复执行。
服务清理与可选的数据卷清理各最多等待 5 分钟；超时会报错，不会强制移除 finalizer。

## Runner 与完整发布

Runner 路径为 `/usr/local/share/application/actions-runner`，配置示例见
`runner.env.example` 和 `actions.runner.service`。完整发布需要 Docker Compose、
Go、gf、kubectl 以及可访问集群的 kubeconfig，使用执行脚本的同一用户登录 Docker Hub。

```sh
bash deploy/git_actions/k8s/deploy.sh 15342537816/one_adventure_backend 0.0.1
```

也支持环境变量 IMAGE_REPOSITORY、IMAGE_TAG、KUBE_NAMESPACE。
镜像共享仓库，以 gateway-0.0.1、user-0.0.1 等标签区分。

### Actions 部署超时与静态 PV

Actions 默认传入 `--local-storage`，为尚未绑定的 Kafka/MySQL/Redis/Loki/Tempo/Grafana PVC 创建静态本地 PV，
数据位于存储节点的 `/var/lib/one-adventure/<namespace>/<组件名>-data`，回收策略为 Retain。
已有绑定卷继续保留。默认适用于单节点；多节点需在仓库 Actions Variables 设置 `STORAGE_NODE_NAME`。

每个 PVC 最多等待 120 秒绑定，每个 Deployment 最多等待 300 秒就绪，失败后输出诊断并退出。
可在 Actions Variables 设置 `PVC_TIMEOUT_SECONDS` 和 `ROLLOUT_TIMEOUT_SECONDS`（正整数秒）。
包含构建、推送和部署的步骤最多运行 45 分钟，整个 job 最多运行 60 分钟。
静态卷文件保存在指定节点，节点必须允许创建上述数据目录。

### Gateway HTTP 外部访问

Gateway Service 使用持久的 NodePort：`http://<Kubernetes节点IP>:30901`，
转发到 gateway Pod 的 HTTP 端口 `8901`。由清单 `stack.yaml` 管理，Pod 重建或再次部署后仍然生效。
GitHub Actions 手动选择 `gateway` 或 `all` 部署即可应用此配置。
集群内部仍使用 `gateway:8901`。节点防火墙或云安全组需允许客户端访问 TCP 30901。

```sh
kubectl -n one-adventure get svc gateway
kubectl get nodes -o wide
```

Actions 将镜像、配置校验值、拉取凭据及本次部署标记一起写入 Pod 模板，单次 apply，
不会在首次创建后追加 patch/restart 触发多次更新。同版本重新部署仍会更新 Pod。
首次创建按副本数启动：gateway 为 2，其余应用为 1；后续滚动更新期间可能临时出现额外 Pod。

## Envoy 查看与动态配置清理

```sh
# 查看静态 ConfigMap、就绪状态和运行时集群
bash deploy/git_actions/k8s/query-envoy.sh

# 查看 Envoy Admin API 的完整动态配置、端点、监听器或统计信息
bash deploy/git_actions/k8s/query-envoy.sh config
bash deploy/git_actions/k8s/query-envoy.sh endpoints
bash deploy/git_actions/k8s/query-envoy.sh listeners
bash deploy/git_actions/k8s/query-envoy.sh stats

# 只查看部署使用的静态 envoy.yaml
bash deploy/git_actions/k8s/query-envoy.sh configmap

# 预览后重启 Envoy Pod，清除进程内动态 CDS/EDS 状态
bash deploy/git_actions/k8s/cleanup-envoy.sh --dry-run
bash deploy/git_actions/k8s/cleanup-envoy.sh
```

查询脚本会临时将 Admin API 转发到本机 `127.0.0.1:19901`，退出时自动清理转发进程。
清理脚本通过 rollout restart 重建 Envoy Deployment Pod；保留 `envoy-config` ConfigMap 和 etcd 服务注册，
xDS 会在 Envoy 重连后重新下发动态配置。两脚本默认使用 namespace `one-adventure`，
也可设置 `KUBE_NAMESPACE` 覆盖。

## Loki、Tempo 与 Grafana

选择 Actions 的 service=all 时部署这些组件；deploy_only 也会应用这些清单。
单服务发布不会修改共享的可观测性组件。
Actions 的 `deploy_observability` 布尔输入默认开启；关闭后，全量发布会跳过三者的
清单应用、本地 PV 创建和就绪等待，集群中已有的 Loki、Tempo、Grafana 资源保持不变。

- Loki：grafana/loki:3.5.0，单副本 StatefulSet，10Gi PVC，使用本地 TSDB/文件存储。
- Tempo：grafana/tempo:2.7.2，单副本 StatefulSet，10Gi PVC，blocks 和 WAL 均保存在持久卷，保留 trace 48 小时。
- Grafana：grafana/grafana:11.5.2，单副本 Deployment，5Gi PVC 保存账号、面板和设置。
  使用 Recreate 更新策略，避免两个 Pod 同时写入单个数据卷。
- PVC 支持默认 StorageClass；Actions 的 --local-storage 会为未绑定卷创建 Retain 本地 PV。
  多节点使用本地卷时必须设置 STORAGE_NODE_NAME，卷数据不会自动跨节点迁移。
- 首次启动前会准备数据目录权限；部署等待 PVC 绑定以及三个工作负载就绪。
  ConfigMap 变更会通过配置 checksum 触发对应 Pod 更新。
- 普通 cleanup 保留数据卷，--delete-data 才删除这三个组件的 PVC 和项目静态 PV。

Grafana 自动配置 Loki（http://loki:3100）和 Tempo（http://tempo:3200）数据源。
浏览器访问 http://<节点IP>:30300，首次登录为 admin/admin，登录后按提示修改密码。
也可通过以下命令只在本机访问：

```sh
kubectl -n one-adventure port-forward service/grafana 3000:3000
```

Loki/Tempo 通过 ClusterIP 提供服务，没有开放节点端口。
这些单副本和本地存储配置提供持久化，不提供跨节点高可用；Grafana 暂未配置 Prometheus 数据源。
