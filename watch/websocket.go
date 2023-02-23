package watch

import (
	"fmt"
	"net/url"
	"os"
	"runtime/debug"
	"strconv"
	"sync"
	"time"

	"github.com/armosec/utils-k8s-go/armometadata"
	"github.com/golang/glog"
	"github.com/gorilla/websocket"
)

type ReqType int

const (
	customerGuidQueryParamKey  = "customerGUID"
	clusterNameQueryParamKey   = "clusterName"
	EventReceiverWebsocketPath = "/k8s/cluster-reports"
)

const (
	PING    ReqType = 0
	MESSAGE ReqType = 1
	EXIT    ReqType = 2
)

const (
	WaitBeforeReportEnv = "WAIT_BEFORE_REPORT"
)

type DataSocket struct {
	message string
	RType   ReqType
}

type WebSocketHandler struct {
	data       chan DataSocket
	u          url.URL
	mutex      *sync.Mutex
	SignalChan chan os.Signal
}

func setWebSocketURL(config *armometadata.ClusterConfig) (*url.URL, error) {
	u, err := url.Parse(config.EventReceiverWebsocketURL)
	if err != nil {
		return nil, err
	}
	u.Path = EventReceiverWebsocketPath
	q := u.Query()
	q.Add(customerGuidQueryParamKey, config.AccountID)
	q.Add(clusterNameQueryParamKey, config.ClusterName)
	u.RawQuery = q.Encode()
	u.ForceQuery = true

	return u, nil
}
func createWebSocketHandler(u *url.URL) *WebSocketHandler {
	glog.Infof("websocket URL: %s", u.String())
	wsh := WebSocketHandler{
		u:          *u,
		data:       make(chan DataSocket),
		mutex:      &sync.Mutex{},
		SignalChan: make(chan os.Signal),
	}
	return &wsh
}

func (wsh *WebSocketHandler) connectToWebSocket(sleepBeforeConnection time.Duration) (*websocket.Conn, error) {

	var err error
	var conn *websocket.Conn

	tries := 60
	for reconnectionCounter := 0; reconnectionCounter < tries; reconnectionCounter++ {
		time.Sleep(time.Second * 1)
		if conn, _, err = websocket.DefaultDialer.Dial(wsh.u.String(), nil); err == nil {
			glog.Infof("connected successfully to: '%s", wsh.u.String())
			wsh.setPingPongHandler(conn)
			return conn, nil
		}
		glog.Error(err)
	}

	err = fmt.Errorf("cant connect to websocket after %d tries", tries)
	glog.Error(err)
	return nil, err

}

// SendReportRoutine function sending updates
func (wsh *WebSocketHandler) SendReportRoutine(isServerReady *bool, reconnectCallback func(bool)) error {
	defer func() {
		if err := recover(); err != nil {
			glog.Errorf("RECOVER sendReportRoutine. %v, stack: %s", err, debug.Stack())
		}
	}()
	for {
		t := getNumericValueFromEnvVar(WaitBeforeReportEnv, 30)
		conn, err := wsh.connectToWebSocket(time.Duration(t) * time.Second)
		if err != nil {
			glog.Error(err)
			return err
		}
		*isServerReady = true

		wsh.handleSendReportRoutine(conn, reconnectCallback)
	}

	// use mutex for writing message that way if write failed only the failed writing will reconnect
}

func (wsh *WebSocketHandler) handleSendReportRoutine(conn *websocket.Conn, reconnectCallback func(bool)) error {
ReconnectLoop:
	for {
		data := <-wsh.data
		wsh.mutex.Lock()

		switch data.RType {
		case MESSAGE:
			timeID := time.Now().UnixNano()
			glog.Infof("sending message, %d", timeID)

			err := conn.WriteMessage(websocket.TextMessage, []byte(data.message))
			if err != nil {
				// count on K8s pod lifecycle logic to restart the process again and then reconnect
				os.Exit(4)

				glog.Errorf("In sendReportRoutine, %d, WriteMessage to websocket: %v", data.RType, err)
				if reconnectCallback != nil {
					reconnectCallback(true)
				}
				t := getNumericValueFromEnvVar(WaitBeforeReportEnv, 60)
				if conn, err = wsh.connectToWebSocket(time.Duration(t) * time.Second); err != nil {
					// TODO: handle retries
					glog.Errorf("sendReportRoutine. %s", err.Error())
					wsh.mutex.Unlock()
					break ReconnectLoop
				}
				if reconnectCallback == nil {
					glog.Infof("resending message. %d", timeID)
					err := conn.WriteMessage(websocket.TextMessage, []byte(data.message))
					if err != nil {
						wsh.mutex.Unlock()
						glog.Errorf("WriteMessage, %d, %v", timeID, err)
						return fmt.Errorf("failed to connect to websocket")
					}
					glog.Infof("message resent, %d", timeID)
				}
			} else {
				glog.Infof("message sent, %d", timeID)
			}
		case EXIT:
			glog.Warningf("websocket received exit code exit. message: %s", data.message)
			// count on K8s pod lifecycle logic to restart the process again and then reconnect
			os.Exit(4)
		}
		wsh.mutex.Unlock()
	}
	return nil
}

func (wh *WatchHandler) SendMessageToWebSocket(jsonData []byte) {
	data := DataSocket{message: string(jsonData), RType: MESSAGE}

	wh.WebSocketHandle.data <- data
}

// ListenerAndSender listen for changes in cluster and send reports to websocket
func (wh *WatchHandler) ListenerAndSender() {
	defer func() {
		if err := recover(); err != nil {
			glog.Errorf("RECOVER ListenerAndSender. %v, stack: %s", err, debug.Stack())
		}
	}()
	wh.SetFirstReportFlag(true)
	for {
		jsonData := prepareDataToSend(wh)
		if jsonData != nil {
			if os.Getenv(printReportEnvironmentVariable) == "true" { // TODO: use logger levels instead
				glog.Infof("%s", string(jsonData))
			}
			wh.SendMessageToWebSocket(jsonData)
		}
		if wh.getFirstReportFlag() {
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
			if end {
				break
			}
			err := conn.WriteControl(websocket.PingMessage, []byte("ping"), time.Now().Add(timeout))
			if err != nil {
				glog.Errorf("WriteControl error: %s", err.Error())
			}
			if counter > 2 {
				if end {
					return
				}
				glog.Errorf("ping closed connection")
				wsh.closeConnection(conn, "ping error")
				end = true
				return
			}
			time.Sleep(timeout)
			counter++
		}
	}()
	go func() {
		for {
			if end {
				break
			}
			if _, _, err := conn.ReadMessage(); err != nil {
				if end {
					break
				}
				end = true
				glog.Errorf("read message closed connection: %s", err.Error())
				wsh.closeConnection(conn, "read message error")
				break
			}
			time.Sleep(timeout)
		}
	}()
}

func (wsh *WebSocketHandler) closeConnection(conn *websocket.Conn, message string) {
	glog.Infof("closing connection: %s", message)
	wsh.mutex.Lock()
	conn.Close()
	wsh.mutex.Unlock()
	glog.Infof("connection closed: %s", message)
	wsh.data <- DataSocket{RType: EXIT, message: message}
}

func getNumericValueFromEnvVar(envVar string, defaultValue int) int {
	if value := os.Getenv(envVar); value != "" {
		if value, err := strconv.Atoi(value); err == nil {
			return value
		}
	}
	return defaultValue
}
