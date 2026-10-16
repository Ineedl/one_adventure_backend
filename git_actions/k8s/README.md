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
xds 与其他服务统一使用 `image_tag` 版本号。

例如 `service=user`、`image_tag=0.0.2` 只编译当前 checkout 的 user，构建并推送
`15342537816/one_adventure_backend:user-0.0.2`，更新 user 配置并等待其部署完成。
单服务发布要求集群已完成首次全量部署，复用 `middleware-auth` 中的密码。
选择 `all` 则发布全部 11 个应用并应用中间件清单。重复使用相同版本号也会重新拉取镜像并更新 Pod。
同一集群的发布串行执行，防止单服务与全量发布相互覆盖。
runner 必须已配置 Docker、kubectl、Go、Python 3/PyYAML，并且 kubeconfig 当前身份有目标集群部署权限。

部署脚本发布全部 11 个应用镜像对应的服务：gateway、gate-server、computing、
user、item、commerce、ws-gateway、order、pay、promotion、xds，以及 etcd、Kafka、Envoy、
MySQL 8.0 和 Redis 7.4。gateway 默认两个副本，其余各一个，稳定后预期有 17 个 Pod。

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
bash git_actions/k8s/deploy.sh 15342537816/one_adventure_backend 0.0.1 --deploy-only
kubectl get deployments,pods -n one-adventure
```

`--deploy-only` 跳过编译、镜像构建和推送。所有对应版本镜像必须已存在。
此模式需要 kubectl、Python 3 和 PyYAML，无需 Docker 和 gf。

当前 xds 镜像使用绝对路径启动 `/app/main`，并基于 Alpine 3.20 构建。重新构建后建议使用
新的 xds 版本标签；其他应用仍使用 `0.0.1` 时，部署需指定：

```sh
bash git_actions/k8s/deploy.sh 15342537816/one_adventure_backend 0.0.1 \
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

Deployment 会自动补齐期望副本，直接删除失败 Pod 会使其重建。
xds 滚动更新卡住时可能同时保留新旧 Pod，应修正 Deployment 的镜像版本。
副本数为 0 的旧 ReplicaSet 是回滚历史，不代表正在运行的 Pod。
Kafka 使用 Recreate 更新策略，更新时先停止旧 Pod，再启动新 Pod，期间服务会中断。
Kafka 数据目前使用 emptyDir，Pod 替换会丢失数据；需要保留数据时应先迁移至持久卷。

应用配置从 `<CONFIG_ROOT>/<服务源码目录>/manifest/config-docker` 生成 Secret，
保留 config.yaml、rpc.yaml、trace.yaml 等完整文件；CONFIG_ROOT 默认仓库根目录。
脚本在临时目录中将 config.yaml 的 MySQL 地址改为 `mysql:3306`，统一使用 root
和 MYSQL_ROOT_PASSWORD，将 Redis 地址改为 `redis:6379` 并设置 REDIS_PASSWORD。
源配置文件不会被修改，其他配置文件原样复制。
Secret 用于避免额外将数据库密码复制到部署清单，但不代表集群已启用静态加密。
脚本为应用 Pod 模板写入配置校验值，配置修改后再次部署会自动滚动更新对应应用。

MySQL、Redis 使用 namespace 内的 ClusterIP Service，不开放节点端口。
其他配置里的 host.docker.internal（如 trace 导出地址）默认解析到集群第一个节点的
InternalIP，其他主机地址可通过 HOST_GATEWAY_IP 指定。

```sh
HOST_GATEWAY_IP=192.168.1.39 bash git_actions/k8s/deploy.sh \
  15342537816/one_adventure_backend 0.0.1 --deploy-only
```

私有镜像可通过 `--docker-config` 导入当前用户的 Docker 登录凭据：

```sh
bash git_actions/k8s/deploy.sh 15342537816/one_adventure_backend 0.0.1 \
  --deploy-only --docker-config "$HOME/.docker/config.json"
```

也支持环境变量 `DOCKER_CONFIG_FILE`。脚本在目标 namespace 创建或更新
`dockerhub-auth` Secret，并为全部 11 个应用 Deployment 设置 imagePullSecrets。
已有 Deployment 首次添加引用时会自动滚动更新 Pod，保留其他镜像拉取 Secret。
文件必须包含实际 `auths` 凭据；仅有 `credsStore`/`credHelpers` 的文件不能直接用于集群拉取。
此参数仅配置 Kubernetes 拉取认证；完整发布的 Docker 推送仍使用当前用户的登录状态。
本次清单未迁移 Compose 的 Nginx 入口与可观测性组件；trace.yaml 中的 Tempo/Loki
地址需要已有对应服务或使用环境配置关闭导出。
etcd 与 Kafka 当前未配置持久卷；生产使用前需要添加持久化存储。

## MySQL 与 Redis

`middleware.yaml` 部署 MySQL `mysql:8.0` 和 Redis `redis:7.4-alpine`，分别使用
10Gi 和 1Gi 的 PVC；Redis 开启 AOF。集群需有默认 StorageClass，或预先提供匹配的 PV。
若出现 `pod has unbound immediate PersistentVolumeClaims`，表示 PVC 尚未绑定存储，
可通过 `kubectl get storageclass`、`kubectl describe pvc mysql-data -n one-adventure` 查看原因。
没有动态存储的单节点集群可直接重新部署并添加 `--local-storage`：

```sh
bash git_actions/k8s/deploy.sh 15342537816/one_adventure_backend 0.0.1 \
  --deploy-only --xds-tag 0.0.2 --local-storage
```

