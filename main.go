package main

import (
	"fmt"
	"io/ioutil"
	"k8s-ca-dashboard-aggregator/watch"

	"github.com/armosec/capacketsgo/k8sshared/probes"

	"github.com/golang/glog"
)

func main() {

	isServerReady := false
	go probes.InitReadinessV1(&isServerReady)
	displayBuildTag()

	wh := watch.CreateWatchHandler()

	if wh == nil {
		return
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
