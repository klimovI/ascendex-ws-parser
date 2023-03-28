package main

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

const (
	wsURL = "wss://ascendex.com/1/api/pro/v1/stream"
	BBO   = "bbo"
)

var symbolRegex = regexp.MustCompile("^[A-Z]+_[A-Z]+$")

type Client struct {
	conn *websocket.Conn
}

type Response struct {
	M      string `json:"m"`
	Symbol string `json:"symbol"`
	Data   Data   `json:"data"`
}

type Data struct {
	Ts  int64     `json:"ts"`
	Bid [2]string `json:"bid"`
	Ask [2]string `json:"ask"`
}

func (c *Client) Connection() error {
	dialer := websocket.DefaultDialer

	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to websocket: %v", err)
	}

	c.conn = conn

	return nil
}

func (c *Client) Disconnect() {
	if err := c.conn.Close(); err != nil {
		fmt.Printf("failed to close websocket connection: %v\n", err)
	}
}

func (c *Client) SubscribeToChannel(symbol string) error {
	if !symbolRegex.MatchString(symbol) {
		return fmt.Errorf(`invalid symbol %s, symbol must be of the form "TOKEN_ASSET"`, symbol)
	}

	symbols := strings.Split(symbol, "_")
	token, asset := symbols[0], symbols[1]
	msg := fmt.Sprintf(`{"op":"sub","ch":"%s:%s/%s"}`, BBO, token, asset)
	data := []byte(msg)

	if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
		c.Disconnect()
		return fmt.Errorf("failed to subscribe to BBO channel: %v", err)
	}

	return nil
}

func (c *Client) ReadMessagesFromChannel(ch chan<- BestOrderBook) {
	go func() {
		for {
			_, msg, err := c.conn.ReadMessage()
			if err != nil {
				c.Disconnect()
				close(ch)
				fmt.Printf("failed to read websocket message: %v\n", err)
				return
			}

			var res Response
			if err := json.Unmarshal(msg, &res); err != nil {
				c.Disconnect()
				close(ch)
				fmt.Printf("failed to unmarshal websocket message: %v\n", err)
				return
			}

			if res.M != BBO {
				continue
			}

			ask, bid := res.Data.Ask, res.Data.Bid
			askPriceStr, askAmountStr := ask[0], ask[1]
			bidPriceStr, bidAmountStr := bid[0], bid[1]

			askPrice, err := strconv.ParseFloat(askPriceStr, 64)
			if err != nil {
				fmt.Printf("invalid ask price: %s\n", askPriceStr)
				continue
			}

			askAmount, err := strconv.ParseFloat(askAmountStr, 64)
			if err != nil {
				fmt.Printf("invalid ask amount: %s\n", askAmountStr)
				continue
			}

			bidPrice, err := strconv.ParseFloat(bidPriceStr, 64)
			if err != nil {
				fmt.Printf("invalid bid price: %s\n", bidPriceStr)
				continue
			}

			bidAmount, err := strconv.ParseFloat(bidAmountStr, 64)
			if err != nil {
				fmt.Printf("invalid bid amount: %s\n", bidAmountStr)
				continue
			}

			bbo := BestOrderBook{
				Ask: Order{
					Amount: askAmount,
					Price:  askPrice,
				},
				Bid: Order{
					Amount: bidAmount,
					Price:  bidPrice,
				},
			}

			ch <- bbo
		}
	}()
}

func (c *Client) WriteMessagesToChannel() {
	data := []byte(`{"op":"ping"}`)

	go func() {
		for range time.Tick(10 * time.Second) {
			err := c.conn.WriteMessage(websocket.PingMessage, data)
			if err != nil {
				c.Disconnect()
				fmt.Printf("failed to ping websocket: %v\n", err)
				return
			}
		}
	}()
}
