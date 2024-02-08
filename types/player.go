package types

import (
	"log"

	"github.com/gorilla/websocket"
)

type Player struct {
	Conn *websocket.Conn `json:"-"`
	Mark string          `json:"mark"` // Player mark: "X" or "O"
	Game *Game           `json:"-"`
}

func (p *Player) setupCloseHandler(game *Game) {
	p.Conn.SetCloseHandler(func(code int, text string) error {
		// Handle player disconnection here
		log.Printf("Player disconnected: %v", p)
		game.HandleDisconnection(p)
		return nil
	})
}

func (p *Player) SendMessageToPlayer(message Message) {
	if err := p.Conn.WriteJSON(message); err != nil {
		log.Printf("Error sending message to player: %v", err)
	}
}

func (p *Player) NotifyPlayer(message string) {
	data := struct {
		Message string `json:"message"`
		Game    *Game  `json:"game"`
	}{
		Message: message,
		Game:    p.Game,
	}

	msg := Message{
		Type: "notification",
		Data: data,
	}

	p.SendMessageToPlayer(msg)
}
