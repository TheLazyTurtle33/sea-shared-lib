package obs

import (
	"fmt"
	"sync/atomic"
)

var reqCounter atomic.Uint64

func newRequestID() string {
	return fmt.Sprintf("req-%d", reqCounter.Add(1))
}

func (c *Client) SetScene(scene string) (map[string]any, error) {
	return c.Send("SetCurrentProgramScene", map[string]any{
		"sceneName": scene,
	})
}