该参数为尚未绑定的 MySQL/Redis PVC 创建静态 hostPath PV，数据目录为节点上的
`/var/lib/one-adventure/<namespace>/mysql-data` 和 `redis-data`，并设置节点亲和性。
已绑定的 PVC 保持原存储，现有 PVC 无需删除；多节点集群必须通过 STORAGE_NODE_NAME
指定存储节点。此模式需要创建集群级 PV 的权限，节点磁盘需有足够空间，PVC 容量声明
不会限制 hostPath 实际用量。节点故障时数据不会自动迁移。
PV 使用 Retain 策略，因此 `cleanup.sh --delete-data` 删除 PVC 后，节点目录和 PV 仍会保留，
脚本不会将旧 PV 自动绑定给新 PVC；重新使用前需人工核对旧数据及 PV 归属。

两个 Deployment 都使用 Recreate，避免更新时多个实例同时写入同一个数据卷。
MySQL 8 首次初始化可能需要几十秒，启动探针会等待 30 秒后使用 `mysqladmin ping`
检查服务，最多允许 10 分钟完成初始化；启动期间出现少量 `ERROR 2003` 探针事件属于正常现象。
脚本等待两者健康检查通过后，才提交应用清单。

密码存储在 `middleware-auth` Secret。MySQL 首次启动自动创建 `test`、
`one_adventure_user`、`one_adventure_item`、`one_adventure_commerce`、
`one_adventure_server` 五个数据库。初始化仅在空数据卷执行，不包含业务表结构或旧数据，
需自行导入；CONFIG_ROOT 指向的自定义配置若使用其他库名，也需自行创建。

已有 MySQL 数据卷不会因修改 MYSQL_ROOT_PASSWORD 自动更新账号密码，密码变更需先在
MySQL 内执行对应的 ALTER USER，再同步部署配置。Redis 密码变更后需手动重启 Redis
（`kubectl rollout restart deployment/redis -n one-adventure`），使其加载新 Secret；
应用配置变更由脚本触发滚动更新。删除 Deployment 会保留 PVC，删除 PVC 可能丢失数据。

```sh
kubectl get deployments,pods,svc,pvc -n one-adventure
kubectl logs deployment/mysql -n one-adventure
kubectl logs deployment/redis -n one-adventure
```

## 本地端口转发

运行脚本后，MySQL 和 Redis 会临时暴露到本机回环地址；脚本保持前台运行，按 `Ctrl-C`
会自动结束两个转发进程，不创建或修改 Kubernetes 资源：

```sh
bash git_actions/k8s/port-forward-middleware.sh
```

默认连接地址：MySQL `127.0.0.1:13306`，Redis `127.0.0.1:16379`。可自定义本地端口：

```sh
bash git_actions/k8s/port-forward-middleware.sh \
  --mysql-port 3307 --redis-port 6380
```

也可以使用 `MYSQL_LOCAL_PORT`、`REDIS_LOCAL_PORT` 和 `KUBE_NAMESPACE` 环境变量。
脚本要求目标 namespace 中已存在 `mysql` 和 `redis` Service，并使用当前 kubectl context。

## 清理部署

清理全部 11 个应用及 etcd、Kafka、Envoy、MySQL、Redis 的 Deployment、Service，
同时删除应用配置 Secret、镜像认证 Secret 和项目 ConfigMap。Deployment 下属的
ReplicaSet 和 Pod 随之删除。执行时不询问确认。

```sh
# 预览当前 kubectl context 下将删除的资源
bash git_actions/k8s/cleanup.sh --dry-run

# 清理默认 one-adventure namespace 中的项目服务
bash git_actions/k8s/cleanup.sh

# 清理自定义部署 namespace（也支持 KUBE_NAMESPACE 环境变量）
bash git_actions/k8s/cleanup.sh my-one-adventure

# 预览包含 MySQL/Redis 数据卷的清理范围
bash git_actions/k8s/cleanup.sh --delete-data --dry-run

# 同时删除 MySQL/Redis 数据卷及密码 Secret
bash git_actions/k8s/cleanup.sh --delete-data
```

默认保留 namespace、MySQL/Redis PVC 和 `middleware-auth` Secret，重新部署时复用已有
数据和密码。指定 `--delete-data` 后，先清理服务，再删除 `mysql-data`、`redis-data` PVC
及 `middleware-auth` Secret；下次部署会重新生成密码。PVC 的删除可能永久清除数据，
底层 PV 按存储的 reclaimPolicy 回收，Retain 类型 PV 不会被脚本直接删除。
namespace 始终保留。etcd、Kafka 当前使用临时存储，清理 Pod 会丢失其中数据。
脚本仅允许清理带有 `app.kubernetes.io/part-of=one-adventure` 标签的项目 namespace，
该标签由部署清单自动设置。namespace 或资源不存在时跳过，支持重复执行。
服务清理与可选的数据卷清理各最多等待 5 分钟；超时会报错，不会强制移除 finalizer。

## Runner 与完整发布

Runner 路径为 `/usr/local/share/application/actions-runner`，配置示例见
`runner.env.example` 和 `actions.runner.service`。完整发布需要 Docker Compose、
Go、gf、kubectl 以及可访问集群的 kubeconfig，使用执行脚本的同一用户登录 Docker Hub。

```sh
bash git_actions/k8s/deploy.sh 15342537816/one_adventure_backend 0.0.1
```

也支持环境变量 IMAGE_REPOSITORY、IMAGE_TAG、KUBE_NAMESPACE。
镜像共享仓库，以 gateway-0.0.1、user-0.0.1 等标签区分。
