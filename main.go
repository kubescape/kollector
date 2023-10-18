package main

import (
	"context"
	"flag"
	"fmt"
	"net/url"
	"os"

	"github.com/kubescape/backend/pkg/servicediscovery"
	v1 "github.com/kubescape/backend/pkg/servicediscovery/v1"
	"github.com/kubescape/backend/pkg/utils"
	logger "github.com/kubescape/go-logger"
	"github.com/kubescape/go-logger/helpers"

	"github.com/kubescape/kollector/config"
	"github.com/kubescape/kollector/consts"
	"github.com/kubescape/kollector/watch"

	"github.com/armosec/utils-k8s-go/armometadata"
	"github.com/armosec/utils-k8s-go/probes"
)

func main() {
	ctx := context.Background()

	isServerReady := false
	go probes.InitReadinessV1(&isServerReady)
	displayBuildTag()

	clusterConfig, err := armometadata.LoadConfig(os.Getenv(consts.ConfigEnvironmentVariable))
	if err != nil {
		logger.L().Ctx(ctx).Fatal("failed to load config", helpers.Error(err))
	}

	services, err := servicediscovery.GetServices(
		v1.NewServiceDiscoveryFileV1("/etc/config/services.json"),
	)
	if err != nil {
		logger.L().Ctx(ctx).Fatal("failed to load services", helpers.Error(err))
	}

	logger.L().Info("loaded event receiver websocket url (service discovery)", helpers.String("url", services.GetReportReceiverWebsocketUrl()))

	sd, err := utils.LoadTokenFromFile("/etc/access-token-secret")
	if err != nil {
		logger.L().Ctx(ctx).Fatal("failed to get secret data", helpers.Error(err))
	}

	kollectorConfig := config.NewKollectorConfig(clusterConfig, *sd, services.GetReportReceiverWebsocketUrl())

	// to enable otel, set OTEL_COLLECTOR_SVC=otel-collector:4317
	if otelHost, present := os.LookupEnv(consts.OtelCollectorSvcEnvironmentVariable); present {
		ctx = logger.InitOtel("kollector",
			os.Getenv(consts.ReleaseBuildTagEnvironmentVariable),
			kollectorConfig.AccountID(),
			kollectorConfig.ClusterName(),
			url.URL{Host: otelHost})
		defer logger.ShutdownOtel(ctx)
	}

	wh, err := watch.CreateWatchHandler(kollectorConfig)
	if err != nil {
		logger.L().Ctx(ctx).Fatal("failed to initialize the WatchHandler", helpers.Error(err))
	}

	go func() {
		for {
			wh.ListenerAndSender(ctx)
		}
	}()

	go func() {
		for {
			wh.NodeWatch(ctx)
		}
	}()

	go func() {
		for {
			wh.PodWatch(ctx)
		}
	}()

	go func() {
		for {
			wh.ServiceWatch(ctx)
		}
	}()

	go func() {
		for {
			wh.SecretWatch(ctx)
		}
	}()
	go func() {
		for {
			wh.NamespaceWatch(ctx)
		}
	}()
	go func() {
		for {
			wh.CronJobWatch(ctx)
		}
	}()
	logger.L().Ctx(ctx).Fatal(wh.WebSocketHandle.SendReportRoutine(ctx, &isServerReady, wh.SetFirstReportFlag).Error())

}

func displayBuildTag() {
	flag.Parse()
	logger.L().Info(fmt.Sprintf("Image version: %s", os.Getenv(consts.ReleaseBuildTagEnvironmentVariable)))
}
