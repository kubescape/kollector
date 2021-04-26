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
	//start websocket
	// if err := wh.WebSocketHandle.StartWebSokcetClient(); err != nil {
	// 	log.Print(err)
	// 	return
	// }

	go func() {
		wh.ListenerAndSender()
	}()

	go func() {
		wh.PodWatch()
	}()

	go func() {
		wh.NodeWatch()
	}()

	go func() {
		wh.ServiceWatch("")
	}()

	go func() {
		wh.SecretWatch()
	}()

	glog.Error(wh.WebSocketHandle.SendReportRoutine(&isServerReady))

}

func displayBuildTag() {
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
	glog.Infof("Image version: %s", imageVersion)
}
