package main

import (
	"io/ioutil"
	"k8s-ca-dashboard-aggregator/watch"
	"log"
	"os/signal"
	"syscall"
	"time"
)

func main() {

	displayBuildTag()

	wh := watch.CreateWatchHandler()

	if wh == nil {
		return
	}
	//start websocket
	if err := wh.WebSocketHandle.StartWebSokcetClient(); err != nil {
		log.Print(err)
		return
	}

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

	signal.Notify(wh.WebSocketHandle.SignalChan, syscall.SIGINT, syscall.SIGTERM)
	<-wh.WebSocketHandle.SignalChan

}

func displayBuildTag() {
	imageVersion := "local build. date: 12-02-2020"
	dat, err := ioutil.ReadFile("./build_number.txt")
	if err == nil {
		imageVersion = string(dat)
	}
	log.Printf("Image version: %s. date: %s (UTC)", imageVersion, time.Now().UTC().String())
}
