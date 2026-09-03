# Nginx LAB cluster

The LAB topology has one load-balancer and three proxy nodes:

```text
client :8080 -> nginx-lb -> nginx-node-{1,2,3} -> host gateway :8000
```

Start the gateway on host port `8000`, then start the cluster:

```bash
docker compose up -d nginx-lb
```

Use `http://localhost:8080` for the load-balanced entry point. Ports `8081`,
`8082`, and `8083` expose each proxy node directly for LAB comparison. The
Nginx-only health endpoint is `/nginx-health`.

Useful commands:

```bash
docker compose ps nginx-lb nginx-node-1 nginx-node-2 nginx-node-3
docker compose logs -f nginx-lb nginx-node-1 nginx-node-2 nginx-node-3
```
