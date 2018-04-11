package memsizeui

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/fjl/memsize"
)

type Handler struct {
	memsize.RootSet

	init     sync.Once
	mux      http.ServeMux
	mu       sync.Mutex
	reports  map[int]Report
	reportID int
}

type Report struct {
	ID       int
	Date     time.Time
	Duration time.Duration
	RootName string
	Sizes    memsize.Sizes
}

type templateCtx struct {
	Roots   []string
	Reports map[int]Report
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.init.Do(func() {
		h.reports = make(map[int]Report)
		h.mux.HandleFunc("/", h.handleRoot)
		h.mux.HandleFunc("/scan", h.handleScan)
		h.mux.HandleFunc("/report/", h.handleReport)
	})
	h.mux.ServeHTTP(w, r)
}

func (h *Handler) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	serveHTML(w, rootTemplate, &templateCtx{Roots: h.Roots(), Reports: h.reports})
}

func (h *Handler) handleScan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "invalid HTTP method, want POST", http.StatusMethodNotAllowed)
		return
	}
	id := h.scan(r.URL.Query().Get("root"))
	w.Header().Add("Location", fmt.Sprintf("../report/%d", id))
	w.WriteHeader(http.StatusSeeOther)
}

func (h *Handler) handleReport(w http.ResponseWriter, r *http.Request) {
	var id int
	fmt.Sscan(strings.TrimPrefix(r.URL.Path, "/report/"), &id)

	rep, ok := h.reports[id]
	if !ok {
		http.Error(w, "report not found", http.StatusNotFound)
		return
	}
	serveHTML(w, reportTemplate, rep)
}

func (h *Handler) scan(root string) int {
	h.mu.Lock()
	defer h.mu.Unlock()

	id := h.reportID
	start := time.Now()
	sizes := h.ScanRoot(root)
	h.reports[id] = Report{
		ID:       id,
		RootName: root,
		Date:     start.Truncate(1 * time.Second),
		Duration: time.Since(start),
		Sizes:    sizes,
	}
	h.reportID++
	return id
}

func serveHTML(w http.ResponseWriter, tpl *template.Template, data interface{}) {
	w.Header().Set("content-type", "text/html")
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	buf.WriteTo(w)
}
