package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// wsClient listens for broadcast messages and sends them on the `receive` channel
func wsClient(receive chan string) {

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	addr := "localhost:8081"
	u := url.URL{Scheme: "ws", Host: addr, Path: "/register"}
	log.Printf("connecting to %s", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()

	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}
			log.Printf("recv: %s", message)
			receive <- string(message)
		}
	}()

	for {
		select {
		case <-done:
			return
		case <-interrupt:
			log.Println("interrupt")
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			return
		}
	}
}

// TestCloseTheLoop starts the server, a client, registers, sends a messages, and tests that the message received by the client matches the one that was sent
func TestCloseTheLoop(t *testing.T) {

	// start the client (register)
	receive := make(chan string)
	go wsClient(receive)

	r1 := <-receive
	expected := "[you are connected]"
	if r1 != expected {
		t.Errorf("Expected '%s', got: '%s'", expected, r1)
	}

	// send a message
	doBroadcast("http://localhost:8081/broadcast?message=a-message")

	r2 := <-receive
	expected = "a-message"
	if r2 != "a-message" {
		t.Errorf("Expected '%s', got: '%s'", expected, r2)
	}

}

var receive chan string

func TestMain(m *testing.M) {

	// start the server
	go main()

	os.Exit(m.Run())

}

func doBroadcast(url string) {
	client := &http.Client{}
	request, err := http.NewRequest("PUT", url, strings.NewReader(""))
	response, err := client.Do(request)
	if err != nil {
		log.Fatal(err)
	} else {
		defer response.Body.Close()
		_, err := ioutil.ReadAll(response.Body)
		if err != nil {
			log.Fatal(err)
		}
	}
}
