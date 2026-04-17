package obs

import (
	"fmt"
	"net/url"
	"sync"
	"time"

	"golang.org/x/net/websocket"
)

// -- types --

type logIn struct {
	Url      string
	Port     string
	Password string
}

type Client struct {
	conn    *websocket.Conn
	mu      sync.Mutex               // guards conn and pending
	pending map[string]chan response // requestId -> response channel
}

type request struct {
	OpCode int         `json:"op"`
	Data   requestData `json:"d"`
}

type requestData struct {
	RequestType   string `json:"requestType"`
	RequestID     string `json:"requestId"`
	RequestStatus any    `json:"requestData,omitempty"`
}

type response struct {
	OpCode int          `json:"op"`
	Data   responseData `json:"d"`
}

type responseData struct {
	RequestType   string         `json:"requestType"`
	RequestID     string         `json:"requestId"`
	RequestStatus requestStatus  `json:"requestStatus"`
	ResponseData  map[string]any `json:"responseData,omitempty"`
}

type requestStatus struct {
	Result  bool   `json:"result"`
	Code    int    `json:"code"`
	Comment string `json:"comment,omitempty"`
}

// helloPayload is what OBS sends on connect (op 0)
type helloPayload struct {
	OpCode int `json:"op"`
	Data   struct {
		ObsWebSocketVersion string `json:"obsWebSocketVersion"`
		RpcVersion          int    `json:"rpcVersion"`
		Authentication      *struct {
			Challenge string `json:"challenge"`
			Salt      string `json:"salt"`
		} `json:"authentication,omitempty"`
	} `json:"d"`
}

// -- singleton --

var instance *Client

func Get() (*Client, error) {
	if instance == nil {
		err := new()
		return instance, err
	}
	return instance, nil
}

// -- init --

func new() error {
	u := url.URL{
		Scheme: "ws",
		Host:   fmt.Sprintf("%s:%s", secret.Url, secret.Port),
	}

	ws, err := websocket.Dial(u.String(), "", "http://localhost/")
	if err != nil {
		instance = nil
		return fmt.Errorf("obs: failed to connect: %w", err)
	}

	c := &Client{
		conn:    ws,
		pending: make(map[string]chan response),
	}

	// read the Hello (op 0) and respond with Identify (op 1)
	if err := c.handshake(); err != nil {
		ws.Close()
		instance = nil
		return fmt.Errorf("obs: handshake failed: %w", err)
	}

	// start the background listener
	go c.listen()

	instance = c
	return nil
}

// -- handshake (op 0 -> op 1) --

func (c *Client) handshake() error {
	var hello helloPayload
	if err := websocket.JSON.Receive(c.conn, &hello); err != nil {
		return fmt.Errorf("failed to read hello: %w", err)
	}

	type identifyData struct {
		RpcVersion     int    `json:"rpcVersion"`
		Authentication string `json:"authentication,omitempty"`
	}
	type identify struct {
		OpCode int          `json:"op"`
		Data   identifyData `json:"d"`
	}

	id := identify{
		OpCode: 1,
		Data:   identifyData{RpcVersion: 1},
	}

	if hello.Data.Authentication != nil {
		auth, err := buildAuth(secret.Password, hello.Data.Authentication.Salt, hello.Data.Authentication.Challenge)
		if err != nil {
			return fmt.Errorf("failed to build auth: %w", err)
		}
		id.Data.Authentication = auth
	}

	if err := websocket.JSON.Send(c.conn, id); err != nil {
		return fmt.Errorf("failed to send identify: %w", err)
	}

	// expect Identified (op 2)
	var identified struct {
		OpCode int `json:"op"`
	}
	if err := websocket.JSON.Receive(c.conn, &identified); err != nil {
		return fmt.Errorf("failed to read identified: %w", err)
	}
	if identified.OpCode != 2 {
		return fmt.Errorf("expected op 2 (Identified), got op %d", identified.OpCode)
	}

	return nil
}

// -- listener goroutine --

func (c *Client) listen() {
	for {
		var resp response
		if err := websocket.JSON.Receive(c.conn, &resp); err != nil {
			// connection closed or broken — clear pending waiters
			c.mu.Lock()
			for id, ch := range c.pending {
				close(ch)
				delete(c.pending, id)
			}
			c.mu.Unlock()
			return
		}

		// op 7 = RequestResponse
		if resp.OpCode == 7 {
			c.mu.Lock()
			ch, ok := c.pending[resp.Data.RequestID]
			if ok {
				ch <- resp
				delete(c.pending, resp.Data.RequestID)
			}
			c.mu.Unlock()
		}
		// op 5 = Event (ignored for now, extend here later)
	}
}

// -- send --

const sendTimeout = 10 * time.Second

// Send sends a request to OBS and blocks until OBS replies or timeout.
// requestType is e.g. "GetVersion", "SetCurrentProgramScene".
// data is the optional request payload (pass nil if none).
func (c *Client) Send(requestType string, data any) (map[string]any, error) {
	id := newRequestID()

	ch := make(chan response, 1)

	c.mu.Lock()
	c.pending[id] = ch
	c.mu.Unlock()

	msg := map[string]any{
		"op": 6, // Request opcode
		"d": map[string]any{
			"requestType": requestType,
			"requestId":   id,
			"requestData": data,
		},
	}

	c.mu.Lock()
	err := websocket.JSON.Send(c.conn, msg)
	c.mu.Unlock()

	if err != nil {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return nil, fmt.Errorf("obs: failed to send %s: %w", requestType, err)
	}

	select {
	case resp, ok := <-ch:
		if !ok {
			return nil, fmt.Errorf("obs: connection closed waiting for %s response", requestType)
		}
		if !resp.Data.RequestStatus.Result {
			return nil, fmt.Errorf("obs: %s failed (code %d): %s",
				requestType,
				resp.Data.RequestStatus.Code,
				resp.Data.RequestStatus.Comment,
			)
		}
		return resp.Data.ResponseData, nil

	case <-time.After(sendTimeout):
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return nil, fmt.Errorf("obs: timed out waiting for %s response", requestType)
	}
}
