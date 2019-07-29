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
	var time_from_last_report time.Time = time.Now()
	wh := watch.CreateWatchHandler()

	go func() {
		for {
			if delta_time := time.Now().Sub(time_from_last_report); delta_time < 20*time.Second {
				time.Sleep(20*time.Second - delta_time)
			}
			jsonData := watch.PrepareDataToSend(&wh)
			if jsonData != nil {
				fmt.Printf("%s\n", string(jsonData))
				wh.SendMessageToWebSocket()
				watch.DeleteJsonData(&wh)
				time_from_last_report = time.Now()
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
