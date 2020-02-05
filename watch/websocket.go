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

func (wsh *WebSocketHandler) reconnectToWebSocket() {
	reconnectionCounter := 0
	var err error
	wsh.conn, _, err = websocket.DefaultDialer.Dial(wsh.u.String(), nil)
	defer wsh.conn.Close()

	for err != nil {
		if reconnectionCounter == 5 {
			log.Printf("ERROR: cant connect to wbsocket")
			return
		}
		log.Printf("dial: %v", err)
		reconnectionCounter++
		log.Printf("wait 5 seconds before tring to reconnect")
		time.Sleep(time.Second * 5)
		log.Printf("reconnect try number %d", reconnectionCounter+1)
		wsh.conn, _, err = websocket.DefaultDialer.Dial(wsh.u.String(), nil)
	}
	wsh.conn.SetPongHandler(func(string) error {
		// log.Printf("pong recieved")
		wsh.keepAliveCounter = 0
		return nil
	})

	//this go function must created in order to get the pong
	go func() {
		for {
			wsh.conn.ReadMessage()
		}
	}()
}

func (wsh *WebSocketHandler) sendReportRoutine() string {
	for {
		data := <-wsh.data
		switch data.RType {
		case MESSAGE:
			log.Println("Sending: message.")
			err := wsh.conn.WriteMessage(websocket.TextMessage, []byte(data.message))
			if err != nil {
				log.Println("ERROR in sendReportRoutine, WriteMessage to websocket:", err)
				wsh.reconnectToWebSocket()
				err := wsh.conn.WriteMessage(websocket.TextMessage, []byte(data.message))
				if err != nil {
					return fmt.Sprintf("WriteMessage to websocket: %v", err)
				}
				log.Println("resending: message.")
				break
			}
		case EXIT:
			return fmt.Sprintf("web socket client got exit with message: %s\n", data.message)
		}
	}
}

func (wsh *WebSocketHandler) pingPongRoutine() {
	for {
		time.Sleep(40 * time.Second)
		// log.Println("Sending: ping.")
		err := wsh.conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(5*time.Second))
		if err != nil {
			log.Println("Write Error: ", err)
			return
		}
		wsh.keepAliveCounter++

		if wsh.keepAliveCounter == MAXPINGMESSAGE {
			wsh.conn.Close()
			wsh.reconnectToWebSocket()
		}
	}
}

//StartWebSokcetClient -
func (wsh *WebSocketHandler) StartWebSokcetClient() {
	defer func() {
		if err := recover(); err != nil {
			fmt.Printf("RECOVER StartWebSokcetClient. error: %v", err)
		}
	}()

	log.Printf("connecting to %s", wsh.u.String())
	wsh.reconnectToWebSocket()

	go func() {
		log.Print(wsh.sendReportRoutine())
	}()

	go func() {
		wsh.pingPongRoutine()
	}()
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
			fmt.Printf("RECOVER ListnerAndSender. error: %v", err)
		}
	}()
	//in the first time we wait until all the data will arrive from the cluster and the we will inform on every change
	log.Printf("wait 40 seconds for aggragate the first data from the cluster\n")
	time.Sleep(40 * time.Second)
	wh.SetFirstReportFlag(true)
	for {
		jsonData := PrepareDataToSend(wh)
		if jsonData != nil {
			fmt.Printf("%s\n", string(jsonData))
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
