package scheduler

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/HaythmKenway/autoscout/internal/db"
	"github.com/HaythmKenway/autoscout/pkg/httpx"
	"github.com/HaythmKenway/autoscout/pkg/localUtils"
	"github.com/HaythmKenway/autoscout/pkg/spider"
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

const (
	FuncSubfinder = "Subfinder"
	FuncHTTPX     = "HTTPX"
	FuncGoSpider  = "GoSpider"
	FuncDalFox    = "DalFox"

	// Config
	InactivityTimeout = 20 * time.Second
)

var (
	schedulerMu sync.Mutex
	running     = false
	TaskQueue   = make(chan Task, 100)
	cancel      context.CancelFunc
	wg          sync.WaitGroup

	CurrentActiveTasks ActiveTasks
	globalTaskID       int64
)

// --- Inactivity Monitor (New Feature) ---

func monitorInactivity(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	lastActivity := time.Now()

	localUtils.Logger("Inactivity monitor started (Timeout: 20s)", 1)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// 1. Check Active Workers
			CurrentActiveTasks.mu.RLock()
			busyWorkers := len(CurrentActiveTasks.tasks)
			CurrentActiveTasks.mu.RUnlock()

			// 2. Check Pending Queue
			queueSize := len(TaskQueue)

			// 3. Reset or Check Timeout
			if busyWorkers > 0 || queueSize > 0 {
				lastActivity = time.Now()
			} else {
				if time.Since(lastActivity) > InactivityTimeout {
					localUtils.Logger("No activity for 20 seconds. Auto-stopping scheduler...", 1)
					go stopScheduler()
					return
				}
			}
		}
	}
}

func addToQueue(ctx context.Context) {
	defer wg.Done()

	// Run immediately on start
	processBatch(ctx)

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			localUtils.Logger("Scheduler stopped", 1)
			return
		case <-ticker.C:
			processBatch(ctx)
		}
	}
}

func processBatch(ctx context.Context) {
	dbConn, err := db.OpenDatabase()
	if err != nil {
		localUtils.Logger(fmt.Sprintf("Scheduler DB Error: %v", err), 1)
		return
	}

	targets, err := db.GetTargetsFromTable(dbConn, 1)
	dbConn.Close()

	if err != nil {
		localUtils.Logger(fmt.Sprintf("Error getting targets: %v", err), 1)
		return
	}

	count := 0
	for _, target := range targets {
		id := atomic.AddInt64(&globalTaskID, 1)
		task := Task{ID: id, Target: target, Title: target}

		select {
		case TaskQueue <- task:
			count++
		case <-ctx.Done():
			return
		}
	}
	if count > 0 {
		localUtils.Logger(fmt.Sprintf("Enqueued %d targets", count), 1)
	}
}

// --- Worker ---

func executeJob(ctx context.Context, workerID int) {
	defer wg.Done()

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
			localUtils.Logger(fmt.Sprintf("[Worker %d] Processing: %s", workerID, task.Target), 1)

			// Add to Active
			CurrentActiveTasks.mu.Lock()
			CurrentActiveTasks.tasks = append(CurrentActiveTasks.tasks, task)
			CurrentActiveTasks.mu.Unlock()

			// Workflow Logic
			pathID, err := determinePath(workerDb, task.Target)
			if err != nil {
				localUtils.Logger(fmt.Sprintf("[Worker %d] No rule for %s: %v", workerID, task.Target, err), 2)
			} else {
				if err := executePath(workerDb, pathID, task.Target); err != nil {
					localUtils.Logger(fmt.Sprintf("[Worker %d] Execution failed: %v", workerID, err), 2)
				}
			}

			// Mark Done
			db.ScanCompleted(workerDb, task.Target)

			// Remove from Active
			CurrentActiveTasks.mu.Lock()
			for i, t := range CurrentActiveTasks.tasks {
				if t.ID == task.ID {
					CurrentActiveTasks.tasks = append(CurrentActiveTasks.tasks[:i], CurrentActiveTasks.tasks[i+1:]...)
					break
				}
			}
			CurrentActiveTasks.mu.Unlock()
		}
	}
}

