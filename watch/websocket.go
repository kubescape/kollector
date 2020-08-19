package watch

import (
	"crypto/tls"
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

func createWebSocketHandler(urlWS, path, clusterName, customerGUID string) *WebSocketHandler {
	scheme := strings.Split(urlWS, "://")[0]
	host := strings.Split(urlWS, "://")[1]
	wsh := WebSocketHandler{data: make(chan DataSocket), keepAliveCounter: 0, u: url.URL{Scheme: scheme, Host: host, Path: path, ForceQuery: true}, mutex: &sync.Mutex{}, SignalChan: make(chan os.Signal)}
	q := wsh.u.Query()
	q.Add("customerGUID", customerGUID)
	q.Add("clusterName", clusterName)
	wsh.u.RawQuery = q.Encode()
	return &wsh
}

func (wsh *WebSocketHandler) connectToWebSocket() (*websocket.Conn, error) {

	var err error
	var conn *websocket.Conn

	if v, ok := os.LookupEnv("CA_IGNORE_VERIFY_CACLI"); ok && v != "" {
		websocket.DefaultDialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	tries := 5
	for reconnectionCounter := 0; reconnectionCounter < tries; reconnectionCounter++ {
		if conn, _, err = websocket.DefaultDialer.Dial(wsh.u.String(), nil); err == nil {
			glog.Infof("connected successfully")
			wsh.setPingPongHandler(conn)
			return conn, nil
		}
		glog.Error(err)
		glog.Infof("connect try: %d", reconnectionCounter)
		time.Sleep(time.Second * 5)
	}

	err = fmt.Errorf("cant connect to wbsocket after %d tries", tries)
	glog.Error(err)
	return nil, err

}

// SendReportRoutine function sending updates
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

	for {
		data := <-wsh.data
		switch data.RType {
		case MESSAGE:
			timeID := time.Now().UnixNano()
			glog.Infof("sending message, %d", timeID)
			wsh.mutex.Lock()
			err := conn.WriteMessage(websocket.TextMessage, []byte(data.message))
			if err != nil {
				glog.Errorf("In sendReportRoutine, %d, WriteMessage to websocket: %v", data.RType, err)
				if conn, err = wsh.connectToWebSocket(); err != nil {
					glog.Errorf("sendReportRoutine. %s", err.Error())
					wsh.mutex.Unlock()
					continue
				}
				glog.Infof("resending message. %d", timeID)
				err := conn.WriteMessage(websocket.TextMessage, []byte(data.message))
				if err != nil {
					glog.Errorf("WriteMessage, %d, %v", timeID, err)
					wsh.mutex.Unlock()
					continue
				}
			}
			wsh.mutex.Unlock()
			glog.Infof("message sent, %d", timeID)

		case EXIT:
			wsh.mutex.Lock()
			glog.Warningf("websocket received exit code exit. message: %s", data.message)
			if conn, err = wsh.connectToWebSocket(); err != nil {
				glog.Errorf("connectToWebSocket. %s", err.Error())
				wsh.mutex.Unlock()
				return err
			}
			wsh.mutex.Unlock()
		}
	}
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

func (wsh *WebSocketHandler) setPingPongHandler(conn *websocket.Conn) {
	end := false
	timeout := 10 * time.Second
	go func() {
		counter := 0
		defaultPING := conn.PingHandler()
		conn.SetPingHandler(func(message string) error {
			counter = 0
			return defaultPING(message)
		})

		defaultPONG := conn.PongHandler()
		conn.SetPongHandler(func(message string) error {
			counter = 0
			return defaultPONG(message)
		})

		// test ping-pong
		for {
			err := conn.WriteControl(websocket.PingMessage, []byte("ping"), time.Now().Add(timeout))
			if err != nil {
				glog.Errorf("WriteControl error: %s", err.Error())
			}
			time.Sleep(timeout)
			if counter > 3 {
				glog.Errorf("closeConnection")
				wsh.closeConnection(conn, "ping pong error")
				end = true
				return
			}
			counter++
		}
	}()
	go func() {
		for {
			if end {
				return
			}
			conn.ReadMessage()
		}
	}()
}

func (wsh *WebSocketHandler) closeConnection(conn *websocket.Conn, message string) {
	wsh.mutex.Lock()
	conn.Close()
	wsh.mutex.Unlock()
	wsh.data <- DataSocket{RType: EXIT, message: message}
}
