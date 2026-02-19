package yeelight

import (
	"context"
	"sync"
)

// notificationHandler manages the mapping between command IDs and their corresponding response channels.
type notificationHandler struct {
	mu      sync.RWMutex
	baseID  int
	pending map[int]chan Response
}

func newNotifier() *notificationHandler {
	return &notificationHandler{
		pending: make(map[int]chan Response),
	}
}

// genID generates a unique command ID for each command sent to the device. It ensures that the ID does not exceed a predefined maximum value.
func (n *notificationHandler) genID() int {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.baseID > maxId {
		n.baseID = 0
	}
	n.baseID++
	return n.baseID
}

// push sends a response to the corresponding channel based on the command ID.
func (n *notificationHandler) push(resp Response) {
	if resp.ID == 0 {
		return
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	// ensure channel exists for the given ID
	if n.pending[resp.ID] == nil {
		n.pending[resp.ID] = make(chan Response)
	}
	// if there are already pending responses for this ID, close the existing channel and create a new one to avoid blocking
	if len(n.pending[resp.ID]) > 0 {
		close(n.pending[resp.ID])
		n.pending[resp.ID] = make(chan Response)
	}
	n.pending[resp.ID] <- resp
}

// wait waits for a response corresponding to the given command ID. It returns the response and a boolean indicating whether the response was received within the timeout period.
func (n *notificationHandler) wait(ctx context.Context, ID int) (*Response, bool) {
	ctx, cancel := context.WithTimeout(ctx, respWaitTimeout)
	defer cancel()
	n.mu.RLock()
	ch, exists := n.pending[ID]
	if !exists {
		ch = make(chan Response)
		n.pending[ID] = ch
	}
	n.mu.RUnlock()
	select {
	case resp := <-ch:
		return &resp, true
	case <-ctx.Done():
		return nil, false
	}
}
