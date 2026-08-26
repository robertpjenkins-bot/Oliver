package vbox

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Record is what Oliver remembers about a machine it made.
//
// Its purpose is the first-run question: a machine with a record has already
// been set up, so starting it must not ask about memory and disks again.
type Record struct {
	Spec        Spec      `json:"spec"`
	Created     time.Time `json:"created"`
	LastStarted time.Time `json:"last_started"`
	Starts      int       `json:"starts"`
}

var unsafeName = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

// recordPath is the file holding one machine's record.
func (m *Manager) recordPath(name string) string {
	return filepath.Join(m.Root, unsafeName.ReplaceAllString(name, "-")+".json")
}

// Save writes a machine's record.
func (m *Manager) Save(r Record) error {
	if err := os.MkdirAll(m.Root, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.recordPath(r.Spec.Name), append(data, '\n'), 0o644)
}

// Load returns a machine's record, and whether there was one.
func (m *Manager) Load(name string) (Record, bool, error) {
	data, err := os.ReadFile(m.recordPath(name))
	if err != nil {
		if os.IsNotExist(err) {
			return Record{}, false, nil
		}
		return Record{}, false, err
	}
	var r Record
	if err := json.Unmarshal(data, &r); err != nil {
		return Record{}, false, err
	}
	return r, true, nil
}

// Forget removes a machine's record without touching the machine itself.
func (m *Manager) Forget(name string) error {
	err := os.Remove(m.recordPath(name))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// MarkStarted records a boot, which is how "has this ever run?" is answered.
func (m *Manager) MarkStarted(name string) error {
	r, ok, err := m.Load(name)
	if err != nil || !ok {
		return err
	}
	r.LastStarted = time.Now()
	r.Starts++
	return m.Save(r)
}

// List returns every machine Oliver has made, oldest first.
func (m *Manager) List() ([]Record, error) {
	entries, err := os.ReadDir(m.Root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []Record
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(m.Root, e.Name()))
		if err != nil {
			continue
		}
		var r Record
		if err := json.Unmarshal(data, &r); err != nil {
			continue
		}
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Created.Before(out[j].Created) })
	return out, nil
}
