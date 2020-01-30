package main

/*

Broadcast server

- clients websocket-connect to `/register` in order to receive all messages
- anyone may broadcast a message to all of the clients using the `/broadcast` endpoint


*/

import (
	"github.com/gorilla/websocket"
	"sync"

	"log"
	"net/http"
)

// registeredClients is a map containing all the currently registeredClients clients (identified with its sendChannel)
var registeredClientsMux sync.RWMutex
var registeredClients map[chan string]struct{}

// messageChannel is a channel that allows the http handler to send messages to all of the sender goroutines via `sender`
var messageChannel chan string

// main
func main() {

	// we're alive!
	log.Print("Starting broadcast server")

	// initialize registeredClients map
	registeredClients = make(map[chan string]struct{})

	// set up the routes / endpoints
	routes()

	// start the sender
	go sender()

	// listen for connections
	err := http.ListenAndServe(":8081", nil)
	if err != nil {
		log.Fatal(err)
	}

}

// routes sets up the endpoints
func routes() {
	http.HandleFunc("/broadcast", broadcast)
	http.HandleFunc("/register", register)
}

// register handles the `register` endpoint
// it lives as long as the client is registered / connected
// kills itself if the client goes away (signalled via deadChannel)
func register(w http.ResponseWriter, r *http.Request) {

	// make a channel that we will use to send messages to this client
	sendChannel := make(chan string)

	log.Print("registering:", sendChannel)

	// set up websocket

	// accept any host
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print(err)
		http.Error(w, "Protocol error", http.StatusBadRequest)
		return
	}

	registeredClientsMux.Lock()
	registeredClients[sendChannel] = struct{}{}
	registeredClientsMux.Unlock()

	// watch for close
	deadChannel := make(chan bool)
	go readClient(ws, sendChannel, deadChannel)

	// welcome the client
	err = ws.WriteMessage(1, []byte("[you are connected]"))
	if err != nil {
		log.Println(err)
	}

	for {
		// wait for a message that needs sent
		select {
		case message := <-sendChannel:
			err := ws.WriteMessage(1, []byte(message))
			if err != nil {
				log.Print(err)
				return
			}
			log.Print("sent message: ", message, " to: ", sendChannel)

		case dead := <-deadChannel:
			if dead {
				log.Print("stopping register handler for: ", sendChannel)
				return
			}
		}
	}

}

// readClient detects close, etc.
func readClient(c *websocket.Conn, sendChannel chan string, dead chan bool) {
	for {
		if _, _, err := c.NextReader(); err != nil { // TODO: confirm that there are no false failure conditions here

			// connection has 'failed'.  get rid of it
			c.Close()

			log.Print("closing: ", sendChannel)

			// remove the client
			registeredClientsMux.Lock()
			delete(registeredClients, sendChannel)
			registeredClientsMux.Unlock()

			// signal the associated register handler to quit
			dead <- true

			break
		}
	}
}

// broadcast sends the provided message to all the registered clients
func broadcast(w http.ResponseWriter, r *http.Request) {

	if r.Method == "PUT" {

		if message := r.URL.Query().Get("message"); len(message) != 0 {
			log.Print("received message: ", r.URL.Query().Get("message"))

			// send the message to the `sender` thread
			messageChannel <- message
		} else {
			http.Error(w, "Missing message", http.StatusBadRequest)
		}

	} else {
		log.Print(r.Method)
		http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
	}

}

// sender sends messages from messageChannel to all the registeredClients
func sender() {

	// create a buffered channel to receive messages that have been received from the `broadcast` endpoint
	messageChannel = make(chan string, 99) // NB: broadcast will start to block when / if this fills up

	// forever
	for {
		// wait for a message
		message := <-messageChannel

		log.Print("sending to all registered clients: ", message)

		// send message to all registered clients
		registeredClientsMux.RLock()
		for sendChannel := range registeredClients {
			log.Print("sending to: ", sendChannel)
			sendChannel <- message
		}
		registeredClientsMux.RUnlock()

	}

}

// websocket things
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}
