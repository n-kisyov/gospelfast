package admin

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log"
	"sync"
	"time"

	importpkg "github.com/gospelfast/gospelfast/internal/import"
)

type JobStatus string

const (
	StatusPending    JobStatus = "pending"
	StatusProcessing JobStatus = "processing"
	StatusComplete   JobStatus = "complete"
	StatusFailed     JobStatus = "failed"
)

type Job struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Source    string    `json:"source"`
	Status    JobStatus `json:"status"`
	Progress  int       `json:"progress"`
	Total     int       `json:"total"`
	Stage     string    `json:"stage"`
	Error     string    `json:"error,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type JobManager struct {
	mu       sync.Mutex
	jobs     map[string]*Job
	pipeline *importpkg.Pipeline
}

func NewJobManager(p *importpkg.Pipeline) *JobManager {
	return &JobManager{
		jobs:     make(map[string]*Job),
		pipeline: p,
	}
}

func (m *JobManager) Start(source, name, format string) *Job {
	id := generateID()

	job := &Job{
		ID:        id,
		Name:      name,
		Source:    source,
		Status:    StatusPending,
		CreatedAt: time.Now(),
	}

	m.mu.Lock()
	m.jobs[id] = job
	snapshot := *job
	m.mu.Unlock()

	go m.run(context.Background(), job, format)

	return &snapshot
}

// Get returns a point-in-time copy of the job so callers can read its
// fields without racing the background goroutine in run(), which mutates
// the shared *Job under m.mu.
func (m *JobManager) Get(id string) *Job {
	m.mu.Lock()
	defer m.mu.Unlock()
	j, ok := m.jobs[id]
	if !ok {
		return nil
	}
	snapshot := *j
	return &snapshot
}

// List returns point-in-time copies of all jobs; see Get.
func (m *JobManager) List() []*Job {
	m.mu.Lock()
	defer m.mu.Unlock()
	jobs := make([]*Job, 0, len(m.jobs))
	for _, j := range m.jobs {
		snapshot := *j
		jobs = append(jobs, &snapshot)
	}
	return jobs
}

func (m *JobManager) run(ctx context.Context, job *Job, format string) {
	m.mu.Lock()
	job.Status = StatusProcessing
	m.mu.Unlock()

	var f importpkg.Format
	switch format {
	case "rawzip":
		f = importpkg.FormatRawzip
	case "osis":
		f = importpkg.FormatOSIS
	case "vpl":
		f = importpkg.FormatVPL
	default:
		job.Status = StatusFailed
		job.Error = "unknown format: " + format
		return
	}

	progress := func(stage string, current, total int) {
		m.mu.Lock()
		job.Stage = stage
		job.Progress = current
		job.Total = total
		m.mu.Unlock()
	}

	err := m.pipeline.ImportFile(ctx, job.Source, job.Name, f, progress)
	m.mu.Lock()
	defer m.mu.Unlock()

	if err != nil {
		job.Status = StatusFailed
		job.Error = err.Error()
		log.Printf("import job %s failed: %v", job.ID, err)
	} else {
		job.Status = StatusComplete
		job.Progress = job.Total
		log.Printf("import job %s completed: %s", job.ID, job.Name)
	}
}

func generateID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}
