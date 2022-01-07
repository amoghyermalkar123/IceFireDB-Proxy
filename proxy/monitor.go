package proxy

import (
	"time"

	"github.com/sirupsen/logrus"

	"github.com/IceFireDB/IceFireDB-Proxy/pkg/cache"
	"github.com/IceFireDB/IceFireDB-Proxy/pkg/config"
	"github.com/IceFireDB/IceFireDB-Proxy/pkg/monitor"
)

func (p *Proxy) StartMonitor() {
	if config.Get().Cache.Enable {
		p.Cache = cache.New(
			time.Millisecond*time.Duration(config.Get().Cache.DefaultExpiration),
			time.Second*time.Duration(config.Get().Cache.CleanupInterval),
			config.Get().Cache.MaxItemsSize,
		)
	}

	hotKeyMonitorConf := &monitor.HotKeyConfS{
		Enable:                  config.Get().Monitor.HotKeyConf.Enable,
		MonitorJobInterval:      config.Get().Monitor.HotKeyConf.MonitorJobInterval,
		MonitorJobLifeTime:      config.Get().Monitor.HotKeyConf.MonitorJobLifetime,
		SecondHotThreshold:      config.Get().Monitor.HotKeyConf.SecondHotThreshold,
		SecondIncreaseThreshold: config.Get().Monitor.HotKeyConf.SecondIncreaseThreshold,
		LruSize:                 config.Get().Monitor.HotKeyConf.LruSize,
		EnableCache:             config.Get().Monitor.HotKeyConf.EnableCache,
		MaxCacheLifeTime:        config.Get().Monitor.HotKeyConf.MaxCacheLifeTime,
	}

	bigKeyMonitorConf := &monitor.BigKeyConfS{
		Enable:           config.Get().Monitor.BigKeyConf.Enable,
		KeyMaxBytes:      config.Get().Monitor.BigKeyConf.KeyMaxBytes,
		ValueMaxBytes:    config.Get().Monitor.BigKeyConf.ValueMaxBytes,
		LruSize:          config.Get().Monitor.BigKeyConf.LruSize,
		EnableCache:      config.Get().Monitor.BigKeyConf.EnableCache,
		MaxCacheLifeTime: config.Get().Monitor.BigKeyConf.MaxCacheLifeTime,
	}

	if config.Get().Monitor.SlowQueryConf.SlowQueryThreshold <= 0 {
		config.Get().Monitor.SlowQueryConf.SlowQueryThreshold = 100
	}

	if config.Get().Monitor.SlowQueryConf.MaxListSize <= 0 {
		config.Get().Monitor.SlowQueryConf.MaxListSize = 64
	}

	slowQueryMonitorConf := &monitor.SlowQueryConfS{
		Enable:                 config.Get().Monitor.SlowQueryConf.Enable,
		SlowQueryTimeThreshold: config.Get().Monitor.SlowQueryConf.SlowQueryThreshold,
		MaxListSize:            config.Get().Monitor.SlowQueryConf.MaxListSize,
	}

	mon, err := monitor.GetNewMonitor(hotKeyMonitorConf, bigKeyMonitorConf, slowQueryMonitorConf)
	if err != nil {
		logrus.Error("初始化指标遥测错误：", err)
		return
	}
	p.Monitor = mon

	_ = monitor.RunPrometheusExporter(mon, config.Get().PrometheusExporterConf)
}
