package websocket_server

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/icreateapp-com/go-zLib/z/providers/auth_provider"
	"github.com/icreateapp-com/go-zLib/z/providers/config_provider"
	"github.com/icreateapp-com/go-zLib/z/providers/logger_provider"
	"github.com/icreateapp-com/go-zLib/z/servers/http_server"
	"github.com/olahol/melody"
	"go.uber.org/fx"
)

type WSHandlerRegister func(m *melody.Melody, hub *Hub)

type WSMessageMiddleware func(ms *melody.Session, raw []byte) (pass bool)

type In struct {
	fx.In

	LC       fx.Lifecycle
	Cfg      *config_provider.Config
	Log      *logger_provider.Logger
	Auth     *auth_provider.Auth
	Handlers []WSHandlerRegister   `group:"ws_handlers"`
	MsgMws   []WSMessageMiddleware `group:"ws_message_middlewares"`
}

type Server struct {
	m   *melody.Melody
	hub *Hub
	log *logger_provider.Logger
}

type Out struct {
	fx.Out

	Server *Server
	Melody *melody.Melody
	Hub    *Hub
	Route  http_server.RouteRegister `group:"routes"`
}

func NewWebSocketServer(in In) (Out, error) {
	mode := strings.TrimSpace(in.Cfg.GetString("websocket.mode", "gin"))
	if mode == "" {
		mode = "gin"
	}
	if mode != "gin" {
		return Out{}, nil
	}

	path := strings.TrimSpace(in.Cfg.GetString("websocket.path", "/ws"))
	if path == "" {
		path = "/ws"
	}

	m := melody.New()
	hub := NewHub()
	s := &Server{m: m, hub: hub, log: in.Log}

	// connect lifecycle
	m.HandleConnect(func(ms *melody.Session) {
		// auth
		tokenHeader := ""
		req := ms.Request
		if req != nil {
			tokenHeader = req.Header.Get("Authorization")
		}
		tokenQuery := ""
		if req != nil {
			tokenQuery = req.URL.Query().Get("token")
		}
		guard := "user"
		if req != nil {
			g := strings.TrimSpace(req.URL.Query().Get("guard"))
			if g != "" {
				guard = g
			}
		}

		userID := ""
		if req != nil {
			userID = strings.TrimSpace(req.URL.Query().Get("user_id"))
		}
		if guard == "client" {
			if userID == "" {
				userID = "anonymous"
			}
			meta := hub.Attach(ms, guard, userID)
			ms.Set("conn_id", meta.ConnID)
			ms.Set("guard", meta.Guard)
			ms.Set("user_id", meta.UserID)
			return
		}

		ok, _, authCtx, err := in.Auth.AuthenticateByGuard(guard, tokenHeader, tokenQuery)
		if !ok || err != nil || authCtx == nil {
			_ = ms.CloseWithMsg([]byte("unauthorized"))
			return
		}
		meta := hub.Attach(ms, guard, authCtx.UserID)
		ms.Set("conn_id", meta.ConnID)
		ms.Set("guard", meta.Guard)
		ms.Set("user_id", meta.UserID)
	})

	m.HandleDisconnect(func(ms *melody.Session) {
		hub.Detach(ms)
	})

	m.HandleMessage(func(ms *melody.Session, msg []byte) {
		hub.Touch(ms)

		for _, mw := range in.MsgMws {
			if mw == nil {
				continue
			}
			if mw(ms, msg) == false {
				return
			}
		}

		var env Envelope
		if err := json.Unmarshal(msg, &env); err != nil {
			return
		}
		if !ValidateEvent(env.Event) {
			return
		}

		switch env.Event {
		case EventSubscribe:
			var req SubscribeRequest
			if err := DecodeData(env.Data, &req); err == nil {
				hub.Subscribe(ms, req.Channels)
			}
		case EventUnsubscribe:
			var req SubscribeRequest
			if err := DecodeData(env.Data, &req); err == nil {
				hub.Unsubscribe(ms, req.Channels)
			}
		default:
			// other events are handled by middlewares/handlers
		}
	})

	// register handlers
	for _, h := range in.Handlers {
		if h != nil {
			h(m, hub)
		}
	}

	route := func(r *gin.Engine) {
		r.GET(path, func(c *gin.Context) {
			m.HandleRequest(c.Writer, c.Request)
		})
	}

	in.LC.Append(fx.Hook{OnStop: func(ctx context.Context) error {
		m.Close()
		return nil
	}})

	if in.Log != nil {
		in.Log.Infow("provider[websocket] enabled", "mode", mode, "path", path)
	}

	return Out{Server: s, Melody: m, Hub: hub, Route: route}, nil
}

func (s *Server) Send(ms *melody.Session, env Envelope) error {
	if strings.TrimSpace(env.ID) == "" {
		env.ID = NewEnvelope(env.Event).ID
	}
	if env.TS == 0 {
		env.TS = time.Now().UnixMilli()
	}
	if !ValidateEvent(env.Event) {
		return errors.New("INVALID_EVENT")
	}
	b, err := json.Marshal(env)
	if err != nil {
		return err
	}
	return ms.Write(b)
}

func (s *Server) Push(target PushTarget, env Envelope) int {
	sessions := s.hub.Targets(target)
	if len(sessions) == 0 {
		return 0
	}
	if strings.TrimSpace(env.ID) == "" {
		env.ID = NewEnvelope(env.Event).ID
	}
	if env.TS == 0 {
		env.TS = time.Now().UnixMilli()
	}
	if !ValidateEvent(env.Event) {
		return 0
	}
	b, err := json.Marshal(env)
	if err != nil {
		return 0
	}
	count := 0
	for _, ms := range sessions {
		if ms == nil {
			continue
		}
		_ = ms.Write(b)
		count++
	}
	return count
}

var WebSocketServerModule = fx.Options(
	fx.Provide(NewWebSocketServer),
)
