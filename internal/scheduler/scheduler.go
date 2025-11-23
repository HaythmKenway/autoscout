package scheduler

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/HaythmKenway/autoscout/internal/db"
	"github.com/HaythmKenway/autoscout/pkg/httpx"
	"github.com/HaythmKenway/autoscout/pkg/localUtils"
)

type Task struct {
	ID     int64
	Target string
	Title  string
}

type ActiveTasks struct {
	mu    sync.RWMutex
	tasks []Task
}

var (
	schedulerMu sync.Mutex
	running     = false
	TaskQueue   = make(chan Task, 100)
	cancel      context.CancelFunc
	wg          sync.WaitGroup

	CurrentActiveTasks ActiveTasks
	globalTaskID       int64
)

func addToQueue(ctx context.Context) {
	defer wg.Done()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			localUtils.Logger("Producer stopped", 1)
			return
		case <-ticker.C:
			// 1. Open Connection strictly for this tick
			dbConn, err := db.OpenDatabase()
			if err != nil {
				localUtils.Logger(fmt.Sprintf("Producer DB Error: %v", err), 1)
				continue
			}

			// 2. Get Targets
			targets, err := db.GetTargetsFromTable(dbConn, 1)

			// 3. Close immediately (don't wait for loop end)
			dbConn.Close()

			if err != nil {
				localUtils.Logger(fmt.Sprintf("Error getting targets: %v", err), 1)
				continue
			}

			count := 0
			for _, target := range targets {
				id := atomic.AddInt64(&globalTaskID, 1)
				task := Task{ID: id, Target: target, Title: target}

				select {
				case TaskQueue <- task:
					count++
				case <-ctx.Done():
					localUtils.Logger("Producer stopped while queuing", 1)
					return
				}
			}
			localUtils.Logger(fmt.Sprintf("Enqueued %d tasks", count), 1)
		}
	}
}

func executeJob(ctx context.Context, workerID int) {
	defer wg.Done()

	// 1. Worker gets its own DB connection to prevent locking/exhaustion
	workerDb, err := db.OpenDatabase()
	if err != nil {
		localUtils.Logger(fmt.Sprintf("Worker %d failed to open DB: %v", workerID, err), 1)
		return
	}
	defer workerDb.Close()

	for {
		select {
		case <-ctx.Done():
			localUtils.Logger(fmt.Sprintf("Worker %d stopped", workerID), 1)
			return

		case task := <-TaskQueue:
			localUtils.Logger(fmt.Sprintf("Task id %d Task %s by worker %d", task.ID, task.Target, workerID), 1)

			// 2. Add to Active List (Thread Safe)
			CurrentActiveTasks.mu.Lock()
			CurrentActiveTasks.tasks = append(CurrentActiveTasks.tasks, task)
			CurrentActiveTasks.mu.Unlock()

			// 3. Execute Logic
			// CRITICAL FIX: Pass the workerDb connection to Httpx
			httpx.Httpx(workerDb, task.Target)

			// Update DB using worker's connection
			if err := db.ScanCompleted(workerDb, task.Target); err != nil {
				localUtils.Logger(fmt.Sprintf("Error updating scan time: %v", err), 1)
			}

			// 4. Remove from Active List (Thread Safe)
			CurrentActiveTasks.mu.Lock()
			for i, t := range CurrentActiveTasks.tasks {
				if t.ID == task.ID {
					// Remove the task by index
					CurrentActiveTasks.tasks = append(CurrentActiveTasks.tasks[:i], CurrentActiveTasks.tasks[i+1:]...)
					break
				}
			}
			CurrentActiveTasks.mu.Unlock()
		}
	}
}

func startScheduler() {
	schedulerMu.Lock()
	defer schedulerMu.Unlock()

	if running {
		localUtils.Logger("Scheduler is already running", 1)
		return
	}

	var ctx context.Context
	ctx, cancel = context.WithCancel(context.Background())

	running = true
	localUtils.Logger("Started the Scheduler", 1)

	// Add workers
	for i := 1; i <= 4; i++ {
		wg.Add(1)
		go executeJob(ctx, i)
	}

	// Add producer
	wg.Add(1)
	go addToQueue(ctx)
}

func stopScheduler() {
	schedulerMu.Lock()
	defer schedulerMu.Unlock()

	if !running {
		localUtils.Logger("Scheduler is already stopped", 1)
		return
	}
	localUtils.Logger("Stopping the Scheduler...", 1)

	if cancel != nil {
		cancel()
	}

	wg.Wait()
	running = false
	localUtils.Logger("Scheduler stopped cleanly", 1)
}

func Skibbidi(start bool) {
	if start {
		startScheduler()
	} else {
		stopScheduler()
	}
}
