package webui

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/robertpjenkins-bot/oliver/internal/device"
	"github.com/robertpjenkins-bot/oliver/internal/imaging"
)

// Job is one drive operation in progress. The page follows it rather than
// holding the connection open for the length of a write: a five gigabyte image
// takes minutes, and a reloaded page must be able to pick the job back up.
type Job struct {
	mu     sync.Mutex
	state  State
	cancel context.CancelFunc
}

// State is a job as the page sees it.
type State struct {
	ID        string  `json:"id"`
	Kind      string  `json:"kind"`
	Device    string  `json:"device"`
	Image     string  `json:"image,omitempty"`
	Phase     string  `json:"phase"`
	Done      int64   `json:"done"`
	Total     int64   `json:"total"`
	Percent   float64 `json:"percent"`
	RateHuman string  `json:"rate_human,omitempty"`
	Remaining string  `json:"remaining,omitempty"`
	Finished  bool    `json:"finished"`
	Error     string  `json:"error,omitempty"`
	Summary   string  `json:"summary,omitempty"`
	Digest    string  `json:"digest,omitempty"`
	Verified  bool    `json:"verified"`
}

// Snapshot returns the job's state, safe to encode while it is still running.
func (j *Job) Snapshot() State {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.state
}

// progress receives the imaging package's reports.
func (j *Job) progress(p imaging.Progress) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.state.Phase = string(p.Phase)
	j.state.Done = p.Done
	j.state.Total = p.Total
	if p.Total > 0 {
		j.state.Percent = float64(p.Done) / float64(p.Total) * 100
	}
	j.state.RateHuman = imaging.HumanBytes(int64(p.Rate())) + "/s"
	if rate := p.Rate(); rate > 0 && p.Done < p.Total {
		j.state.Remaining = shortDuration(time.Duration(float64(p.Total-p.Done)/rate) * time.Second)
	} else {
		j.state.Remaining = ""
	}
}

func (j *Job) setPhase(phase string) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.state.Phase = phase
	j.state.Remaining = ""
}

// finishWith records what an operation achieved. The job is not marked
// finished here: that happens once the work function has returned, so a
// failure after a partial result is still reported as a failure.
func (j *Job) finishWith(res *imaging.Result, summary string) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.state.Summary = summary
	if res != nil {
		j.state.Digest = res.Source
		j.state.Verified = res.Verified
	}
}

func (j *Job) end(err error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.state.Finished = true
	j.state.Phase = "done"
	j.state.Remaining = ""
	if err != nil {
		j.state.Error = err.Error()
		j.state.Summary = ""
		return
	}
	j.state.Percent = 100
}

// start runs an operation in the background and returns the job following it.
//
// One job per drive at a time: two writes to the same stick would interleave
// and produce something that is neither image.
func (s *Server) start(kind string, dev *device.Device, image string, work func(context.Context, *Job) error) (*Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, existing := range s.jobs {
		st := existing.Snapshot()
		if st.Device == dev.Path && !st.Finished {
			return nil, fmt.Errorf("%s is already being %sd", dev.Path, st.Kind)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	job := &Job{
		state: State{
			ID:     newToken()[:16],
			Kind:   kind,
			Device: dev.Path,
			Image:  image,
			Phase:  "starting",
		},
		cancel: cancel,
	}
	if s.jobs == nil {
		s.jobs = map[string]*Job{}
	}
	s.jobs[job.state.ID] = job

	go func() {
		defer cancel()
		job.end(work(ctx, job))
	}()
	return job, nil
}

func (s *Server) job(id string) *Job {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.jobs[id]
}

func (s *Server) listJobs(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	out := make([]State, 0, len(s.jobs))
	for _, j := range s.jobs {
		out = append(out, j.Snapshot())
	}
	s.mu.Unlock()
	send(w, out)
}

func (s *Server) cancelJob(w http.ResponseWriter, r *http.Request) {
	job := s.job(r.PathValue("id"))
	if job == nil {
		fail(w, http.StatusNotFound, fmt.Errorf("no such job"))
		return
	}
	job.cancel()
	send(w, job.Snapshot())
}

// jobEvents streams a job's state until it finishes.
//
// Server-sent events rather than a socket: the traffic goes one way, and a
// page that is closed mid-write leaves nothing behind but a cancelled request.
func (s *Server) jobEvents(w http.ResponseWriter, r *http.Request) {
	job := s.job(r.PathValue("id"))
	if job == nil {
		fail(w, http.StatusNotFound, fmt.Errorf("no such job"))
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		fail(w, http.StatusInternalServerError, fmt.Errorf("this connection cannot stream"))
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	var last State
	for {
		state := job.Snapshot()
		if state != last {
			last = state
			payload, err := json.Marshal(state)
			if err != nil {
				return
			}
			if _, err := fmt.Fprintf(w, "data: %s\n\n", payload); err != nil {
				return
			}
			flusher.Flush()
		}
		if state.Finished {
			return
		}
		select {
		case <-r.Context().Done():
			// The page went away. The write carries on: a drive left half
			// written because a tab was closed is worse than one finished.
			return
		case <-ticker.C:
		}
	}
}
