package obs

import (
	"fmt"
	"sync/atomic"
)

var reqCounter atomic.Uint64

func newRequestID() string {
	return fmt.Sprintf("req-%d", reqCounter.Add(1))
}
