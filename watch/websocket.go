package watch

import (
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

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
	keepAliveCounter int
}

func createWebSocketHandler(urlWS, path, clusterName, customerGuid string) *WebSocketHandler {
	scheme := strings.Split(urlWS, "://")[0]
	host := strings.Split(urlWS, "://")[1]
	wsh := WebSocketHandler{data: make(chan DataSocket), keepAliveCounter: 0, u: url.URL{Scheme: scheme, Host: host, Path: path, ForceQuery: true}}
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
		log.Printf("connect try: %d", reconnectionCounter+1)
		if wsh.conn, _, err = websocket.DefaultDialer.Dial(wsh.u.String(), nil); err == nil {
			break
		}
		log.Printf("dial: %v", err)
		reconnectionCounter++
		log.Printf("wait 5 seconds before reconnecting")
		time.Sleep(time.Second * 5)
	}
	if reconnectionCounter == 5 {
		return fmt.Errorf("ERROR: reconnectToWebSocket, cant connect to wbsocket")
	}

	wsh.conn.SetPongHandler(func(string) error {
		// log.Printf("pong recieved")
		wsh.keepAliveCounter = 0
		return nil
	})

	//this go function must created in order to get the pong
	go func() {
		for {
			if _, _, err := wsh.conn.ReadMessage(); err != nil {
				log.Print(err.Error())
			}
		}
	}()
	return nil
}

func (wsh *WebSocketHandler) sendReportRoutine() error {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("RECOVER sendReportRoutine. error: %v", err)
			wsh.conn.Close()
		}
	}()

	for {
		data := <-wsh.data
		switch data.RType {
		case MESSAGE:
			log.Println("Sending: message.")
			err := wsh.conn.WriteMessage(websocket.TextMessage, []byte(data.message))
			if err != nil {
				log.Println("ERROR in sendReportRoutine, WriteMessage to websocket:", err)
				if err := wsh.reconnectToWebSocket(); err != nil {
					log.Printf(err.Error())
					continue
				}
				err := wsh.conn.WriteMessage(websocket.TextMessage, []byte(data.message))
				if err != nil {
					log.Printf("WriteMessage to websocket: %v", err)
					continue
				}
				log.Println("resending: message.")
			}
		case EXIT:
			wsh.conn.Close()
			log.Printf("web socket client got exit with message: %s", data.message)
			return nil
		}
	}
}

func (wsh *WebSocketHandler) pingPongRoutine() error {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("RECOVER pingPongRoutine. error: %v", err)
			wsh.conn.Close()
		}
	}()
	for {
		time.Sleep(40 * time.Second)
		err := wsh.conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(5*time.Second))
		if err != nil {
			log.Println("Write Error: ", err)
		}
		wsh.keepAliveCounter++

		if wsh.keepAliveCounter == MAXPINGMESSAGE {
			wsh.keepAliveCounter = 0
			log.Printf("sent %d pings without receiving any pongs. restaring connection", MAXPINGMESSAGE)
			wsh.conn.Close()
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
			log.Printf("RECOVER StartWebSokcetClient. error: %v", err)
			wsh.conn.Close()
		}
	}()
	log.Printf("connecting to %s", wsh.u.String())
	if err := wsh.reconnectToWebSocket(); err != nil {
		return err
	}

	go func() {
		log.Print(wsh.sendReportRoutine())
	}()

	go func() {
		log.Print(wsh.pingPongRoutine())
	}()
	return nil
}

//SendMessageToWebSocket -
func (wh *WatchHandler) SendMessageToWebSocket(jsonData []byte) {
	data := DataSocket{message: string(jsonData), RType: MESSAGE}

	wh.WebSocketHandle.data <- data
}

// ListnerAndSender listen for changes in cluster and send reports to websocket
func (wh *WatchHandler) ListnerAndSender() {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("RECOVER ListnerAndSender. error: %v", err)
		}
	}()
	//in the first time we wait until all the data will arrive from the cluster and the we will inform on every change
	log.Printf("wait 40 seconds for aggragate the first data from the cluster\n")
	time.Sleep(40 * time.Second)
	wh.SetFirstReportFlag(true)
	for {
		jsonData := PrepareDataToSend(wh)
		if jsonData != nil {
			log.Printf("%s\n", string(jsonData))
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
