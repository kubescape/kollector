package main

import (
	"flag"
	"os"

	logger "github.com/kubescape/go-logger"
	"github.com/kubescape/go-logger/helpers"
	"github.com/kubescape/kollector/watch"

	"github.com/armosec/utils-k8s-go/probes"
)

func main() {

	isServerReady := false
	go probes.InitReadinessV1(&isServerReady)
	displayBuildTag()

	wh, err := watch.CreateWatchHandler()
	if err != nil {
		logger.L().Fatal("failed to initialize the WatchHandler", helpers.Error(err))
	}

	go func() {
		for {
			wh.ListenerAndSender()
		}
	}()

	go func() {
		for {
			wh.PodWatch()
		}
	}()

	go func() {
		for {
			wh.NodeWatch()
		}
	}()

	go func() {
		for {
			wh.ServiceWatch()
		}
	}()

	go func() {
		for {
			wh.SecretWatch()
		}
	}()
	go func() {
		for {
			wh.NamespaceWatch()
		}
	}()
	go func() {
		for {
			wh.CronJobWatch()
		}
	}()
	logger.L().Fatal(wh.WebSocketHandle.SendReportRoutine(&isServerReady, wh.SetFirstReportFlag).Error())

}

func displayBuildTag() {
	flag.Parse()
	logger.L().Info("Image version", helpers.String("release", os.Getenv("RELEASE")))
}
