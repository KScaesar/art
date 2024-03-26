package rabbit

import (
	"github.com/KScaesar/Artifex"
)

func HandleSetMessageId() IngressHandleFunc {
	return func(message *Ingress, _ *Artifex.RouteParam) error {
		msg_id := Artifex.GenerateRandomCode(12)
		message.MsgId = msg_id
		message.Logger = message.Logger.WithKeyValue("msg_id", msg_id)
		return nil
	}
}

func PrintError() func(msg *Ingress, _ *Artifex.RouteParam, err error) error {
	return func(msg *Ingress, _ *Artifex.RouteParam, err error) error {
		if err != nil {
			msg.Logger.Error("handle %v fail: %v", msg.AmqpMsg.RoutingKey, err)
			return err
		}
		msg.Logger.Info("handle %v success", msg.AmqpMsg.RoutingKey)
		return nil
	}
}
