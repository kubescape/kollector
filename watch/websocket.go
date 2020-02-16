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
	data chan DataSocket
	// conn             *websocket.Conn
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

func (wsh *WebSocketHandler) connectToWebSocket() (*websocket.Conn, error) {

	reconnectionCounter := 0
	var err error
	var conn *websocket.Conn

	for reconnectionCounter < 5 {
		glog.Infof("connect try: %d", reconnectionCounter+1)
		if conn, _, err = websocket.DefaultDialer.Dial(wsh.u.String(), nil); err == nil {
			glog.Infof("connected successfully")
			return conn, nil
		}
		glog.Error(err)
		reconnectionCounter++
		glog.Infof("wait 5 seconds before reconnecting")
		time.Sleep(time.Second * 5)
	}
	if reconnectionCounter == 5 {
		glog.Errorf("connectToWebSocket, cant connect to wbsocket")
		return conn, fmt.Errorf("cant connect to wbsocket after %d tries", 5)
	}
	return conn, nil
}

func (wsh *WebSocketHandler) SendReportRoutine() error {
	defer func() {
		if err := recover(); err != nil {
			glog.Errorf("RECOVER sendReportRoutine. %v", err)
		}
	}()
	conn, err := wsh.connectToWebSocket()
	defer conn.Close()
	if err != nil {
		glog.Error(err)
		return err
	}

	// use mutex for writing message that way if write failed only the failed writing will reconnect
	var mutex = &sync.Mutex{}

	go func() {
		for {
			time.Sleep(40 * time.Second)
			// test ping works
			mutex.Lock()
			if e := conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(5*time.Second)); e != nil {
				glog.Errorf("PING. %v", e)
				if conn, e = wsh.connectToWebSocket(); e != nil {
					panic(e)
				}
			}
			mutex.Unlock()
		}
	}()

	for {
		data := <-wsh.data
		switch data.RType {
		case MESSAGE:
			timeID := time.Now().UnixNano()
			glog.Infof("sending message, %d", timeID)
			mutex.Lock()
			err := conn.WriteMessage(websocket.TextMessage, []byte(data.message))
			if err != nil {
				glog.Errorf("sendReportRoutine, %d, WriteMessage to websocket: %v", err)
				if conn, err = wsh.connectToWebSocket(); err != nil {
					glog.Errorf("sendReportRoutine. %s", err.Error())
					mutex.Unlock()
					continue
				}
				glog.Infof("resending message. %d", timeID)
				err := conn.WriteMessage(websocket.TextMessage, []byte(data.message))
				if err != nil {
					glog.Errorf("WriteMessage, %d, %v", timeID, err)
					mutex.Unlock()
					continue
				}
			}
			mutex.Unlock()
			glog.Infof("message sent, %d", timeID)

		case EXIT:
			glog.Warningf("websocket received exit code exit. message: %s", data.message)
			return nil
		}
	}
}

// func (wsh *WebSocketHandler) pingPongRoutine() error {
// 	for {
// 		time.Sleep(40 * time.Second)

// 		if err := wsh.conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(5*time.Second)); err != nil {
// 			glog.Errorf("PING. %v", err)
// 			if err := wsh.connectToWebSocket(); err != nil {
// 				panic(err)
// 			}
// 		}
// 		// messageType, _, _ := wsh.conn.ReadMessage()
// 		// if messageType != websocket.PongMessage {
// 		// 	glog.Error("PONG. expecting messageType 10 (pong type), received: %d", messageType)
// 		// } else {
// 		// 	continue
// 		// }

// 	}
// }

//StartWebSokcetClient -
// func (wsh *WebSocketHandler) StartWebSokcetClient() error {
// 	defer func() {
// 		if err := recover(); err != nil {
// 			glog.Errorf("RECOVER StartWebSokcetClient. %v", err)
// 		}
// 	}()
// 	glog.Infof("connecting to %s", wsh.u.String())
// 	if err := wsh.reconnectToWebSocket(); err != nil {
// 		return err
// 	}
// 	return nil
// }

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
