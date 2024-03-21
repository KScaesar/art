package leaf

import (
	"encoding/json"
	"strconv"

	"code.xasxly.com/platform-server/artemis.git/pkg/leaf"
	"google.golang.org/protobuf/proto"

	"code.xasxly.com/platform-server/artemis.git/pkg/util"
)

//go:generate protoc --go_out=. base.proto auth.proto

type LeafId = uint16

const (
	// request
	LeafId_Ping LeafId = 0
	LeafId_Auth LeafId = 1

	// response
	LeafId_Pong  LeafId = 99
	LeafId_Error LeafId = 100
)

func NewLeafMessage(leafId LeafId, body proto.Message) *LeafMessage {
	return &LeafMessage{LeafId: leafId, Body: body}
}

// LeafMessage
// 公司的 ws server 使用了 Leaf framework
//
// message 代表一種高階抽象
// 內容有 header(metadata) + body
//
// message = leafId + leafBody
// http://code.xasxly.com/server/mch-durotar/-/blob/master/leaf/processor.go#L15-18
//
// 只使用 BigEndian
// http://code.xasxly.com/server/mch-durotar/-/blob/master/leaf/adapters.go#L57
//
// Leaf 中，当选择使用 TCP 协议时，在网络中传输的消息都会使用以下格式：
// --------------
// | len | data |
// --------------
// https://github.com/name5566/leaf/blob/master/TUTORIAL_ZH.md
type LeafMessage struct {
	LeafId LeafId // 2 byte 大小
	Body   proto.Message
}

func NewWsMessage(rawMessage []byte, sess *Session, logger util.Logger) *WsMessage {
	return &WsMessage{
		RawMessage: rawMessage,
		LeafId:     0,
		LeafBody:   nil,
		Parent:     sess,
		Logger:     logger,
	}
}

type WsMessage struct {
	RawMessage []byte

	LeafId   LeafId
	LeafBody []byte

	Parent *Session
	Logger util.Logger
}

//

type HandleFunc = util.HandleFunc[*WsMessage]
type Middleware = util.Middleware[*WsMessage]
type WsMux = util.Mux[LeafId, *WsMessage]

func NewWsMux() *WsMux {
	getLeafId := func(message *WsMessage) (string, error) {
		return "/" + strconv.Itoa(int(message.LeafId)), nil
	}

	mux := util.NewMux[LeafId](getLeafId)
	mux.SetDelimiter('/', true)
	mux.Handler(LeafId_Error, HandleErrorResponse)
	return mux
}

// handler

func PrintError(message *WsMessage, _ *util.RouteParam, err error) error {
	if err == nil {
		if message.LeafId != LeafId_Pong && message.LeafId != LeafId_Ping {
			message.Logger.Debug("'%v' handle leafId=%v success", message.Parent.Identifier, message.LeafId)
		}
		return nil
	}

	message.Logger.Error("'%v' handle leafId=%v fail: %v", message.Parent.Identifier, message.LeafId, err)
	return err
}

func HandleErrorResponse(message *WsMessage, _ *util.RouteParam) error {
	body := &leaf.ErrorResponse{}
	err := proto.Unmarshal(message.LeafBody, body)
	if err != nil {
		return err
	}

	var Err error
	statusName := leaf.STATUS_name[int32(body.Status)]
	if body.Msg != "" {
		Err = util.ErrorWrapWithMessage(util.ErrUniversal, "status=%v, text=%v", statusName, body.Msg)
	} else {
		Err = util.ErrorWrapWithMessage(util.ErrUniversal, "status=%v", statusName)
	}
	message.Logger.Error("'%v' receive ErrResponse: %v", message.Parent.Identifier, Err)

	message.Parent.Stop()
	return nil
}

func PrintDetailHandler(newBody func(leafId int) (any, bool)) HandleFunc {
	return func(message *WsMessage, _ *util.RouteParam) error {
		if newBody == nil {
			message.Logger.Debug("print websocket leafId=%v", message.LeafId)
			return nil
		}

		body, ok := newBody(int(message.LeafId))
		if !ok {
			message.Logger.Debug("print websocket leafId=%v", message.LeafId)
			return nil
		}

		err := proto.Unmarshal(message.LeafBody, body.(proto.Message))
		if err != nil {
			return err
		}

		bBody, err := json.Marshal(body)
		if err != nil {
			return err
		}

		message.Logger.Info("print message: %T=%v\n\n", body, string(bBody))
		return nil
	}
}

func SetMessageIdHandler() HandleFunc {
	return func(message *WsMessage, _ *util.RouteParam) error {
		msgId := util.GenerateRandomCode(16)
		message.Logger = message.Logger.WithMessageId(msgId)
		return nil
	}
}

func DecodeLeafMessageFunc(crypto bool, key []byte) HandleFunc {
	return func(message *WsMessage, _ *util.RouteParam) (err error) {
		if crypto {
			message.RawMessage, err = util.DecryptECB(key, message.RawMessage)
			if err != nil {
				return err
			}
		}

		id, body := leaf.DecodeLeafMessage(message.RawMessage)
		message.LeafId = id
		message.LeafBody = body

		if message.LeafId != LeafId_Pong && message.LeafId != LeafId_Ping {
			message.Logger.Info("'%v' receive leafId=%v", message.Parent.Identifier, message.LeafId)
		}
		return nil
	}
}

func ExcludedMessageMiddleware(excludedLeafId []int) Middleware {
	excluded := make(map[LeafId]bool, len(excludedLeafId))
	for i := 0; i < len(excludedLeafId); i++ {
		excluded[LeafId(excludedLeafId[i])] = true
	}

	return func(next HandleFunc) HandleFunc {
		return func(msg *WsMessage, route *util.RouteParam) error {
			if excluded[msg.LeafId] {
				return nil
			}
			return next(msg, route)
		}
	}
}
