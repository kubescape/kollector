package watch

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"runtime/debug"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	v1 "github.com/kubescape/backend/pkg/server/v1"
	logger "github.com/kubescape/go-logger"
	"github.com/kubescape/go-logger/helpers"
)

type ReqType int

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
	headers    http.Header
}

func getRequestHeaders(accessKey string) http.Header {
	headers := http.Header{}
	headers.Add(v1.AccessKeyHeader, accessKey)
	return headers
}

func createWebSocketHandler(u *url.URL, accessKey string) *WebSocketHandler {
	logger.L().Info("connecting websocket", helpers.String("URL", u.String()))
	wsh := WebSocketHandler{
		u:          *u,
		data:       make(chan DataSocket),
		mutex:      &sync.Mutex{},
		SignalChan: make(chan os.Signal),
		headers:    getRequestHeaders(accessKey),
	}
	return &wsh
}

func (wsh *WebSocketHandler) connectToWebSocket(ctx context.Context, sleepBeforeConnection time.Duration) (*websocket.Conn, error) {

	var err error
	var conn *websocket.Conn

	tries := 60
	for reconnectionCounter := 0; reconnectionCounter < tries; reconnectionCounter++ {
		time.Sleep(time.Second * 1)
		if conn, _, err = websocket.DefaultDialer.Dial(wsh.u.String(), wsh.headers); err == nil {
			logger.L().Ctx(ctx).Info("connected successfully", helpers.String("URL", wsh.u.String()))
			wsh.setPingPongHandler(ctx, conn)
			return conn, nil
		}
	}

	return nil, fmt.Errorf("cant connect to websocket after %d tries", tries)

}

// SendReportRoutine function sending updates
func (wsh *WebSocketHandler) SendReportRoutine(ctx context.Context, isServerReady *bool, reconnectCallback func(bool)) error {
	defer func() {
		if err := recover(); err != nil {
			logger.L().Ctx(ctx).Error("RECOVER sendReportRoutine", helpers.Interface("error", err), helpers.String("stack", string(debug.Stack())))
		}
	}()
	for {
		t := getNumericValueFromEnvVar(WaitBeforeReportEnv, 30)
		conn, err := wsh.connectToWebSocket(ctx, time.Duration(t)*time.Second)
		if err != nil {
			return err
		}
		*isServerReady = true

		wsh.handleSendReportRoutine(ctx, conn, reconnectCallback)
	}

	// use mutex for writing message that way if write failed only the failed writing will reconnect
}

func (wsh *WebSocketHandler) handleSendReportRoutine(ctx context.Context, conn *websocket.Conn, reconnectCallback func(bool)) error {
	for {
		data := <-wsh.data
		wsh.mutex.Lock()

		switch data.RType {
		case MESSAGE:
			timeID := time.Now().UnixNano()
			err := conn.WriteMessage(websocket.TextMessage, []byte(data.message))
			if err != nil {
				// count on K8s pod lifecycle logic to restart the process again and then reconnect
				os.Exit(4)

			} else {
				logger.L().Ctx(ctx).Debug("message sent", helpers.Int("time", int(timeID)))
			}
		case EXIT:
			logger.L().Ctx(ctx).Error("websocket received exit code exit", helpers.String("message", data.message))
			// count on K8s pod lifecycle logic to restart the process again and then reconnect
			os.Exit(4)
		}
		wsh.mutex.Unlock()
	}
}

func (wh *WatchHandler) SendMessageToWebSocket(jsonData []byte) {
	data := DataSocket{message: string(jsonData), RType: MESSAGE}

	wh.WebSocketHandle.data <- data
}

// ListenerAndSender listen for changes in cluster and send reports to websocket
func (wh *WatchHandler) ListenerAndSender(ctx context.Context) {
	defer func() {
		if err := recover(); err != nil {
			logger.L().Ctx(ctx).Error("RECOVER ListenerAndSender", helpers.Interface("error", err), helpers.String("stack", string(debug.Stack())))
		}
	}()
	wh.SetFirstReportFlag(true)
	for {
		jsonData := prepareDataToSend(ctx, wh)
		if jsonData == nil || isEmptyFirstReport(jsonData) {
			continue // skip (ususally first) report in case it is empty
		}
		if jsonData != nil {
			logger.L().Ctx(ctx).Debug("sending report to websocket", helpers.String("report", string(jsonData)))
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

func (wsh *WebSocketHandler) setPingPongHandler(ctx context.Context, conn *websocket.Conn) {
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
				logger.L().Ctx(ctx).Error(err.Error())
			}
			if counter > 2 {
				if end {
					return
				}
				logger.L().Ctx(ctx).Error("ping closed connection")
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
				logger.L().Ctx(ctx).Error("read message closed connection", helpers.Error(err))
				wsh.closeConnection(conn, "read message error")
				break
			}
			time.Sleep(timeout)
		}
	}()
}

func (wsh *WebSocketHandler) closeConnection(conn *websocket.Conn, message string) {
	wsh.mutex.Lock()
	conn.Close()
	wsh.mutex.Unlock()
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
