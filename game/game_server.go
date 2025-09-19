package game

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

/* ---------- Data Types ---------- */

// Player holds the websocket connection and its position.
type Player struct {
	Conn *websocket.Conn
	X    int
	Y    int
}

// Input from a player: movement deltas.
type PlayerInput struct {
	Dx int `json:"dx"`
	Dy int `json:"dy"`
}

// Internal channel message.
type channelInputType struct {
	Conn *websocket.Conn
	Dx   int
	Dy   int
}

/* ---------- Globals ---------- */

var (
	tickRate     = 20 // 20 updates/sec
	upgrader     = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	inputChan    = make(chan channelInputType, 1024) // buffered input channel
	players      = make(map[*websocket.Conn]*Player) // all connected players
	playersMutex sync.Mutex                          // guards the players map
	boardSize    = 200
)

/* ---------- WebSocket Handling ---------- */

// HandlePlayerJoin upgrades HTTP -> WS and starts reading that player’s input.
func HandlePlayerJoin(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("upgrade error:", err)
		return
	}

	// new player at (1,1)
	p := &Player{Conn: conn, X: 1, Y: 1}

	playersMutex.Lock()
	players[conn] = p
	playersMutex.Unlock()

	// each connection gets its own reader goroutine
	go readInputs(p)
}

// Continuously read input from one player and send to the central channel.
func readInputs(p *Player) {
	defer func() {
		p.Conn.Close()
		playersMutex.Lock()
		delete(players, p.Conn)
		playersMutex.Unlock()
	}()

	for {
		var in PlayerInput
		if err := p.Conn.ReadJSON(&in); err != nil {
			fmt.Println("player disconnected:", err)
			return
		}
		inputChan <- channelInputType{Conn: p.Conn, Dx: in.Dx, Dy: in.Dy}
	}
}

/* ---------- Game Loop ---------- */

// GameLoop ticks at tickRate and processes input & broadcasts state.
func GameLoop() {
	ticker := time.NewTicker(time.Second / time.Duration(tickRate))
	for range ticker.C {
		drainInputs()
		broadcastState()
	}
}

// Apply all queued inputs to update player positions.
func drainInputs() {
	for {
		select {
		case in := <-inputChan:
			playersMutex.Lock()
			if p, ok := players[in.Conn]; ok {
				p.X += in.Dx
				p.Y += in.Dy
				// clamp to board
				if p.X < 0 {
					p.X = 0
				}
				if p.X > boardSize-1 {
					p.X = boardSize - 1
				}
				if p.Y < 0 {
					p.Y = 0
				}
				if p.Y > boardSize-1 {
					p.Y = boardSize - 1
				}
			}
			playersMutex.Unlock()
		default:
			return
		}
	}
}

// Broadcast the current positions of all players to everyone.
func broadcastState() {
	type publicPos struct {
		X int `json:"x"`
		Y int `json:"y"`
	}

	playersMutex.Lock()
	state := make([]publicPos, 0, len(players))
	for _, p := range players {
		state = append(state, publicPos{X: p.X, Y: p.Y})
	}

	for conn := range players {
		if err := conn.WriteJSON(state); err != nil {
			fmt.Println("broadcast error:", err)
			conn.Close()
			delete(players, conn)
		}
	}
	playersMutex.Unlock()
}
