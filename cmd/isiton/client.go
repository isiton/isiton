// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"log"
	//	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	flushPeriod    = 1 * time.Second
	minFlushPeriod = 3 * time.Second

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// Client is an middleman between the websocket connection and the hub.
type Client struct {
	hub *Hub

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan interface{}
}

// readPump pumps messages from the websocket connection to the hub.
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				log.Printf("error: %v", err)
			}
			break
		}
		message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))
		// c.hub.broadcast <- message
	}
}

// write writes a message with the given message type and payload.
func (c *Client) write(mt int, payload []byte) error {
	c.conn.SetWriteDeadline(time.Now().Add(writeWait))
	return c.conn.WriteMessage(mt, payload)
}

// writePump pumps messages from the hub to the websocket connection.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	tickerFlush := time.NewTicker(flushPeriod)
	defer func() {
		tickerFlush.Stop()
		ticker.Stop()
		c.conn.Close()
	}()

	sendMessages := make([]interface{}, 16)
	sendMessagesIndex := 0

	lastFlushed := time.Now()

	writeMessage := func(message interface{}) {
		c.conn.SetWriteDeadline(time.Now().Add(writeWait))
		c.conn.WriteJSON(message)
	}

	flushMessages := func() {
		if sendMessagesIndex > 0 {
			writeMessage(sendMessages[:sendMessagesIndex])
			sendMessagesIndex = 0
		}
		lastFlushed = time.Now()
	}

	// flush buffer of updates O(N)*1 message, instead of O(1)*N messages
	addMessage := func(message interface{}) {
		sendMessages[sendMessagesIndex] = message
		sendMessagesIndex++
		if sendMessagesIndex == cap(sendMessages) {
			flushMessages()
		}
	}

	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				// The hub closed the channel.
				// log.Println("Hub closed channel")
				c.write(websocket.CloseMessage, []byte{})
				return
			}

			addMessage(message)
			for len(c.send) > 0 {
				addMessage(<-c.send)
			}

		case <-tickerFlush.C:
			if lastFlushed.Add(minFlushPeriod).Before(time.Now()) {
				//log.Println("Flushing messages after minFlushPeriod")
				flushMessages()
			}
		case <-ticker.C:
			if err := c.write(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		}
	}
}

// serveWs handles websocket requests from the peer.
func serveWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	client := &Client{hub: hub, conn: conn, send: make(chan interface{}, 256)}
	client.hub.register <- client
	go client.writePump()
	client.readPump()
}
