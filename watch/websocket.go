package watch

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"runtime/debug"
	"strconv"
	"sync"
	"time"

	"github.com/armosec/utils-k8s-go/armometadata"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	logger "github.com/kubescape/go-logger"
	"github.com/kubescape/go-logger/helpers"
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
	logger.L().Info("connecting websocket", helpers.String("URL", u.String()))
	wsh := WebSocketHandler{
		u:          *u,
		data:       make(chan DataSocket),
		mutex:      &sync.Mutex{},
		SignalChan: make(chan os.Signal),
	}
	return &wsh
}

func (wsh *WebSocketHandler) connectToWebSocket(ctx context.Context, sleepBeforeConnection time.Duration) (net.Conn, error) {

	var err error
	var conn net.Conn

	tries := 60
	for reconnectionCounter := 0; reconnectionCounter < tries; reconnectionCounter++ {
		time.Sleep(time.Second * 1)

		if conn, _, _, err = ws.DefaultDialer.Dial(ctx, wsh.u.String()); err == nil {
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

func (wsh *WebSocketHandler) handleSendReportRoutine(ctx context.Context, conn net.Conn, reconnectCallback func(bool)) error {

	for {
		data := <-wsh.data
		wsh.mutex.Lock()

		switch data.RType {
		case MESSAGE:
			timeID := time.Now().UnixNano()
			err := wsutil.WriteClientText(conn, []byte(data.message))
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

func (wsh *WebSocketHandler) setPingPongHandler(ctx context.Context, conn net.Conn) {
	end := false
	timeout := 2 * time.Second

	counter := 0

	// test ping
	go func() {

		for {
			// Send ping
			err := wsutil.WriteClientMessage(conn, ws.OpPing, []byte{})

			if err != nil {
				logger.L().Ctx(ctx).Error("ping closed connection", helpers.String("error", err.Error()))
			} else {
				counter = 0
			}

			if end || counter > 3 {
				logger.L().Ctx(ctx).Error("ping closed connection", helpers.String("error", err.Error()))
				wsh.closeConnection(conn, "ping error")
				end = true
				break
			}

			time.Sleep(timeout)
			counter++

		}

	}()

	go func() {
		defer wsh.closeConnection(conn, "read message error")
		for {
			if end {
				break
			}

			header, err := ws.ReadHeader(conn)
			if err != nil {
				logger.L().Ctx(ctx).Error("read message closed connection", helpers.String("error", err.Error()))
				return
			}

			switch header.OpCode {
			case ws.OpClose:
				logger.L().Ctx(ctx).Error("read message closed connection")
				end = true
				return
			case ws.OpPing:
				err := wsutil.WriteClientMessage(conn, ws.OpPong, []byte{})
				if err != nil {
					logger.L().Ctx(ctx).Error("pong closed connection", helpers.String("error", err.Error()))
					return
				}
				counter = 0
			default:
				counter = 0
			}

			time.Sleep(timeout)

		}

	}()

}

func (wsh *WebSocketHandler) closeConnection(conn net.Conn, message string) {
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
