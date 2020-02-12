package main

import (
	"io/ioutil"
	"k8s-ca-dashboard-aggregator/watch"
	"log"
	"os/signal"
	"syscall"
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
		signal.Notify(wh.WebSocketHandle.SignalChan, syscall.SIGABRT)
	}()

	go func() {
		wh.PodWatch()
		signal.Notify(wh.WebSocketHandle.SignalChan, syscall.SIGABRT)
	}()

	go func() {
		wh.NodeWatch()
		signal.Notify(wh.WebSocketHandle.SignalChan, syscall.SIGABRT)
	}()

	go func() {
		wh.ServiceWatch("")
		signal.Notify(wh.WebSocketHandle.SignalChan, syscall.SIGABRT)
	}()

	signal.Notify(wh.WebSocketHandle.SignalChan, syscall.SIGINT, syscall.SIGTERM)
	<-wh.WebSocketHandle.SignalChan

}

func displayBuildTag() {
	imageVersion := "UNKNOWN"
	dat, err := ioutil.ReadFile("./build_number.txt")
	if err == nil {
		imageVersion = string(dat)
	}
	log.Printf("Image version: %s", imageVersion)
}
