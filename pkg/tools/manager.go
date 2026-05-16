package tools

import (
	"fmt"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/HaythmKenway/autoscout/pkg/localUtils"
)

type Job struct {
	ID        string
	Tool      string
	Target    string
	StartTime time.Time
	Cmd       *exec.Cmd
	IsKilling bool
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
	m.mu.Lock()
	job, ok := m.jobs[id]
	if ok {
		job.IsKilling = true
	}
	m.mu.Unlock()

	if !ok {
		return fmt.Errorf("job not found: %s", id)
	}

	if job.Cmd != nil && job.Cmd.Process != nil {
		pid := job.Cmd.Process.Pid
		// 1. Try to kill the entire process group (negative PID)
		err := syscall.Kill(-pid, syscall.SIGKILL)
		if err == nil {
			localUtils.Logger(fmt.Sprintf("[JobManager] Killed process group for job %s (PID: %d)", id, pid), 1)
			return nil
		}

		// 2. Fallback: Kill the process directly if PGID kill failed
		localUtils.Logger(fmt.Sprintf("[JobManager] PGID kill failed for job %s: %v. Trying direct PID kill.", id, err), 2)
		if err := job.Cmd.Process.Kill(); err != nil {
			localUtils.Logger(fmt.Sprintf("[JobManager] Direct kill failed for job %s: %v", id, err), 2)
			return err
		}
		localUtils.Logger(fmt.Sprintf("[JobManager] Directly killed job %s (PID: %d)", id, pid), 1)
		return nil
	}
	
	localUtils.Logger(fmt.Sprintf("[JobManager] Job %s has no active process to kill", id), 2)
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
