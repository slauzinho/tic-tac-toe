package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// CheckOrigin function should be carefully considered in a production environment
	// For development, you can allow connections from any origin like this:
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Move struct {
	Row int `json:"row"`
	Col int `json:"col"`
}

type Message struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

type Game struct {
	Board   [3][3]string `json:"board"`
	Player1 *Player      `json:"player1"`
	Player2 *Player      `json:"player2"`
	Current *Player      `json:"current"`
	Winner  string       `json:"winner"` // "X", "O" or "draw"
	Status  string       `json:"status"` // started, waiting, ended
}

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

func NewGame() *Game {
	return &Game{
		Status: "waiting",
	}
}

// AddPlayer adds a player to the game
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

	g.notifyPlayer(player, "Waiting for opponent to join")
}

func (g *Game) HandleDisconnection(disconnectedPlayer *Player) {
	g.Status = "waiting"

	log.Println(g.Player1 == disconnectedPlayer)

	if g.Player1 == disconnectedPlayer {
		g.Player1 = nil
		g.notifyPlayer(g.Player2, "Your opponent has disconnected")
	} else if g.Player2 == disconnectedPlayer {
		g.Player2 = nil
		g.notifyPlayer(g.Player1, "Your opponent has disconnected")
	}
}

func (g *Game) notifyPlayersGameStarted() {
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

		g.sendMessageToPlayer(player, message)
	}
}

func (g *Game) notifyPlayerTurn() {
	data := struct {
		Game *Game `json:"game"`
	}{
		Game: g,
	}

	message := Message{
		Type: "playerTurn",
		Data: data,
	}

	g.sendMessageToPlayer(g.Player1, message)
	g.sendMessageToPlayer(g.Player2, message)
}

func (g *Game) notifyPlayer(player *Player, message string) {
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

	g.sendMessageToPlayer(player, msg)
}

func (g *Game) notifyPlayerGameEnded() {
	data := struct {
		Game *Game `json:"game"`
	}{
		Game: g,
	}
	message := Message{
		Type: "gameEnded",
		Data: data,
	}

	g.sendMessageToPlayer(g.Player1, message)
	g.sendMessageToPlayer(g.Player2, message)
}

func (g *Game) sendMessageToPlayer(player *Player, message Message) {
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
		g.notifyPlayerGameEnded()
		g.Current = nil

		return
	}

	if g.CheckDraw() {
		g.Status = "ended"
		g.Winner = "draw"
		g.notifyPlayerGameEnded()
		g.Current = nil

		return
	}

	if g.Current == g.Player1 {
		g.Current = g.Player2
	} else {
		g.Current = g.Player1
	}

	g.notifyPlayerTurn()
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

var game = NewGame()

func playGame(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrade:", err)
		return
	}
	defer conn.Close()

	if game.Player1 != nil && game.Player2 != nil {
		log.Println("Game is full")
		return
	}

	game.AddPlayer(conn)

	if game.Status == "started" {
		game.notifyPlayersGameStarted()
	}

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			break
		}

		var msg Message
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Println("error unmarshalling message:", err)
			continue
		}

		switch msg.Type {
		case "move":

			if game.Status == "ended" {
				log.Println("Game has ended")
				continue
			}

			moveData, err := json.Marshal(msg.Data)
			if err != nil {
				log.Println("error marshalling move data:", err)
				break
			}

			var move Move
			if err := json.Unmarshal(moveData, &move); err != nil {
				log.Println("error unmarshalling message:", err)
				continue
			}

			log.Printf("Player made a move: Row %d, Col %d", move.Row, move.Col)

			game.MakeMove(move, conn)

		default:
			log.Println("unknown message type:", msg.Type)
		}
	}
}

func main() {
	http.HandleFunc("/ws", playGame)
	log.Fatal(http.ListenAndServe(":4000", nil))
}
