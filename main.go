package main

import (
	"log"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
)

// create struct message for response
type Message struct {
	Name    string
	Message string
}

// create struct hub to control and broadcast message to client
type hub struct {
	clients               map[*websocket.Conn]bool // manage client connected or not
	clientRegisterChannel chan *websocket.Conn     // register client from channel
	clientRemovalChannel  chan *websocket.Conn     // delete client from channel
	broadcastMessage      chan Message             // channel for broadcast message
}

// run hub on concurrency / asynchronous
func (h *hub) run() {
	for {
		select {
		case conn := <-h.clientRegisterChannel:
			h.clients[conn] = true
		case conn := <-h.clientRemovalChannel:
			delete(h.clients, conn)
		case msg := <-h.broadcastMessage:
			for conn := range h.clients {
				err := conn.WriteJSON(msg)
				if err != nil {
					log.Printf("Error broadcast message to connection %v : %v", conn, err)
				}
			}
		}
	}
}

// upgrade http protocol to websocket protocol
func AllowUpgrade(ctx *fiber.Ctx) error {
	if websocket.IsWebSocketUpgrade(ctx) {
		return ctx.Next()
	}

	return fiber.ErrUpgradeRequired
}

func SendMessage(h *hub) func(*websocket.Conn) {
	return func(conn *websocket.Conn) {
		defer func() {
			h.clientRemovalChannel <- conn
			_ = conn.Close()
		}()

		name := conn.Query("name", "anonymous")
		h.clientRegisterChannel <- conn

		for {
			msgType, body, err := conn.ReadMessage()
			if err != nil {
				return
			}

			if msgType == websocket.TextMessage {
				h.broadcastMessage <- Message{
					Name:    name,
					Message: string(body),
				}
			}
		}
	}
}

func main() {
	// call hub class
	h := &hub{
		clients:               make(map[*websocket.Conn]bool),
		clientRegisterChannel: make(chan *websocket.Conn),
		clientRemovalChannel:  make(chan *websocket.Conn),
		broadcastMessage:      make(chan Message),
	}

	go h.run()

	app := fiber.New()
	app.Use("/ws", AllowUpgrade)
	app.Use("/ws/chat", websocket.New(SendMessage(h)))
	err := app.Listen(":8000")
	if err != nil {
		log.Fatalf("Error listen websocket port : %v", err)
	}
}
