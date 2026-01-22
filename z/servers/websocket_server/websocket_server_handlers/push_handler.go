package websocket_server_handlers

import (
	"context"
	"encoding/json"

	"github.com/icreateapp-com/go-zLib/z/providers/event_bus_provider"
	"github.com/icreateapp-com/go-zLib/z/providers/logger_provider"
	"github.com/icreateapp-com/go-zLib/z/servers/websocket_server"
	"github.com/olahol/melody"
	"go.uber.org/fx"
)

type In struct {
	fx.In

	LC  fx.Lifecycle
	Log *logger_provider.Logger
	Bus *event_bus_provider.EventBus `optional:"true"`
	WS  *websocket_server.Server     `optional:"true"`
	M   *melody.Melody               `optional:"true"`
	Hub *websocket_server.Hub        `optional:"true"`
}

func NewPushHandler(in In) {
	if in.WS == nil || in.Bus == nil {
		return
	}

	id := in.Bus.On("ws.push", func(ctx context.Context, event event_bus_provider.Event[any]) {
		// payload could be websocket_server.PushEvent or map
		b, err := json.Marshal(event.Payload)
		if err != nil {
			return
		}
		var pe websocket_server.PushEvent
		if err := json.Unmarshal(b, &pe); err != nil {
			return
		}
		if !websocket_server.ValidateEvent(pe.Msg.Event) {
			return
		}
		if pe.Msg.ID == "" {
			pe.Msg.ID = websocket_server.NewEnvelope(pe.Msg.Event).ID
		}
		if pe.Msg.TS == 0 {
			pe.Msg.TS = websocket_server.NewEnvelope(pe.Msg.Event).TS
		}
		in.WS.Push(pe.Target, pe.Msg)
	})

	in.LC.Append(fx.Hook{OnStop: func(ctx context.Context) error {
		in.Bus.Off("ws.push", id)
		return nil
	}})

	if in.Log != nil {
		in.Log.Infow("ws push handler enabled")
	}
}

var PushHandlerModule = fx.Options(
	fx.Invoke(NewPushHandler),
)
