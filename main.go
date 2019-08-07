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

const TIME int = 20

func main() {
	var timeFromLastReport time.Time = time.Now()
	wh := watch.CreateWatchHandler()
	var cn string
	if cn = os.Getenv("CA_CLUSTER_NAME"); cn == "" {
		log.Println("there is no cluster name")
		cn = "superCluster"
		//return
	}
	//start websocket
	wh.WebSocketHandle.StartWebSokcetClient("report.eudev2.cyberarmorsoft.com", "k8s/cluster-reports", cn, "1e3a88bf-92ce-44f8-914e-cbe71830d566" /*customer guid*/)

	go func() {
		for {
			if deltaTime := time.Now().Sub(timeFromLastReport); deltaTime < 20*time.Second {
				time.Sleep(20*time.Second - deltaTime)
			}
			jsonData := watch.PrepareDataToSend(&wh)
			if jsonData != nil {
				fmt.Printf("%s\n", string(jsonData))
				wh.SendMessageToWebSocket()
				watch.DeleteJsonData(&wh)
				timeFromLastReport = time.Now()
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
