package scheduler

import (
	"github.com/robfig/cron/v3"
)

// Scheduler wraps the cron instance.
type Scheduler struct {
	cron *cron.Cron
}

// NewScheduler creates a new scheduler.
func NewScheduler() *Scheduler {
	return &Scheduler{
		cron: cron.New(cron.WithSeconds()), // WithSeconds allows for more granular scheduling if needed.
	}
}

// AddFunc adds a func to the Cron to be executed on the given schedule.
func (s *Scheduler) AddFunc(spec string, cmd func()) (cron.EntryID, error) {
	return s.cron.AddFunc(spec, cmd)
}

// Start starts the cron scheduler.
func (s *Scheduler) Start() {
	s.cron.Start()
}

// Stop stops the cron scheduler.
func (s *Scheduler) Stop() {
	s.cron.Stop()
}
