package handler

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/zeromicro/go-zero/rest/httpx"

	"github.com/nrbackback/zero-chatroom/chat/internal/svc"
	"github.com/nrbackback/zero-chatroom/chat/internal/types"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func WsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.WsConnectRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.Error(w, err)
			return
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println(err)
			return
		}
		if err := conn.WriteMessage(1, []byte("connected")); err != nil {
			return
		}
		c := &Client{
			ID:   req.ID,
			conn: conn,
			send: make(chan Msg),
		}
		register(c)

		go c.writePump()
		go c.readPump()
	}
}
