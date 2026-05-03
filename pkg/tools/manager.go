package tools

import (
	"fmt"
	"os/exec"
	"sync"
	"time"
)

type Job struct {
	ID        string
	Tool      string
	Target    string
	StartTime time.Time
	Cmd       *exec.Cmd
}

type JobManager struct {
	mu   sync.RWMutex
	jobs map[string]*Job
}

var DefaultJobManager = &JobManager{
	jobs: make(map[string]*Job),
}

func (m *JobManager) Register(tool, target string, cmd *exec.Cmd) string {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Use a shorter, more readable ID for UI
	id := fmt.Sprintf("%s-%d", tool, time.Now().Unix()%10000)
	job := &Job{
		ID:        id,
		Tool:      tool,
		Target:    target,
		StartTime: time.Now(),
		Cmd:       cmd,
	}
	m.jobs[id] = job
	return id
}

func (m *JobManager) Unregister(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.jobs, id)
}

func (m *JobManager) StopJob(id string) error {
	m.mu.RLock()
	job, ok := m.jobs[id]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("job not found: %s", id)
	}

	if job.Cmd != nil && job.Cmd.Process != nil {
		return job.Cmd.Process.Kill()
	}
	return nil
}

func (m *JobManager) ListJobs() []*Job {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var jobs []*Job
	for _, job := range m.jobs {
		jobs = append(jobs, job)
	}
	return jobs
}
