package main

import (
	"fmt"
	"io/ioutil"
	"k8s-armo-collector/watch"
	"log"

	"github.com/armosec/utils-k8s-go/probes"

	"github.com/golang/glog"
)

func main() {

	isServerReady := false
	go probes.InitReadinessV1(&isServerReady)
	displayBuildTag()

	wh, err := watch.CreateWatchHandler()
	if err != nil {
		log.Fatalf("failed to initialize the WatchHandler, reason: %s", err.Error())
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
	glog.Error(wh.WebSocketHandle.SendReportRoutine(&isServerReady, wh.SetFirstReportFlag))

}

func displayBuildTag() {
	// flag.Set("alsologtostderr", "1")

	imageVersion := "local build"
	dat, err := ioutil.ReadFile("./build_number.txt")
	if err == nil {
		imageVersion = string(dat)
	} else {
		dat, err = ioutil.ReadFile("./build_date.txt")
		if err == nil {
			imageVersion = fmt.Sprintf("%s, date: %s", imageVersion, string(dat))
		}
	}
	fmt.Printf("Image version: %s", imageVersion)
}
