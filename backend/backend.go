package main

import (
	"context"
	"gomonitor/backend/server"
	"gomonitor/backend/utils"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/vo"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
)

type cmdFlags struct {
	NacosIP             string
	NacosPort           uint64
	NamespaceId         string
	ServerIP            string
	MonitorServices     []string
	MonitorServiceGroup string
}

var (
	flags cmdFlags
	//nacos默认配置
	logDir     string = "/tmp/nacos/log"
	cacheDir   string = "/tmp/nacos/cache"
	rotateTime string = "12h"
	maxAge     int64  = 3
	logLevel   string = "info"
	timeoutMs  uint64 = 5000
	//server
	monitorServer server.MonitorServer
	err           error
)

func init() {
	flag.StringVarP(&flags.NacosIP, "nacosip", "i", "", "nacos server ip")
	flag.StringVarP(&flags.MonitorServiceGroup, "group", "g", "DEFAULT_GROUP", "monitor service group")
	flag.StringArrayVar(&flags.MonitorServices, "monitorservices", nil, "monitor service names")
	flag.Uint64VarP(&flags.NacosPort, "nacosport", "p", 8848, "nacos server port")
	flag.StringVarP(&flags.NamespaceId, "namespace", "n", "public", "nacos namespace id (not namespace name)")
}

func main() {
	if utils.CheckK8s() {
		log.Info("running on kubernetes")
	} else {
		log.Info("running on bare metal")
		flag.Parse()
		log.Info(flags.MonitorServices)
		if flags.NacosIP == "" || flags.NamespaceId == "" || flags.MonitorServices == nil {
			flag.Usage()
			return
		}
		//get nacos config
		sc := []constant.ServerConfig{
			{
				IpAddr: flags.NacosIP,
				Port:   flags.NacosPort,
			},
		}
		cc := constant.ClientConfig{
			NamespaceId:         flags.NamespaceId, //namespace id
			TimeoutMs:           timeoutMs,
			NotLoadCacheAtStart: true,
			LogDir:              logDir,
			CacheDir:            cacheDir,
			RotateTime:          rotateTime,
			MaxAge:              maxAge,
			LogLevel:            logLevel,
		}

		monitorServer, err = server.NewBareMetalServer(vo.NacosClientParam{
			ClientConfig:  &cc,
			ServerConfigs: sc,
		}, flags.NacosIP, flags.NacosPort, flags.MonitorServices, flags.MonitorServiceGroup)

		if err != nil {
			log.Errorln(err)
			return
		}

		stopCh := make(chan os.Signal, 1)
		signal.Notify(stopCh, os.Interrupt, syscall.SIGTERM)
		ctx, cancel := context.WithCancel(context.Background())

		monitorServer.Start(ctx)

		<-stopCh
		cancel()
		time.Sleep(5 * time.Second)
	}
}
