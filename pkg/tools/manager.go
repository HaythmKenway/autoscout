package tools

import (
	"fmt"
	"os/exec"
	"sync"
	"syscall"
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
	sem  chan struct{}
}

var DefaultJobManager = &JobManager{
	jobs: make(map[string]*Job),
	sem:  make(chan struct{}, 5), // Limit to 5 concurrent tools
}

func (m *JobManager) Register(tool, target string, cmd *exec.Cmd) string {
	// Wait for slot in semaphore
	m.sem <- struct{}{}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Ensure the command starts in its own process group
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	} else {
		cmd.SysProcAttr.Setpgid = true
	}

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
	delete(m.jobs, id)
	m.mu.Unlock()

	// Release slot in semaphore
	<-m.sem
}

func (m *JobManager) StopJob(id string) error {
	m.mu.RLock()
	job, ok := m.jobs[id]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("job not found: %s", id)
	}

	if job.Cmd != nil && job.Cmd.Process != nil {
		// Kill the entire process group (negative PID)
		return syscall.Kill(-job.Cmd.Process.Pid, syscall.SIGKILL)
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
