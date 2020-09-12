package accesslog

// LogChan describes the log channel.
// See `Broker` for details.
type LogChan chan Log

// A Broker holds the active listeners,
// incoming logs on its Notifier channel
// and broadcast event data to all registered listeners.
//
// Exports the `NewListener` and `CloseListener` methods.
type Broker struct {
	// Logs are pushed to this channel
	// by the main events-gathering `run` routine.
	Notifier LogChan

	// NewListener action.
	newListeners chan LogChan

	// CloseListener action.
	closingListeners chan LogChan

	// listeners store.
	listeners map[LogChan]bool

	// force-terminate all listeners.
	close chan struct{}
}

// newBroker returns a new broker factory.
func newBroker() *Broker {
	b := &Broker{
		Notifier:         make(LogChan, 1),
		newListeners:     make(chan LogChan),
		closingListeners: make(chan LogChan),
		listeners:        make(map[LogChan]bool),
		close:            make(chan struct{}),
	}

	// Listens and Broadcasts events.
	go b.run()

	return b
}

// run listens on different channels and act accordingly.
func (b *Broker) run() {
	for {
		select {
		case s := <-b.newListeners:
			// A new channel has started to listen.
			b.listeners[s] = true

		case s := <-b.closingListeners:
			// A listener has dettached.
			// Stop sending them the logs.
			delete(b.listeners, s)

		case log := <-b.Notifier:
			// A new log sent by the logger.
			// Send it to all active listeners.
			for clientMessageChan := range b.listeners {
				clientMessageChan <- log
			}

		case <-b.close:
			for clientMessageChan := range b.listeners {
				delete(b.listeners, clientMessageChan)
				close(clientMessageChan)
			}
		}
	}
}

// notify sends the "log" to all active listeners.
func (b *Broker) notify(log Log) {
	b.Notifier <- log
}

// NewListener returns a new log channel listener.
// The caller SHALL NOT use this to write logs.
func (b *Broker) NewListener() LogChan {
	// Each listener registers its own message channel with the Broker's connections registry.
	logs := make(LogChan)
	// Signal the broker that we have a new listener.
	b.newListeners <- logs
	return logs
}

// CloseListener removes the "ln" listener from the active listeners.
func (b *Broker) CloseListener(ln LogChan) {
	b.closingListeners <- ln
}

// As we cant export a read-only and pass it as closing client
// we will return a read-write channel on NewListener and add a note that the user
// should NOT send data back to the channel, its use is read-only.
// func (b *Broker) CloseListener(ln <-chan *Log) {
// 	b.closingListeners <- ln
// }
