# Broadcast

## Overview

- clients websocket-connect to `/register` in order to receive all messages
- anyone may broadcast a message to all of the clients using the `/broadcast` endpoint
- connection status is monitored, so that closed connections are cleaned up as appropriate

## Details

Clients register by establishing a websocket connection.

Messages PUT to `/broadcast` ("PUT .../broadcast?message=<message>") are added to a single channel (messageChannel).
The `sender` goroutine reads that channel and sends the message to each of the clients' `register` goroutine via the client's sendChannel.

Registered clients are stored in a map using the sendChannel for that client's `register` handler as the key.

## Testing

### Manual

Start the server:

```console
$ go run main.go
```

You may manually test by opening the `./index.html` file, which registers with the server to receive all message, then using curl to broadcast a message:

```console
$ curl -v -X PUT http://localhost:8081/broadcast?message=my-message
```

### Automated

```console
$ go test -v ./...
```

This tests the complete life-cycle:

- start the server
- start a client
- register the client with the server
- validate connection to server
- send a message to .../broadcast
- receive the message via websocket
- validate message content

