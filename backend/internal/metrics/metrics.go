// Package metrics is a tiny dependency-free Prometheus text exporter for
// the handful of counters this service needs.
package metrics

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
)

type Registry struct {
	mu       sync.Mutex
	counters map[string]*atomic.Int64 // full series name incl. labels
	gauges   map[string]*atomic.Int64
	help     map[string]string
}

func New() *Registry {
	return &Registry{counters: map[string]*atomic.Int64{}, gauges: map[string]*atomic.Int64{}, help: map[string]string{}}
}

func series(name string, labels ...string) string {
	if len(labels) == 0 {
		return name
	}
	parts := make([]string, 0, len(labels)/2)
	for i := 0; i+1 < len(labels); i += 2 {
		v := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`).Replace(labels[i+1])
		parts = append(parts, fmt.Sprintf(`%s="%s"`, labels[i], v))
	}
	return name + "{" + strings.Join(parts, ",") + "}"
}

func (r *Registry) get(m map[string]*atomic.Int64, key string) *atomic.Int64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	v, ok := m[key]
	if !ok {
		v = &atomic.Int64{}
		m[key] = v
	}
	return v
}

func (r *Registry) Help(name, text string) {
	r.mu.Lock()
	r.help[name] = text
	r.mu.Unlock()
}

// Inc adds 1 to a counter. labels are key, value pairs.
func (r *Registry) Inc(name string, labels ...string) {
	r.get(r.counters, series(name, labels...)).Add(1)
}

// Gauge adds delta to a gauge.
func (r *Registry) Gauge(name string, delta int64, labels ...string) {
	r.get(r.gauges, series(name, labels...)).Add(delta)
}

// Write renders the Prometheus text exposition format.
func (r *Registry) Write(w io.Writer) {
	r.mu.Lock()
	defer r.mu.Unlock()
	emit := func(m map[string]*atomic.Int64, typ string) {
		keys := make([]string, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		seen := map[string]bool{}
		for _, k := range keys {
			base := k
			if i := strings.IndexByte(k, '{'); i >= 0 {
				base = k[:i]
			}
			if !seen[base] {
				seen[base] = true
				if h := r.help[base]; h != "" {
					fmt.Fprintf(w, "# HELP %s %s\n", base, h)
				}
				fmt.Fprintf(w, "# TYPE %s %s\n", base, typ)
			}
			fmt.Fprintf(w, "%s %d\n", k, m[k].Load())
		}
	}
	emit(r.counters, "counter")
	emit(r.gauges, "gauge")
}
