package conductor

import (
	"time"
)

// DeDuplication is the based interface for handling deduplication of messages.
// Use this for ensuring messages aren't processed twice in the hub.
type DeDuplication interface {
	Start()
	Add(message *Message)
	Remove(message *Message)
	IsDuplicate(message *Message) bool
}

// StandardDeDuplication is the default implmentation of DeDuplication.
// It works by holding the message in memory for a period of time waiting to see if a duplication will arrive.
// If durablity is enabled for the message it will be removed as soon as a message is fin'ed.
type StandardDeDuplication struct {
	timestamps map[string]time.Time
	ttl        time.Duration //ttl is Time To Live in the timestamp list. A good default value for this is X seconds.
	ticker     *time.Ticker
}

// NewDeDuper creates a StandardDeDuplication to use.
// tick is how often the cleanup sweep should happen.
// ttl is how long a message should live in the deduper cache.
func NewDeDuper(tick, ttl time.Duration) *StandardDeDuplication {
	return &StandardDeDuplication{timestamps: make(map[string]time.Time),
		ticker: time.NewTicker(ttl),
		ttl:    ttl}
}

// Start kicks off the ticker so it can do a sweep based on the ttl and cleanup any stale messages
// that didn't get purged (this is much more likely with messages that aren't durable).
func (deduper *StandardDeDuplication) Start() {
	go deduper.doTick()
}

// Add puts a timestamp in the timestamps map based on the message's ID.
// The message will then be checked in the ticker's clean up sweep to remove the message if it is past the ttl.
func (deduper *StandardDeDuplication) Add(message *Message) {
	deduper.timestamps[message.ID] = time.Now()
}

// Remove removes a message based on the ID of the message from the timestamp map.
func (deduper *StandardDeDuplication) Remove(message *Message) {
	delete(deduper.timestamps, message.ID)
}

// IsDuplicate checks to see if the message has a duplicate id of any of the messages stored in the timestamp map.
func (deduper *StandardDeDuplication) IsDuplicate(message *Message) bool {
	_, exist := deduper.timestamps[message.ID]
	return exist
}

func (deduper *StandardDeDuplication) doTick() {
	defer func() {
		deduper.ticker.Stop()
	}()

	for { // blocking loop with select to wait for stimulation.
		select {
		case <-deduper.ticker.C:
			deduper.cleanupSweep()
		}
	}
}

func (deduper *StandardDeDuplication) cleanupSweep() {
	now := time.Now()
	for key := range deduper.timestamps {
		start := deduper.timestamps[key]
		elapsed := now.Sub(start)
		if elapsed > deduper.ttl {
			delete(deduper.timestamps, key)
		}
	}
}
