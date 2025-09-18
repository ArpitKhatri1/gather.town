package main

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type Book struct {
	ID     string `json:id`
	Title  string `json:title`
	Author string `json:author`
}

var book = []Book{
	{ID: "1", Title: "1983", Author: "arpit"},
	{ID: "12", Title: "1983afe", Author: "arpit"},
}

var (
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	clients  = make(map[*websocket.Conn]bool)
	brodcast = make(chan []byte)
	mu       sync.Mutex
	history  []string
)

func main() {

	r := gin.Default()

	r.StaticFile("/", "./index.html")

	r.GET("/ws", func(c *gin.Context) {
		wsHandler(c.Writer, c.Request)
	})

	go handleMessage()

	r.Run(":8080")

}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("There was some error updgradinig the packages")
		return
	}

	defer conn.Close()
	mu.Lock()
	clients[conn] = true
	for _, historyItem := range history {
		err := conn.WriteMessage(websocket.TextMessage, []byte(historyItem))
		if err != nil {
			delete(clients, conn)
			break
		}
	}
	mu.Unlock()

	for {
		_, msg, err := conn.ReadMessage() // read message from connenctions

		if err != nil {
			mu.Lock()
			delete(clients, conn)
			mu.Unlock()
			break

		}
		mu.Lock()
		history = append(history, string(msg))
		mu.Unlock()
		brodcast <- msg
	}
}

func handleMessage() {
	for {
		msg := <-brodcast
		mu.Lock()
		for conn := range clients {
			err := conn.WriteMessage(websocket.TextMessage, msg)
			if err != nil {
				conn.Close()
				delete(clients, conn)

			}
		}
		mu.Unlock()
	}

}
