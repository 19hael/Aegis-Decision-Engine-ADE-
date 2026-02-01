package scheduler

import (
	"container/heap"
	"context"
	"log/slog"
	"sync"
	"time"
)

// Job represents a scheduled job
type Job struct {
	ID        string
	ExecuteAt time.Time
	Payload   interface{}
	Handler   func(context.Context, interface{}) error
	Index     int // For heap
}

// JobQueue implements a priority queue for jobs
type JobQueue []*Job

func (jq JobQueue) Len() int { return len(jq) }

func (jq JobQueue) Less(i, j int) bool {
	return jq[i].ExecuteAt.Before(jq[j].ExecuteAt)
}

func (jq JobQueue) Swap(i, j int) {
	jq[i], jq[j] = jq[j], jq[i]
	jq[i].Index = i
	jq[j].Index = j
}

func (jq *JobQueue) Push(x interface{}) {
	n := len(*jq)
	job := x.(*Job)
	job.Index = n
	*jq = append(*jq, job)
}

func (jq *JobQueue) Pop() interface{} {
	old := *jq
	n := len(old)
	job := old[n-1]
	old[n-1] = nil
	job.Index = -1
	*jq = old[0 : n-1]
	return job
}

// Scheduler manages scheduled jobs
type Scheduler struct {
	mu       sync.Mutex
	jobs     JobQueue
	logger   *slog.Logger
	running  bool
	stopChan chan struct{}
	wg       sync.WaitGroup
}

// NewScheduler creates a new scheduler
func NewScheduler(logger *slog.Logger) *Scheduler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Scheduler{
		jobs:     make(JobQueue, 0),
		logger:   logger,
		stopChan: make(chan struct{}),
	}
}

// Start starts the scheduler
func (s *Scheduler) Start(ctx context.Context) {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.mu.Unlock()

	s.wg.Add(1)
	go s.run(ctx)
	s.logger.Info("scheduler started")
}

// Stop stops the scheduler
func (s *Scheduler) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	s.mu.Unlock()

	close(s.stopChan)
	s.wg.Wait()
	s.logger.Info("scheduler stopped")
}

// Schedule adds a job to the scheduler
func (s *Scheduler) Schedule(job *Job) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	heap.Push(&s.jobs, job)
	s.logger.Info("job scheduled", "id", job.ID, "execute_at", job.ExecuteAt)
}

// Cancel removes a job by ID
func (s *Scheduler) Cancel(jobID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, job := range s.jobs {
		if job.ID == jobID {
			heap.Remove(&s.jobs, i)
			s.logger.Info("job cancelled", "id", jobID)
			return true
		}
	}
	return false
}

func (s *Scheduler) run(ctx context.Context) {
	defer s.wg.Done()

	timer := time.NewTimer(time.Second)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopChan:
			return
		case <-timer.C:
			s.processDueJobs(ctx)
			timer.Reset(time.Second)
		}
	}
}

func (s *Scheduler) processDueJobs(ctx context.Context) {
	s.mu.Lock()
	now := time.Now()

	for s.jobs.Len() > 0 {
		job := s.jobs[0]
		if job.ExecuteAt.After(now) {
			break
		}

		heap.Pop(&s.jobs)
		s.mu.Unlock()

		// Execute job in goroutine
		go s.executeJob(ctx, job)

		s.mu.Lock()
	}
	s.mu.Unlock()
}

func (s *Scheduler) executeJob(ctx context.Context, job *Job) {
	s.logger.Info("executing scheduled job", "id", job.ID)
	
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	if err := job.Handler(ctx, job.Payload); err != nil {
		s.logger.Error("job execution failed", "id", job.ID, "error", err)
	} else {
		s.logger.Info("job completed", "id", job.ID)
	}
}

// GetPendingJobs returns pending jobs (for debugging)
func (s *Scheduler) GetPendingJobs() []*Job {
	s.mu.Lock()
	defer s.mu.Unlock()

	jobs := make([]*Job, len(s.jobs))
	copy(jobs, s.jobs)
	return jobs
}
