package main

import (
	"fmt"
	"k8s-ca-dashboard-aggregator/watch"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {

	if aggImageVersion := os.Getenv("CA_AGGREGATOR_IMAGE_VERSION"); aggImageVersion != "" {
		log.Printf("aggregator image version: %v\n", aggImageVersion)
	}

	wh := watch.CreateWatchHandler()

	if wh == nil {
		return
	}
	//start websocket
	wh.WebSocketHandle.StartWebSokcetClient()

	go func() {
		//in the first time we wait till all the data will arrive from the cluster and the we will inform on every change
		log.Printf("wait 40 seconds for aggragate the first data from the cluster\n")
		time.Sleep(40 * time.Second)
		for {
			jsonData := watch.PrepareDataToSend(wh)
			if jsonData != nil {
				fmt.Printf("%s\n", string(jsonData))
				wh.SendMessageToWebSocket(jsonData)
			}
			if watch.WaitTillNewDataArrived(wh) {
				continue
			}
		}
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

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan

}
