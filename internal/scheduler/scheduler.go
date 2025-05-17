package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/HaythmKenway/autoscout/internal/db"
	"github.com/HaythmKenway/autoscout/pkg/httpx"
	"github.com/HaythmKenway/autoscout/pkg/localUtils"
)

type Task struct {
	ID     int
	Target string
	Title  string
}

var (
	running   = false
	TaskQueue = make(chan Task, 100)
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	CurrentActiveTasks ActiveTasks
)

type ActiveTasks struct {
	tasks[] Task
}
func addToQueue(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			localUtils.Logger("Producer stopped", 1)
			return
		case <-ticker.C:
			targets, _ := db.GetTargetsFromTable(1)
			for i, target := range targets {
				task := Task{ID: i, Target: target,Title: target}
				TaskQueue <- task
			}
			localUtils.Logger(fmt.Sprintf("Enqueued %d tasks", len(targets)), 1)
		}
	}
}

func executeJob(ctx context.Context, workerID int) {
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			localUtils.Logger(fmt.Sprintf("Worker %d stopped", workerID), 1)
			return

		case task := <-TaskQueue:
			localUtils.Logger(fmt.Sprintf("Task id %d Task %s by worker %d", task.ID, task.Target, workerID), 1)
			CurrentActiveTasks.tasks = append(CurrentActiveTasks.tasks, task)
			httpx.Httpx(task.Target)
			db.ScanCompleted(task.Target)
			for i, t := range CurrentActiveTasks.tasks {
				if t.ID == task.ID { 
					CurrentActiveTasks.tasks = append(CurrentActiveTasks.tasks[:i], CurrentActiveTasks.tasks[i+1:]...)
					break
				}
			}
		}
	}
}

func startScheduler() {
	if running {
		localUtils.Logger("Scheduler is already running", 1)
		return
	}
	running = true
	localUtils.Logger("Started the Scheduler", 1)

	var ctx context.Context
	ctx, cancel = context.WithCancel(context.Background())

	for i := 1; i <= 4; i++ {
		wg.Add(1)
		go executeJob(ctx, i)
	}

	go addToQueue(ctx)
}

func stopScheduler() {
	if !running {
		localUtils.Logger("Scheduler is already stopped", 1)
		return
	}
	localUtils.Logger("Stopping the Scheduler", 1)

	cancel()  
	wg.Wait() 
	running = false

	localUtils.Logger("Scheduler stopped", 1)
}

func Skibbidi(start bool) {
	if !running && start {
		startScheduler()
	} else if running && !start {
		stopScheduler()
	}
}

