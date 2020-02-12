package watch

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/gorilla/websocket"
)

type ReqType int

const MAXPINGMESSAGE = 3

const (
	PING    ReqType = 0
	MESSAGE ReqType = 1
	EXIT    ReqType = 2
)

type DataSocket struct {
	message string
	RType   ReqType
}

type WebSocketHandler struct {
	data             chan DataSocket
	conn             *websocket.Conn
	u                url.URL
	mutex            *sync.Mutex
	SignalChan       chan os.Signal
	keepAliveCounter int
}

func createWebSocketHandler(urlWS, path, clusterName, customerGuid string) *WebSocketHandler {
	scheme := strings.Split(urlWS, "://")[0]
	host := strings.Split(urlWS, "://")[1]
	wsh := WebSocketHandler{data: make(chan DataSocket), keepAliveCounter: 0, u: url.URL{Scheme: scheme, Host: host, Path: path, ForceQuery: true}, mutex: &sync.Mutex{}, SignalChan: make(chan os.Signal)}
	q := wsh.u.Query()
	q.Add("customerGUID", customerGuid)
	q.Add("clusterName", clusterName)
	wsh.u.RawQuery = q.Encode()
	return &wsh
}

func (wsh *WebSocketHandler) reconnectToWebSocket() error {

	reconnectionCounter := 0
	var err error

	for reconnectionCounter < 5 {
		glog.Infof("connect try: %d", reconnectionCounter+1)
		if wsh.conn, _, err = websocket.DefaultDialer.Dial(wsh.u.String(), nil); err == nil {
			glog.Infof("connected successfully")
			return nil
		}
		glog.Error(err)
		reconnectionCounter++
		glog.Infof("wait 5 seconds before reconnecting")
		time.Sleep(time.Second * 5)
	}
	if reconnectionCounter == 5 {
		wsh.conn.Close()
		glog.Errorf("reconnectToWebSocket, cant connect to wbsocket")
		return fmt.Errorf("cant connect to wbsocket after %d tries", 5)
	}
	return nil
}

func (wsh *WebSocketHandler) sendReportRoutine() error {
	defer func() {
		if err := recover(); err != nil {
			glog.Errorf("RECOVER sendReportRoutine. %v", err)
		}
	}()
	for {

		data := <-wsh.data
		switch data.RType {
		case MESSAGE:
			glog.Infof("sending message")
			err := wsh.conn.WriteMessage(websocket.TextMessage, []byte(data.message))
			if err != nil {
				glog.Errorf("sendReportRoutine, WriteMessage to websocket: %v", err)
				if err := wsh.reconnectToWebSocket(); err != nil {
					glog.Errorf("sendReportRoutine. %s", err.Error())
					continue
				}
				glog.Infof("resending message")
				err := wsh.conn.WriteMessage(websocket.TextMessage, []byte(data.message))
				if err != nil {
					glog.Errorf("WriteMessage to websocket: %v", err)
					continue
				}
			}
			glog.Infof("message sent")

		case EXIT:
			wsh.conn.Close()
			glog.Warningf("web socket client got exit with message: %s", data.message)
			return nil
		}
	}
}

func (wsh *WebSocketHandler) pingPongRoutine() error {
	defer func() {
		if err := recover(); err != nil {
			glog.Errorf("RECOVER pingPongRoutine. %v", err)
		}
	}()
	for {
		time.Sleep(40 * time.Second)

		if err := wsh.conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(5*time.Second)); err != nil {
			glog.Errorf("PING. %v", err)
		}
		messageType, _, err := wsh.conn.ReadMessage()
		if err != nil {
			glog.Errorf("PONG. %v", err)
			if err := wsh.reconnectToWebSocket(); err != nil {
				glog.Error(err)
			}
		} else if messageType != websocket.PongMessage {
			glog.Error("PONG. expecting messageType 10 (pong type), received: %d", messageType)
		} else {
			continue
		}

		wsh.keepAliveCounter++

		if wsh.keepAliveCounter == MAXPINGMESSAGE {
			wsh.keepAliveCounter = 0
			glog.Warningf("sent %d pings without receiving any pongs. restaring connection", MAXPINGMESSAGE)

			if err := wsh.reconnectToWebSocket(); err != nil {
				return err
			}
		}
	}
}

//StartWebSokcetClient -
func (wsh *WebSocketHandler) StartWebSokcetClient() error {
	defer func() {
		if err := recover(); err != nil {
			glog.Errorf("RECOVER StartWebSokcetClient. %v", err)
		}
	}()
	glog.Infof("connecting to %s", wsh.u.String())
	if err := wsh.reconnectToWebSocket(); err != nil {
		return err
	}

	go func() {
		for {
			glog.Error(wsh.sendReportRoutine())
		}
	}()

	go func() {
		for {
			glog.Error(wsh.pingPongRoutine())
		}
	}()
	return nil
}

//SendMessageToWebSocket -
func (wh *WatchHandler) SendMessageToWebSocket(jsonData []byte) {
	data := DataSocket{message: string(jsonData), RType: MESSAGE}

	wh.WebSocketHandle.data <- data
}

// ListenerAndSender listen for changes in cluster and send reports to websocket
func (wh *WatchHandler) ListenerAndSender() {
	defer func() {
		if err := recover(); err != nil {
			glog.Errorf("RECOVER ListnerAndSender. %v", err)
		}
	}()

	//in the first time we wait until all the data will arrive from the cluster and the we will inform on every change
	glog.Infof("wait 40 seconds for aggragate the first data from the cluster\n")
	time.Sleep(40 * time.Second)
	wh.SetFirstReportFlag(true)
	for {
		jsonData := PrepareDataToSend(wh)
		if jsonData != nil {
			glog.Infof("%s\n", string(jsonData))
			wh.SendMessageToWebSocket(jsonData)
		}
		if wh.GetFirstReportFlag() {
			wh.SetFirstReportFlag(false)
		}
		if WaitTillNewDataArrived(wh) {
			continue
		}
	}
}
