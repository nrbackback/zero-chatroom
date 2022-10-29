package handler

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"
	"github.com/zeromicro/go-zero/core/logx"

	"github.com/nrbackback/zero-chatroom/chat/internal/types"
)

type Hub struct {
	clients map[int]*Client

	broadcast chan Msg
}

type Msg struct {
	Msg    string
	FromID int
	ToIDs  []int
	Time   int64
}

type Client struct {
	ID   int
	conn *websocket.Conn
	send chan Msg
}

var h *Hub

func InitHub() {
	h = &Hub{
		broadcast: make(chan Msg),
		clients:   make(map[int]*Client),
	}
}

func RunHub() {
	for {
		select {
		case message := <-h.broadcast:
			for _, client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client.ID)
				}
			}
		}
	}
}

func register(c *Client) {
	h.clients[c.ID] = c
}

func unregister(c *Client) {
	delete(h.clients, c.ID)
	close(c.send)
}

var maxMessageSize = int64(512)
var pongWait = 60 * time.Second

func (c *Client) readPump() {
	logx.Errorw("test close writer error", logx.Field("error", "fdfsdfds"))
	defer func() {
		unregister(c)
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}
		var req types.WsRequestItem
		if err := json.Unmarshal(message, &req); err != nil {
			logx.Errorw("unmarshal message when read error", logx.Field("message", string(message)), logx.Field("error", err))
			continue
		}
		h.broadcast <- Msg{
			Msg:    req.Message,
			FromID: c.ID,
			ToIDs:  req.ToID,
			Time:   time.Now().Unix(),
		}
	}
}

var pingPeriod = (pongWait * 9) / 10
var writeWait = 10 * time.Second

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				logx.Errorw("NextWriterd error", logx.Field("error", err))
				continue
			}
			resp := types.WsResponseItem{
				Message: message.Msg,
				FromID:  message.FromID,
				Time:    message.Time,
			}
			v, _ := json.Marshal(resp)
			w.Write(v)
			if err := w.Close(); err != nil {
				logx.Errorw("close writer error", logx.Field("error", err))
				continue
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				logx.Errorw("write ping message error", logx.Field("error", err))
				continue
			}
		}
	}
}
