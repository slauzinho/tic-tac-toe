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
	Board   [3][3]string
	Player1 *Player
	Player2 *Player
	Current *Player
	Status  string
}

type Player struct {
	Conn *websocket.Conn
	Mark string // Player mark: "X" or "O"
}

func NewGame() *Game {
	return &Game{
		Status: "waiting",
	}
}

// AddPlayer adds a player to the game
func (g *Game) AddPlayer(conn *websocket.Conn) {
	player := &Player{Conn: conn}
	if g.Player1 == nil {
		g.Player1 = player
		g.Player1.Mark = "X"
	} else if g.Player2 == nil {
		g.Player2 = player
		g.Player2.Mark = "O"
		g.Current = g.Player1
		g.Status = "started"
	}

}

func (g *Game) notifyPlayersGameStarted() {
	type GameStartedData struct {
		Board [3][3]string `json:"board"`
	}

	data := GameStartedData{
		Board: g.Board,
	}
	message := Message{
		Type: "gameStarted",
		Data: data,
	}
	g.sendMessageToPlayer(g.Player1, message)
	g.sendMessageToPlayer(g.Player2, message)
}

func (g *Game) notifyPlayerTurn() {

	type PlayerTurnData struct {
		Board [3][3]string `json:"board"`
		Turn  string       `json:"turn"`
	}

	data := PlayerTurnData{
		Board: g.Board,
	}

	if g.Current == g.Player1 {
		data.Turn = g.Player1.Mark
	} else {
		data.Turn = g.Player2.Mark
	}

	message := Message{
		Type: "playerTurn",
		Data: g.Board,
	}

	g.sendMessageToPlayer(g.Player1, message)
	g.sendMessageToPlayer(g.Player2, message)
}

func (g *Game) notifyPlayerWon() {
	type PlayerWonData struct {
		Board  [3][3]string `json:"board"`
		Winner string       `json:"winner"`
	}

	data := PlayerWonData{
		Board:  g.Board,
		Winner: g.Current.Mark,
	}
	message := Message{
		Type: "playerWon",
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
		g.notifyPlayerWon()
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


// Implement other game methods: MakeMove, CheckWin, etc.

var game = NewGame() // Global game instance

func playGame(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrade:", err)
		return
	}
	defer conn.Close()

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
