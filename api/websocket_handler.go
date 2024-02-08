package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/slauzinho/tic-tac-toe/types"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// CheckOrigin function should be carefully considered in a production environment
	// For development, you can allow connections from any origin like this:
	CheckOrigin: func(r *http.Request) bool { return true },
}

func NewGame() *types.Game {
	return &types.Game{
		Status: "waiting",
	}
}

var game = NewGame()

func PlayGame(w http.ResponseWriter, r *http.Request) {
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
		game.NotifyPlayersGameStarted()
	}

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			break
		}

		var msg types.Message
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

			var move types.Move
			if err := json.Unmarshal(moveData, &move); err != nil {
				log.Println("error unmarshalling message:", err)
				continue
			}

			log.Printf("Player made a move: Row %d, Col %d", move.Row, move.Col)

			game.MakeMove(move, conn)

		case "playAgain":
			if game.Winner != "" {
				game.ResetGame()
			}

		default:
			log.Println("unknown message type:", msg.Type)
		}
	}
}