// --- Dynamic Workflow Logic ---

func determinePath(dbConn *sql.DB, target string) (int, error) {
	query := `SELECT rule_name, match_type, match_criteria, target_path_id FROM branching_rules ORDER BY priority ASC`
	rows, err := dbConn.Query(query)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var rName, mType, mCriteria string
		var pathID int
		if err := rows.Scan(&rName, &mType, &mCriteria, &pathID); err != nil {
			continue
		}

		matched := false
		switch mType {
		case "REGEX":
			matched, _ = regexp.MatchString(mCriteria, target)
		case "EXACT":
			matched = (target == mCriteria)
		case "TYPE":
			matched = strings.Contains(target, mCriteria)
		}

		if matched {
			return pathID, nil
		}
	}
	return 0, fmt.Errorf("no match found")
}

func executePath(dbConn *sql.DB, pathID int, initialTarget string) error {
	query := `
		SELECT f.func_name, i.input_source, i.args 
		FROM proc_path_items i
		JOIN proc_funcs f ON i.proc_func_id = f.proc_func_id
		WHERE i.proc_path_id = ?
		ORDER BY i.exec_order ASC
	`
	rows, err := dbConn.Query(query, pathID)
	if err != nil {
		return err
	}
	defer rows.Close()

	type Step struct {
		FuncName, InputSource, Args string
	}
	var steps []Step
	for rows.Next() {
		var s Step
		if err := rows.Scan(&s.FuncName, &s.InputSource, &s.Args); err == nil {
			steps = append(steps, s)
		}
	}

	currentPayload := []string{initialTarget}

	for _, step := range steps {
		localUtils.Logger(fmt.Sprintf(" -> Running %s", step.FuncName), 1)

		var inputTargets []string
		if step.InputSource == "USER_INPUT" {
			inputTargets = []string{initialTarget}
		} else {
			inputTargets = currentPayload
		}

		output, err := runTool(dbConn, step.FuncName, inputTargets, step.Args)
		if err != nil {
			return err
		}
		if len(output) > 0 {
			currentPayload = output
		}
	}
	return nil
}

func runTool(dbConn *sql.DB, funcName string, targets []string, args string) ([]string, error) {
	var results []string

	for _, target := range targets {
		switch funcName {
		case FuncSubfinder:
			db.SubdomainEnum(target)
			subs, _ := db.GetSubsFromTable(dbConn, target)
			results = append(results, subs...)

		case FuncHTTPX:
			httpx.Httpx(dbConn, target)
			urls, _ := db.GetDataFromTable(dbConn, target)
			results = append(results, urls...)

		case FuncGoSpider:
			res, err := spider.Spider(target)
			if err == nil {
				db.AddSpiderTargets(dbConn, target, res)
				results = append(results, res...)
			}
		}
	}
	return localUtils.RemoveDuplicates(results), nil
}

// --- Scheduler Control ---

// IsRunning allows the UI to check state without managing it manually
func IsRunning() bool {
	schedulerMu.Lock()
	defer schedulerMu.Unlock()
	return running
}

func startScheduler() {
	schedulerMu.Lock()
	defer schedulerMu.Unlock()

	if running {
		return
	}

	var ctx context.Context
	ctx, cancel = context.WithCancel(context.Background())
	running = true
	localUtils.Logger("Started the Scheduler", 1)

	// Start Workers
	for i := 1; i <= 4; i++ {
		wg.Add(1)
		go executeJob(ctx, i)
	}

	// Start Producer
	wg.Add(1)
	go addToQueue(ctx)

	// Start Inactivity Monitor
	go monitorInactivity(ctx)
}

func stopScheduler() {
	schedulerMu.Lock()
	defer schedulerMu.Unlock()

	if !running {
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
