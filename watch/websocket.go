package watch

import (
	"log"
	"net/url"
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

func CreateWebSocketHandler() *WebSocketHandler {
	return &WebSocketHandler{data: make(chan DataSocket), keepAliveCounter: 0}
}

func (wsh *WebSocketHandler) reconnectToWebSocket() {
	reconnectionCounter := 0
	var err error
	wsh.conn, _, err = websocket.DefaultDialer.Dial(wsh.u.String(), nil)

	for err != nil {
		log.Printf("dial: %v", err)
		reconnectionCounter++
		log.Printf("wait 5 seconds before tring to reconnect")
		time.Sleep(time.Second * 5)
		log.Printf("reconnect try number %d", reconnectionCounter+1)
		wsh.conn, _, err = websocket.DefaultDialer.Dial(wsh.u.String(), nil)
	}
	wsh.conn.SetPongHandler(func(string) error {
		log.Printf("in PongHandler")
		wsh.keepAliveCounter = 0
		return nil
	})
	go func() {
		for {
			wsh.conn.ReadMessage()
		}
	}()
}

//StartWebSokcetClient -
func (wsh *WebSocketHandler) StartWebSokcetClient(urlWS string, path string, cluster_name string, customer_guid string) {

	wsh.u = url.URL{Scheme: "wss", Host: urlWS, Path: path, ForceQuery: true}
	q := wsh.u.Query()
	q.Add("customerGUID", customer_guid)
	q.Add("clusterName", cluster_name)
	wsh.u.RawQuery = q.Encode()
	log.Printf("connecting to %s", wsh.u.String())
	wsh.reconnectToWebSocket()

	//defer conn.Close()

	go func() {
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
						log.Println("WriteMessage to websocket:", err)
					} else {
						log.Println("resending: message.")
						break
					}
				}
			case EXIT:
				log.Printf("web socket client got exit with message: %s\n", data.message)
				return
			}
		}
	}()

	go func() {
		for {
			time.Sleep(40 * time.Second)
			log.Println("Sending: ping.")
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
	}()
}

//SendMessageToWebSocket -
func (wh *WatchHandler) SendMessageToWebSocket(jsonData []byte) {
	data := DataSocket{message: string(jsonData), RType: MESSAGE}

	wh.WebSocketHandle.data <- data
}
