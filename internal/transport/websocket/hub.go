package websocket

import (
	"time"

	"github.com/bytedance/sonic"
	fw "github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"github.com/ridwanmuh3/simba-server/internal/model"
)

const (
	pongWait   = 60 * time.Second
	pingPeriod = 54 * time.Second
	maxMsgSize = 512
)

type Client struct {
	hub     *Hub
	conn    *fw.Conn
	send    chan []byte
	userID  uint
	dapurID uint
}

func (c *Client) Send(data []byte) {
	select {
	case c.send <- data:
	default:
	}
}

type Hub struct {
	clients    map[*Client]bool
	rooms      map[uint]map[*Client]bool
	register   chan *Client
	unregister chan *Client
	broadcast  chan broadcastMsg
	log        *zap.SugaredLogger
}

type broadcastMsg struct {
	dapurID uint
	userID  *uint
	data    []byte
}

func NewHub(log *zap.SugaredLogger) *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		rooms:      make(map[uint]map[*Client]bool),
		register:   make(chan *Client, 64),
		unregister: make(chan *Client, 64),
		broadcast:  make(chan broadcastMsg, 256),
		log:        log,
	}
}

func (h *Hub) Run() {
	for {
		select {
		case c := <-h.register:
			h.clients[c] = true
			if h.rooms[c.dapurID] == nil {
				h.rooms[c.dapurID] = make(map[*Client]bool)
			}
			h.rooms[c.dapurID][c] = true
		case c := <-h.unregister:
			if _, ok := h.clients[c]; ok {
				delete(h.clients, c)
				if room, ok := h.rooms[c.dapurID]; ok {
					delete(room, c)
					if len(room) == 0 {
						delete(h.rooms, c.dapurID)
					}
				}
				close(c.send)
			}
		case msg := <-h.broadcast:
			h.broadcastTo(msg)
		}
	}
}

func (h *Hub) broadcastTo(msg broadcastMsg) {
	room, ok := h.rooms[msg.dapurID]
	if !ok {
		return
	}
	for c := range room {
		if msg.userID != nil && c.userID != *msg.userID {
			continue
		}
		c.Send(msg.data)
	}
}

func (h *Hub) BroadcastToDapur(dapurID uint, event model.EventPayload[any]) {
	h.broadcast <- broadcastMsg{
		dapurID: dapurID,
		data:    mustMarshal(event),
	}
}

func (h *Hub) BroadcastToDapurAndUser(dapurID uint, userID uint, event model.EventPayload[any]) {
	h.broadcast <- broadcastMsg{
		dapurID: dapurID,
		userID:  &userID,
		data:    mustMarshal(event),
	}
}

func mustMarshal(v any) []byte {
	b, _ := sonic.Marshal(v)
	return b
}

func (h *Hub) Upgrade(c *fw.Conn) {
	auth, err := getAuthFromConn(c)
	if err != nil {
		_ = c.Close()
		return
	}

	client := &Client{
		hub:     h,
		conn:    c,
		send:    make(chan []byte, 256),
		userID:  auth.userID,
		dapurID: auth.dapurID,
	}
	h.register <- client
	defer func() { h.unregister <- client }()

	c.SetReadLimit(maxMsgSize)
	c.SetReadDeadline(time.Now().Add(pongWait))
	c.SetPongHandler(func(string) error {
		return c.SetReadDeadline(time.Now().Add(pongWait))
	})

	go func() {
		ticker := time.NewTicker(pingPeriod)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := c.WriteMessage(fw.PingMessage, nil); err != nil {
					return
				}
			case <-time.After(pongWait + pingPeriod):
				return
			}
		}
	}()

	go func() {
		for {
			msg, ok := <-client.send
			if !ok {
				_ = c.Close()
				return
			}
			if err := c.WriteMessage(fw.TextMessage, msg); err != nil {
				return
			}
		}
	}()

	for {
		if _, _, err := c.ReadMessage(); err != nil {
			break
		}
	}
}

type wsAuth struct {
	userID  uint
	dapurID uint
}

func getAuthFromConn(c *fw.Conn) (*wsAuth, error) {
	authVal := c.Locals("auth")
	if authVal == nil {
		return nil, fiber.NewError(fiber.StatusUnauthorized, "authentication required")
	}
	auth, ok := authVal.(*model.Auth)
	if !ok || auth.CurrentDapurID == nil {
		return nil, fiber.NewError(fiber.StatusUnauthorized, "dapur not selected")
	}
	return &wsAuth{
		userID:  uint(auth.ID),
		dapurID: *auth.CurrentDapurID,
	}, nil
}
