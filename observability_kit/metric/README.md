# metric

Prometheus 拉取式指标服务，默认由业务进程在独立端口的 `/metrics` 暴露：

- CPU：`process_cpu_seconds_total`
- 内存：`process_resident_memory_bytes`、`process_virtual_memory_bytes`
- 网络字节：`process_network_receive_bytes_total`、`process_network_transmit_bytes_total`（由标准进程采集器在支持的平台提供）
