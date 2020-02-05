package main

import (
	"io/ioutil"
	"k8s-ca-dashboard-aggregator/watch"
	"log"
	"os"
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
		wh.ListnerAndSender()
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

func displayBuildTag() {
	imageVersion := "UNKNOWN"
	dat, err := ioutil.ReadFile("./build_number.txt")
	if err == nil {
		imageVersion = string(dat)
	}
	log.Printf("Image version: %s", imageVersion)
}
