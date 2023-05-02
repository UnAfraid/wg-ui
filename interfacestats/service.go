package interfacestats

import (
	"context"
	"sync"
	"time"

	"github.com/UnAfraid/wg-ui/api/model"
	"github.com/UnAfraid/wg-ui/api/subscription"
	"github.com/UnAfraid/wg-ui/server"
	"github.com/UnAfraid/wg-ui/wg"
	"github.com/sirupsen/logrus"
)

const (
	updateStatsInterval = time.Second
)

type Service interface {
	Close()
}

type service struct {
	wgService                  wg.Service
	serverService              server.Service
	serverSubscriptionService  subscription.Service[*model.ServerChangedEvent]
	previousInterfaceStats     map[string]*wg.InterfaceStats
	previousInterfaceStatsLock sync.RWMutex
	stopChan                   chan struct{}
	stoppedChan                chan struct{}
}

func NewService(
	wgService wg.Service,
	serverService server.Service,
	serverSubscriptionService subscription.Service[*model.ServerChangedEvent],
) Service {
	s := &service{
		wgService:                 wgService,
		serverService:             serverService,
		serverSubscriptionService: serverSubscriptionService,
		previousInterfaceStats:    make(map[string]*wg.InterfaceStats),
		stopChan:                  make(chan struct{}),
		stoppedChan:               make(chan struct{}),
	}

	go s.runPeriodicTasks()

	return s
}

func (s *service) runPeriodicTasks() {
	defer close(s.stoppedChan)
	for {
		select {
		case <-s.stopChan:
			return
		case <-time.After(updateStatsInterval):
			s.updateStats()
		}
	}
}

func (s *service) updateStats() {
	if !s.serverSubscriptionService.HasSubscribers() {
		return
	}

	servers, err := s.serverService.FindServers(context.Background(), &server.FindOptions{})
	if err != nil {
		logrus.
			WithError(err).
			Error("failed to find servers")
		return
	}

	for _, svc := range servers {
		if !svc.Enabled || !svc.Running {
			continue
		}

		interfaceStats, err := s.wgService.InterfaceStats(svc.Name)
		if err != nil {
			logrus.
				WithError(err).
				WithField("name", svc.Name).
				Error("failed to get interface stats")
			continue
		}
		if interfaceStats == nil {
			continue
		}

		previousInterfaceStats := s.getPreviousInterfaceStats(svc.Name)
		if previousInterfaceStats == nil || *interfaceStats != *previousInterfaceStats {
			s.setPreviousInterfaceStats(svc.Name, interfaceStats)

			err = s.serverSubscriptionService.Notify(&model.ServerChangedEvent{
				Node:   model.ToServer(svc),
				Action: model.ServerActionInterfaceStatsUpdated,
			})
			if err != nil {
				logrus.
					WithError(err).
					WithField("name", svc.Name).
					Error("failed notify server interface stats updated")
				continue
			}
		}
	}
}

func (s *service) getPreviousInterfaceStats(name string) *wg.InterfaceStats {
	s.previousInterfaceStatsLock.RLock()
	defer s.previousInterfaceStatsLock.RUnlock()
	return s.previousInterfaceStats[name]
}

func (s *service) setPreviousInterfaceStats(name string, stats *wg.InterfaceStats) {
	s.previousInterfaceStatsLock.Lock()
	defer s.previousInterfaceStatsLock.Unlock()
	s.previousInterfaceStats[name] = stats
}

func (s *service) Close() {
	close(s.stopChan)
	<-s.stoppedChan
}
