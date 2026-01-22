package websocket_server

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	EventHeartbeat   = "ws.heartbeat"
	EventSubscribe   = "ws.subscribe"
	EventUnsubscribe = "ws.unsubscribe"
)

type Envelope struct {
	ID    string      `json:"id"`
	Event string      `json:"event"`
	TS    int64       `json:"ts"`
	Data  interface{} `json:"data,omitempty"`
}

func NewEnvelope(event string) Envelope {
	return Envelope{ID: uuid.NewString(), Event: event, TS: time.Now().UnixMilli()}
}

func ValidateEvent(event string) bool {
	e := strings.TrimSpace(event)
	return e != "" && strings.HasPrefix(e, "ws.")
}

func (e Envelope) Validate() error {
	if strings.TrimSpace(e.ID) == "" {
		return errors.New("EMPTY_ID")
	}
	if !ValidateEvent(e.Event) {
		return errors.New("INVALID_EVENT")
	}
	if e.TS == 0 {
		return errors.New("EMPTY_TS")
	}
	return nil
}

func DecodeData[T any](data interface{}, out *T) error {
	if out == nil {
		return errors.New("NIL_OUT")
	}
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, out)
}

type SubscribeRequest struct {
	Channels []string `json:"channels"`
}

type PushTarget struct {
	Guard     string   `json:"guard"`
	UserID    string   `json:"user_id"`
	UserIDs   []string `json:"user_ids"`
	Channel   string   `json:"channel"`
	ConnID    string   `json:"conn_id"`
	ConnIDs   []string `json:"conn_ids"`
	Broadcast bool     `json:"broadcast"`
}

type PushEvent struct {
	Target PushTarget `json:"target"`
	Msg    Envelope   `json:"msg"`
}

type HeartbeatEventPayload struct {
	ConnID string `json:"conn_id"`
	Guard  string `json:"guard"`
	UserID string `json:"user_id"`
	Type   string `json:"type"`
	TS     int64  `json:"ts"`
}
