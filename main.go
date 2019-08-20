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
		//in the first time we wait till all the data will arrive from the cluster and the we will inform on every change
		log.Printf("wait 40 seconds for aggragate the first data from the cluster\n")
		time.Sleep(40 * time.Second)
		for {
			jsonData := watch.PrepareDataToSend(&wh)
			if jsonData != nil {
				fmt.Printf("%s\n", string(jsonData))
				wh.SendMessageToWebSocket(jsonData)
			}
			if watch.WaitTillNewDataArrived(&wh) {
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
