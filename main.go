package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"k8s-armo-collector/watch"
	"os"

	"github.com/armosec/capacketsgo/k8sshared/probes"
	"github.com/armosec/utils-k8s-go/armometadata"

	"github.com/golang/glog"
)

func LoadEnvironmentVaribles() error {
	_, err := armometadata.LoadConfig("", true)
	return err
}

func main() {

	isServerReady := false
	go probes.InitReadinessV1(&isServerReady)
	displayBuildTag()

	if _, err := os.Stat("/etc/config/clusterData.json"); errors.Is(err, os.ErrNotExist) {
		glog.Error("file /etc/config/clusterData.json is not exist, some features will not work(for example: scan new image)")
	} else if err := LoadEnvironmentVaribles(); err != nil {
		glog.Error(err)
	}

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
