package leaf

import (
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"code.xasxly.com/platform-server/artemis.git/pkg/pubsub"
)

var DefaultDialer = websocket.DefaultDialer

type GorillaConn = websocket.Conn

func DecorateAdapter(adp pubsub.IAdapter) pubsub.IAdapter {
	wsId := pubsub.GenerateRandomCode(6)
	wsName := adp.Identifier()

	logger := adp.Log().
		WithKeyValue("ws_id", wsId).
		WithKeyValue("ws_name", wsName)
	adp.SetLog(logger)
	return adp
}

type Websocket interface {
	pubsub.IAdapter
	Send(messages ...*Egress) error
	Listen() (err error)
}

type ClientFactory struct {
	Hub             *pubsub.Hub[pubsub.IAdapter]
	SendPingSeconds int
	Connection      func() (*GorillaConn, error)
	RetryAction     func(Websocket) error

	containerMutex sync.Mutex
	Container      ApplicationContainer
}

func (f *ClientFactory) CreateWebsocket(name string) (Websocket, error) {
	conn, Err := f.Connection()
	if Err != nil {
		return nil, Err
	}

	f.containerMutex.Lock()
	defer f.containerMutex.Unlock()
	defer f.Container.NextApp()

	ingressMux, egressMux := f.Container.IngressMux(), f.Container.EgressMux(conn)

	opt := pubsub.NewPubSubOption[Ingress, Egress]().
		Identifier(name).
		AdapterHub(f.Hub).
		DecorateAdapter(f.Container.DecorateAdapter).
		Lifecycle(f.Container.Lifecycle).
		HandleRecv(ingressMux.HandleMessage)

	var mu sync.Mutex
	f.pingpong(ingressMux, opt, &mu, conn)

	if f.RetryAction != nil {
		retryCnt := 0
		opt.AdapterFixup(0, func(adp pubsub.IAdapter) error {
			retryCnt++
			adp.Log().Info("retry %v times ws start", retryCnt)
			mu.Lock()
			freshConn, err := f.Connection()
			if err != nil {
				mu.Unlock()
				adp.Log().Error("retry ws fail: %v", err)
				return Err
			}
			mu.Unlock()
			retryCnt = 0
			adp.Log().Info("retry ws success")
			conn = freshConn
			return f.RetryAction(adp.(Websocket))
		})
	}

	opt.AdapterSend(func(adp pubsub.IAdapter, message *Egress) (err error) {
		logger := adp.Log().WithKeyValue("msg_id", message.MsgId())

		mu.Lock()
		defer mu.Unlock()

		err = egressMux.HandleMessage(message, nil)
		if err != nil {
			logger.Error("send leafId=%v: %v", message.LeafId, err)
			return
		}
		if message.LeafId != LeafId_Pong && message.LeafId != LeafId_Ping {
			logger.Info("send leafId=%v", message.LeafId)
		}
		return nil
	})

	opt.AdapterRecv(func(adp pubsub.IAdapter) (*Ingress, error) {
		_, bMessage, err := conn.ReadMessage()
		if err != nil {
			return nil, err
		}
		return NewIngress(bMessage, adp.(Websocket)), nil
	})

	opt.AdapterStop(func(adp pubsub.IAdapter) (err error) {
		defer func() {
			if err != nil {
				adp.Log().Error("ws stop: %v", err)
			}
			adp.Log().Info("ws stop")
		}()

		mu.Lock()
		defer mu.Unlock()

		err = conn.WriteMessage(websocket.CloseMessage, nil)
		if err != nil {
			return err
		}
		return conn.Close()
	})

	adp, err := opt.Build()
	if err != nil {
		return nil, err
	}
	return adp.(Websocket), err
}

func (f *ClientFactory) pingpong(
	ingressMux *IngressMux,
	opt *pubsub.AdapterOption[Ingress, Egress],
	mu *sync.Mutex,
	conn *GorillaConn,
) {
	if f.SendPingSeconds <= 0 {
		return
	}

	waitNotify := make(chan error, 1)
	ingressMux.HandlerByNumber(LeafId_Pong, func(_ *Ingress, _ *pubsub.RouteParam) error {
		// fmt.Printf("handle pong\n")
		waitNotify <- nil
		return nil
	})
	opt.SendPing(func() error {
		bMessage, err := EncodeLeafMessage(NewEgress(LeafId_Ping, &Ping{}))
		if err != nil {
			return err
		}
		mu.Lock()
		defer mu.Unlock()
		// fmt.Printf("send ping\n")
		return conn.WriteMessage(websocket.BinaryMessage, bMessage)
	}, waitNotify, f.SendPingSeconds*2)
}

