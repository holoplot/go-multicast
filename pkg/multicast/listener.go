package multicast

import (
	"net"
	"sync"
)

type Listener struct {
	mutex     sync.RWMutex
	ifis      []*net.Interface
	consumers []*Consumer
}

func NewListener(ifis []*net.Interface) *Listener {
	return &Listener{
		ifis:      ifis,
		consumers: make([]*Consumer, 0),
	}
}

func (l *Listener) AddConsumer(addr *net.UDPAddr, cb ConsumerPacketCallback) (*Consumer, error) {
	consumer, err := NewConsumer(addr, l.ifis, cb)
	if err != nil {
		return nil, err
	}

	l.mutex.Lock()
	l.consumers = append(l.consumers, consumer)
	l.mutex.Unlock()

	return consumer, nil
}

func (l *Listener) RemoveConsumer(consumer *Consumer) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	for i, c := range l.consumers {
		if c == consumer {
			// Remove consumer from slice
			l.consumers = append(l.consumers[:i], l.consumers[i+1:]...)
			break
		}
	}

	consumer.Close()
}

func (l *Listener) Close() {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	for _, consumer := range l.consumers {
		consumer.Close()
	}

	l.consumers = make([]*Consumer, 0)
}

func (l *Listener) Interfaces() []*net.Interface {
	return l.ifis
}

func (l *Listener) Consumers() []*Consumer {
	l.mutex.RLock()
	defer l.mutex.RUnlock()

	// Return a copy to avoid external modification
	result := make([]*Consumer, len(l.consumers))
	copy(result, l.consumers)

	return result
}
