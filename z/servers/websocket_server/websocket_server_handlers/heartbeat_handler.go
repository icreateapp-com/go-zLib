package websocket_server_handlers

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/icreateapp-com/go-zLib/z/providers/config_provider"
	"github.com/icreateapp-com/go-zLib/z/providers/event_bus_provider"
	"github.com/icreateapp-com/go-zLib/z/providers/logger_provider"
	"github.com/icreateapp-com/go-zLib/z/servers/websocket_server"
	"github.com/olahol/melody"
	"go.uber.org/fx"
)

const (
	heartbeatSessionLastPongAtKey = "ws_hb_last_pong_at"
	heartbeatSessionLastPingAtKey = "ws_hb_last_ping_at"
)

type HeartbeatHandlerIn struct {
	fx.In

	Cfg *config_provider.Config
	Log *logger_provider.Logger
	Bus *event_bus_provider.EventBus `optional:"true"`
}

type HeartbeatMiddlewareOut struct {
	fx.Out

	MW websocket_server.WSMessageMiddleware `group:"ws_message_middlewares"`
}

func NewHeartbeatMiddleware(in HeartbeatHandlerIn) HeartbeatMiddlewareOut {
	interval := time.Duration(in.Cfg.GetInt("websocket.heartbeat.interval_sec", 20)) * time.Second
	if interval <= 0 {
		interval = 20 * time.Second
	}
	intervalMs := interval.Milliseconds()
	if intervalMs <= 0 {
		intervalMs = 20000
	}

	eventName := strings.TrimSpace(in.Cfg.GetString("websocket.heartbeat.event_name", "ws.heartbeat"))
	if eventName == "" {
		eventName = "ws.heartbeat"
	}

	mw := func(ms *melody.Session, raw []byte) bool {
		if ms == nil {
			return true
		}

		var env websocket_server.Envelope
		if err := json.Unmarshal(raw, &env); err != nil {
			return true
		}

		// only intercept heartbeat ping
		if env.Event != websocket_server.EventHeartbeat {
			return true
		}
		if s, ok := env.Data.(string); !ok || s != "ping" {
			return true
		}

		nowMs := time.Now().UnixMilli()
		ms.Set(heartbeatSessionLastPingAtKey, nowMs)

		ts := env.TS
		if ts == 0 {
			ts = nowMs
		}

		// emit immediately
		connID, _ := ms.Get("conn_id")
		guard, _ := ms.Get("guard")
		userID, _ := ms.Get("user_id")
		if in.Bus != nil {
			in.Bus.EmitAsync(context.Background(), eventName, map[string]interface{}{
				"conn_id": connID,
				"guard":   guard,
				"user_id": userID,
				"type":    "ping",
				"ts":      ts,
			})
		}

		// rate limit pong per session
		lastAny, _ := ms.Get(heartbeatSessionLastPongAtKey)
		lastMs, _ := lastAny.(int64)
		if lastMs > 0 && nowMs-lastMs < intervalMs {
			// don't propagate to core (avoid extra ping handling)
			return false
		}
		ms.Set(heartbeatSessionLastPongAtKey, nowMs)

		pong := websocket_server.NewEnvelope(websocket_server.EventHeartbeat)
		pong.ID = env.ID
		pong.TS = nowMs
		pong.Data = "pong"
		if b, err := json.Marshal(pong); err == nil {
			_ = ms.Write(b)
		}

		// do not propagate to server core
		return false
	}

	if in.Log != nil {
		in.Log.Infow("ws heartbeat handler enabled", "interval", interval.String(), "event", eventName)
	}

	return HeartbeatMiddlewareOut{MW: mw}
}

type HeartbeatScannerOut struct {
	fx.Out

	Register websocket_server.WSHandlerRegister `group:"ws_handlers"`
}

func NewHeartbeatTimeoutScannerRegister(in HeartbeatHandlerIn) HeartbeatScannerOut {
	scanInterval := time.Duration(in.Cfg.GetInt("websocket.heartbeat.interval_sec", 20)) * time.Second
	if scanInterval <= 0 {
		scanInterval = 20 * time.Second
	}
	timeout := time.Duration(in.Cfg.GetInt("websocket.heartbeat.timeout_sec", 60)) * time.Second
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	if timeout < scanInterval {
		timeout = scanInterval
	}

	register := func(m *melody.Melody, hub *websocket_server.Hub) {
		if hub == nil {
			return
		}
		go func() {
			ticker := time.NewTicker(scanInterval)
			defer ticker.Stop()
			for range ticker.C {
				sessions := hub.ListSessions()
				nowMs := time.Now().UnixMilli()
				for s := range sessions {
					if s == nil {
						continue
					}
					lastAny, _ := s.Get(heartbeatSessionLastPingAtKey)
					lastMs, _ := lastAny.(int64)
					if lastMs == 0 {
						continue
					}
					if nowMs-lastMs > timeout.Milliseconds() {
						_ = s.CloseWithMsg([]byte("heartbeat timeout"))
					}
				}
			}
		}()

		if in.Log != nil {
			in.Log.Infow("ws heartbeat timeout scanner enabled", "interval", scanInterval.String(), "timeout", timeout.String())
		}
	}

	return HeartbeatScannerOut{Register: register}
}

var HeartbeatHandlerModule = fx.Options(
	fx.Provide(NewHeartbeatMiddleware),
	fx.Provide(NewHeartbeatTimeoutScannerRegister),
)