//

type Server struct {
	Hub             *pubsub.Hub[pubsub.IAdapter]
	WaitPingSeconds int
	Upgrade         websocket.Upgrader
	Authenticate    func(w http.ResponseWriter, r *http.Request) (wsName string, err error)

	containerMutex sync.Mutex
	Container      ApplicationContainer
}

func (f *Server) Serve(c *gin.Context) {
	session, err := f.createWebsocket(c.Writer, c.Request)
	if err != nil {
		return
	}
	<-session.WaitStop()
}

func (f *Server) createWebsocket(w http.ResponseWriter, r *http.Request) (Websocket, error) {
	wsName, err := f.Authenticate(w, r)
	if err != nil {
		return nil, err
	}

	conn, err := f.Upgrade.Upgrade(w, r, nil)
	if err != nil {
		return nil, err
	}

	f.containerMutex.Lock()
	defer f.containerMutex.Unlock()
	defer f.Container.NextApp()

	ingressMux, egressMux := f.Container.IngressMux(), f.Container.EgressMux(conn)

	opt := pubsub.NewPubSubOption[Ingress, Egress]().
		Identifier(wsName).
		AdapterHub(f.Hub).
		DecorateAdapter(f.Container.DecorateAdapter).
		Lifecycle(f.Container.Lifecycle).
		HandleRecv(ingressMux.HandleMessage)

	var mu sync.Mutex
	f.pingpong(ingressMux, opt, &mu, conn)

	opt.AdapterSend(func(adp pubsub.IAdapter, message *Egress) (err error) {
		logger := adp.Log().WithKeyValue("msg_id", message.MsgId())

		mu.Lock()
		defer mu.Unlock()

		err = egressMux.HandleMessage(message, nil)
		if err != nil {
			logger.Error("send leafId=%v: %v", message.LeafId, err)
			return
		}
		if message.LeafId != LeafId_Pong && message.LeafId != LeafId_Ping {
			logger.Info("send leafId=%v", message.LeafId)
		}
		return nil
	})

	opt.AdapterRecv(func(adp pubsub.IAdapter) (*Ingress, error) {
		_, bMessage, err := conn.ReadMessage()
		if err != nil {
			return nil, err
		}
		return NewIngress(bMessage, adp.(Websocket)), nil
	})

	opt.AdapterStop(func(adp pubsub.IAdapter) (err error) {
		defer func() {
			if err != nil {
				adp.Log().Error("ws stop: %v", err)
			}
			adp.Log().Info("ws stop")
		}()

		mu.Lock()
		defer mu.Unlock()

		err = conn.WriteMessage(websocket.CloseMessage, nil)
		if err != nil {
			return err
		}
		return conn.Close()
	})

	adp, err := opt.Build()
	if err != nil {
		return nil, err
	}
	return adp.(Websocket), err
}

func (f *Server) pingpong(
	ingressMux *IngressMux,
	opt *pubsub.AdapterOption[Ingress, Egress],
	mu *sync.Mutex,
	conn *GorillaConn,
) {
	if f.WaitPingSeconds <= 0 {
		return
	}

	waitNotify := make(chan error, 1)
	ingressMux.HandlerByNumber(LeafId_Ping, func(_ *Ingress, _ *pubsub.RouteParam) error {
		// fmt.Printf("handle ping\n")
		waitNotify <- nil
		return nil
	})

	opt.WaitPing(waitNotify, f.WaitPingSeconds, func() error {
		bMessage, err := EncodeLeafMessage(NewEgress(LeafId_Pong, &Pong{}))
		if err != nil {
			return err
		}
		mu.Lock()
		defer mu.Unlock()
		// fmt.Printf("send pong\n")
		return conn.WriteMessage(websocket.BinaryMessage, bMessage)
	})
	return
}
