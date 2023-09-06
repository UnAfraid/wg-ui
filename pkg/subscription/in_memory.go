package subscription

import (
	"context"
	"path/filepath"
	"sync"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type inMemorySubscription struct {
	observers     map[channelKey]chan []byte
	observersLock sync.RWMutex
}

func NewInMemorySubscription() Subscription {
	return &inMemorySubscription{
		observers:     make(map[channelKey]chan []byte),
		observersLock: sync.RWMutex{},
	}
}

func (s *inMemorySubscription) Notify(bytes []byte, channel string) error {
	s.observersLock.RLock()
	defer s.observersLock.RUnlock()

	channel = joinPath(channel)

	for k, v := range s.observers {
		match, err := filepath.Match(k.channel, channel)
		if err != nil {
			logrus.
				WithError(err).
				WithField("a", k.channel).
				WithField("b", channel).
				Warn("failed to match glob pattern")
		}
		if match {
			v <- bytes
		}
	}
	return nil
}

func (s *inMemorySubscription) Subscribe(ctx context.Context, channel string) (<-chan []byte, error) {
	uuidValue, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}

	s.observersLock.Lock()
	defer s.observersLock.Unlock()

	key := newChannelKey(uuidValue.String(), joinPath(channel))
	observerChan := make(chan []byte)
	go func() {
		<-ctx.Done()
		s.unsubscribe(key, observerChan)
	}()

	s.observers[key] = observerChan
	return observerChan, nil
}

func (s *inMemorySubscription) HasSubscribers(channel string) bool {
	if len(channel) == 0 {
		return len(s.observers) != 0
	}
	channel = joinPath(channel)

	for k := range s.observers {
		match, err := filepath.Match(k.channel, channel)
		if err != nil {
			logrus.
				WithError(err).
				WithField("a", k.channel).
				WithField("b", channel).
				Warn("failed to match glob pattern")
		}
		if match {
			return true
		}
	}
	return false
}

func (s *inMemorySubscription) unsubscribe(key channelKey, observerChan chan<- []byte) {
	s.observersLock.Lock()
	defer s.observersLock.Unlock()

	delete(s.observers, key)
	close(observerChan)
}
