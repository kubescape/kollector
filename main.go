package main

import (
	"fmt"
	"k8s-ca-dashboard-aggregator/watch"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const TIME int = 20

func main() {
	wh := watch.CreateWatchHandler()

	go func() {
		for {
			time.Sleep(20 * time.Second)
			fmt.Printf("%s\n", string(watch.PrepareDataToSend(&wh)))
			wh.SendMessageToWebSocket()
			watch.DeleteJsonData(&wh)
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
