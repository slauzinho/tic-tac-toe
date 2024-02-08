package types

import (
	"log"

	"github.com/gorilla/websocket"
)

type Game struct {
	Board   [3][3]string `json:"board"`
	Player1 *Player      `json:"player1"`
	Player2 *Player      `json:"player2"`
	Current *Player      `json:"current"`
	Winner  string       `json:"winner"` // "X", "O" or "draw"
	Status  string       `json:"status"` // started, waiting, ended
}

func (g *Game) HandleDisconnection(disconnectedPlayer *Player) {
	g.Status = "waiting"

	log.Println(g.Player1 == disconnectedPlayer)

	if g.Player1 == disconnectedPlayer {
		g.Player1 = nil
		g.NotifyPlayer(g.Player2, "Your opponent has disconnected")
	} else if g.Player2 == disconnectedPlayer {
		g.Player2 = nil
		g.NotifyPlayer(g.Player1, "Your opponent has disconnected")
	}
}

func (g *Game) AddPlayer(conn *websocket.Conn) {
	player := &Player{Conn: conn}
	player.setupCloseHandler(g)
	if g.Player1 == nil {
		g.Player1 = player
		g.Player1.Mark = "X"
	} else if g.Player2 == nil {
		g.Player2 = player
		g.Player2.Mark = "O"
		g.Current = g.Player1
		g.Status = "started"
	}

	g.NotifyPlayer(player, "Waiting for opponent to join")
}

func (g *Game) NotifyPlayersGameStarted() {
	players := []*Player{g.Player1, g.Player2}

	for _, player := range players {
		data := struct {
			Game *Game  `json:"game"`
			Mark string `json:"mark"`
		}{
			Game: g,
			Mark: player.Mark,
		}

		message := Message{
			Type: "gameStarted",
			Data: data,
		}

		g.SendMessageToPlayer(player, message)
	}
}

func (g *Game) NotifyPlayerTurn() {
	data := struct {
		Game *Game `json:"game"`
	}{
		Game: g,
	}

	message := Message{
		Type: "playerTurn",
		Data: data,
	}

	g.SendMessageToPlayer(g.Player1, message)
	g.SendMessageToPlayer(g.Player2, message)
}

func (g *Game) NotifyPlayer(player *Player, message string) {
	data := struct {
		Message string `json:"message"`
		Game    *Game  `json:"game"`
	}{
		Message: message,
		Game:    g,
	}

	msg := Message{
		Type: "notification",
		Data: data,
	}

	g.SendMessageToPlayer(player, msg)
}

func (g *Game) NotifyPlayerGameEnded() {
	data := struct {
		Game *Game `json:"game"`
	}{
		Game: g,
	}
	message := Message{
		Type: "gameEnded",
		Data: data,
	}

	g.SendMessageToPlayer(g.Player1, message)
	g.SendMessageToPlayer(g.Player2, message)
}

func (g *Game) SendMessageToPlayer(player *Player, message Message) {
	if err := player.Conn.WriteJSON(message); err != nil {
		log.Printf("Error sending message to player: %v", err)
	}
}

func (g *Game) MakeMove(move Move, conn *websocket.Conn) {
	var player *Player

	if g.Player1.Conn == conn {
		player = g.Player1
	} else if g.Player2.Conn == conn {
		player = g.Player2
	}

	if player != g.Current {
		log.Println("It's not your turn")
		return
	}

	if g.Board[move.Row][move.Col] != "" {
		log.Println("Cell already taken")
		return
	}

	g.Board[move.Row][move.Col] = player.Mark

	hasWon := g.CheckWin()

	if hasWon {
		g.Status = "ended"
		g.Winner = player.Mark
		g.NotifyPlayerGameEnded()
		g.Current = nil

		return
	}

	if g.CheckDraw() {
		g.Status = "ended"
		g.Winner = "draw"
		g.NotifyPlayerGameEnded()
		g.Current = nil

		return
	}

	if g.Current == g.Player1 {
		g.Current = g.Player2
	} else {
		g.Current = g.Player1
	}

	g.NotifyPlayerTurn()
}

func (g *Game) CheckWin() bool {
	// Check rows
	for row := 0; row < 3; row++ {
		if g.Board[row][0] == g.Board[row][1] && g.Board[row][1] == g.Board[row][2] && g.Board[row][0] != "" {
			return true
		}
	}

	// Check columns
	for col := 0; col < 3; col++ {
		if g.Board[0][col] == g.Board[1][col] && g.Board[1][col] == g.Board[2][col] && g.Board[0][col] != "" {
			return true
		}
	}

	// Check diagonals
	if g.Board[0][0] == g.Board[1][1] && g.Board[1][1] == g.Board[2][2] && g.Board[0][0] != "" {
		return true
	}
	if g.Board[0][2] == g.Board[1][1] && g.Board[1][1] == g.Board[2][0] && g.Board[0][2] != "" {
		return true
	}

	// No winner found
	return false
}

func (g *Game) CheckDraw() bool {
	for row := 0; row < 3; row++ {
		for col := 0; col < 3; col++ {
			if g.Board[row][col] == "" {
				return false
			}
		}
	}
	return true
}
