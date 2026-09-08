package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	obsconfig "one_adventure_observability_config"
	obslog "one_adventure_observability_log"
	metric "one_adventure_observability_metric"
	tracekit "one_adventure_observability_trace/trace"

	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/os/gcmd"

	"gate_server/internal/registration"
	"gate_server/internal/websocket"
)

var (
	Main = gcmd.Command{
		Name:  "main",
		Usage: "main",
		Brief: "start websocket server",
		Func: func(ctx context.Context, parser *gcmd.Parser) (err error) {
			observability, err := obsconfig.Load(ctx, "trace.yaml")
			if err != nil {
				return err
			}
			shutdownTrace, err := tracekit.Init("gate_server", observability.TraceRuntime())
			if err != nil {
				return err
			}
			defer shutdownTrace(context.Background())
			shutdownMetric, err := metric.Init(observability.MetricRuntime())
			if err != nil {
				return err
			}
			defer shutdownMetric(context.Background())
			shutdownLog := obslog.Init("gate_server", obslog.NewInstanceID("gate_server"), observability.LogRuntime())
			defer shutdownLog(context.Background())
			server, err := websocket.New(ctx)
			if err != nil {
				return err
			}
			if err = server.Start(); err != nil {
				return err
			}
			var regCfg struct {
				EtcdEndpoints []string      `json:"etcdEndpoints"`
				DialTimeout   time.Duration `json:"dialTimeout"`
				EnvoyAddress  string        `json:"envoyAddress"`
				LeaseTTL      int64         `json:"leaseTtl"`
				Address       string        `json:"address"`
				WSPort        int           `json:"wsPort"`
				InstanceID    string        `json:"instanceId"`
			}
			if v, e := g.Cfg().Get(ctx, "gateRegistration"); e == nil {
				_ = v.Scan(&regCfg)
			}
			if regCfg.DialTimeout <= 0 {
				regCfg.DialTimeout = 5 * time.Second
			}
			if regCfg.LeaseTTL <= 0 {
				regCfg.LeaseTTL = 30
			}
			if regCfg.WSPort == 0 {
				regCfg.WSPort = 8902
			}
			if regCfg.Address == "" {
				regCfg.Address = registration.LocalAddress()
			}
			if regCfg.InstanceID == "" {
				regCfg.InstanceID = obslog.NewInstanceID("gate_server")
			}
			lease, name, e := registration.Acquire(ctx, registration.Config{EtcdEndpoints: regCfg.EtcdEndpoints, DialTimeout: regCfg.DialTimeout, EnvoyAddress: regCfg.EnvoyAddress, LeaseTTL: regCfg.LeaseTTL, Address: regCfg.Address, WSPort: regCfg.WSPort, InstanceID: regCfg.InstanceID})
			if e != nil {
				return e
			}
			if lease == nil {
				_ = server.Shutdown()
				return fmt.Errorf("no unplayed servers available; gate server exiting")
			}
			serverInfo, e := registration.FetchServerInfo(ctx, regCfg.EnvoyAddress, name)
			if e != nil {
				_ = lease.Release(context.Background())
				_ = server.Shutdown()
				return e
			}
			cachedServer := websocket.ServerInfo{ServerId: int64(serverInfo.GetId()), ServerName: serverInfo.GetName()}
			cachedChannels := make([]websocket.ChannelInfo, 0, len(serverInfo.GetChannels()))
			for _, channel := range serverInfo.GetChannels() {
				cachedChannels = append(cachedChannels, websocket.ChannelInfo{ChannelId: int64(channel.GetId()), ChannelIndex: int(channel.GetIndex()), ChannelName: channel.GetName()})
			}
			server.Manager().SetServerInfo(cachedServer, cachedChannels)
			regCtx, cancelReg := context.WithCancel(context.Background())
			defer cancelReg()
			defer lease.Release(context.Background())
			go lease.KeepAlive(regCtx)
			go func() {
				ticker := time.NewTicker(10 * time.Second)
				defer ticker.Stop()
				for {
					select {
					case <-regCtx.Done():
						return
					case <-ticker.C:
						for channelID, n := range server.Manager().ChannelCounts() {
							key := fmt.Sprintf("gate_server:%s:ws_num:%d", regCfg.InstanceID, channelID)
							_ = g.Redis().SetEX(regCtx, key, n, 30)
						}
					}
				}
			}()
			_ = name
			ghttp.Wait()
			return server.Shutdown()
		},
	}
)
