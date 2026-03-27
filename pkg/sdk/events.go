package sdk

import (
	"sync"
)

// EventType categorizes system events sent to the GUI.
type EventType string

const (
	EventVFSMounted   EventType = "vfs_mounted"
	EventVFSUnmounted EventType = "vfs_unmounted"
	EventPeerJoined   EventType = "peer_joined"
	EventPeerLeft     EventType = "peer_left"
	EventSyncProg     EventType = "sync_progress"
	EventAlertKill    EventType = "sovereignty_kill" // Watchdog Triggered
)

// SystemEvent is the standard IPC message struct broadcasted to the GUI.
type SystemEvent struct {
	Type    EventType   `json:"type"`
	Payload interface{} `json:"payload"`
}

// EventBus is the central nervous system connecting Backend <-> GUI.
type EventBus struct {
	mu          sync.RWMutex
	subscribers map[EventType][]chan SystemEvent
}

func NewEventBus() *EventBus {
	return &EventBus{
		subscribers: make(map[EventType][]chan SystemEvent),
	}
}

// Subscribe returns a channel that receives events of a specific type.
func (b *EventBus) Subscribe(event EventType) <-chan SystemEvent {
	b.mu.Lock()
	defer b.mu.Unlock()

	ch := make(chan SystemEvent, 100) // Buffer to prevent GUI blocking
	b.subscribers[event] = append(b.subscribers[event], ch)
	return ch
}

// Emit broadcasts an event to all interested subscribers (e.g the Wails/Tauri IPC bridge).
func (b *EventBus) Emit(event EventType, payload interface{}) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	msg := SystemEvent{
		Type:    event,
		Payload: payload,
	}

	for _, ch := range b.subscribers[event] {
		select {
		case ch <- msg: // Non-blocking send
		default:
			// If channel buffer is full, drop or log (GUI parsing too slow)
		}
	}
}
