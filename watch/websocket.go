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

	for err != nil {
		if reconnectionCounter == 5 {
			panic("cant connect to wbsocket")
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
		defer log.Print(recover())
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
				log.Println("WriteMessage to websocket:", err)
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

	log.Printf("connecting to %s", wsh.u.String())
	wsh.reconnectToWebSocket()

	//defer conn.Close()

	go func() {
		log.Fatal(wsh.sendReportRoutine())
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
