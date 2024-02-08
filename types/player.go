package types

import (
	"log"

	"github.com/gorilla/websocket"
)

type Player struct {
	Conn *websocket.Conn `json:"-"`
	Mark string          `json:"mark"` // Player mark: "X" or "O"
}

func (p *Player) setupCloseHandler(game *Game) {
	p.Conn.SetCloseHandler(func(code int, text string) error {
		// Handle player disconnection here
		log.Printf("Player disconnected: %v", p)
		game.HandleDisconnection(p)
		return nil
	})
}
