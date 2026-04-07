package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Identity  string     `yaml:"identity"`
	Cell      string     `yaml:"cell"`
	Port      int        `yaml:"port"`
	ServeUI   bool       `yaml:"serveUI"`
	Endpoints []Endpoint `yaml:"endpoints"`
}

type Endpoint struct {
	Method   string `yaml:"method"`
	Path     string `yaml:"path"`
	Response any    `yaml:"response"`
	// Upstream call to make when this endpoint is hit
	Upstream *UpstreamCall `yaml:"upstream,omitempty"`
}

type UpstreamCall struct {
	URL    string `yaml:"url"`
	Method string `yaml:"method"`
	Label  string `yaml:"label"`
}

// SSE event stream for the mission control dashboard
type TraceEvent struct {
	Timestamp string `json:"timestamp"`
	Source     string `json:"source"`
	Target     string `json:"target"`
	Method    string `json:"method"`
	Path      string `json:"path"`
	Status    int    `json:"status"`
	Latency   string `json:"latency"`
	MTLS      bool   `json:"mtls"`
	Error     string `json:"error,omitempty"`
}

var (
	sseClients   = make(map[chan TraceEvent]bool)
	sseClientsMu sync.Mutex
	config       Config
)

func broadcastTrace(evt TraceEvent) {
	sseClientsMu.Lock()
	defer sseClientsMu.Unlock()
	for ch := range sseClients {
		select {
		case ch <- evt:
		default:
		}
	}
}

func sseHandler(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	ch := make(chan TraceEvent, 64)
	sseClientsMu.Lock()
	sseClients[ch] = true
	sseClientsMu.Unlock()

	defer func() {
		sseClientsMu.Lock()
		delete(sseClients, ch)
		sseClientsMu.Unlock()
	}()

	for {
		select {
		case evt := <-ch:
			data, _ := json.Marshal(evt)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

func callUpstream(upstream *UpstreamCall, sourceIdentity string) (int, string, time.Duration) {
	start := time.Now()
	req, err := http.NewRequest(upstream.Method, upstream.URL, nil)
	if err != nil {
		return 0, err.Error(), time.Since(start)
	}
	req.Header.Set("X-Source-Identity", sourceIdentity)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err.Error(), time.Since(start)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(body), time.Since(start)
}

func makeHandler(ep Endpoint) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != ep.Method && ep.Method != "*" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "*")

		result := map[string]any{
			"source":    config.Identity,
			"cell":      config.Cell,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}

		// If there's an upstream call, make it and trace it
		if ep.Upstream != nil {
			status, body, latency := callUpstream(ep.Upstream, config.Identity)

			// Check for mTLS indicators
			mtls := r.TLS != nil || r.Header.Get("X-Forwarded-Client-Cert") != ""

			evt := TraceEvent{
				Timestamp: time.Now().UTC().Format("15:04:05"),
				Source:    config.Cell + "/" + config.Identity,
				Target:    ep.Upstream.Label,
				Method:    ep.Upstream.Method,
				Path:      ep.Upstream.URL,
				Status:    status,
				Latency:   latency.Round(time.Millisecond).String(),
				MTLS:      mtls,
			}
			if status == 0 || status >= 400 {
				evt.Error = truncate(body, 200)
			}
			broadcastTrace(evt)

			result["upstream"] = map[string]any{
				"target":  ep.Upstream.Label,
				"status":  status,
				"latency": latency.Round(time.Millisecond).String(),
				"body":    truncateJSON(body),
			}
		}

		// Merge static response data
		if ep.Response != nil {
			if m, ok := ep.Response.(map[string]any); ok {
				for k, v := range m {
					result[k] = v
				}
			} else {
				result["data"] = ep.Response
			}
		}

		// Trace the inbound request too
		mtls := r.TLS != nil || r.Header.Get("X-Forwarded-Client-Cert") != ""
		sourceHeader := r.Header.Get("X-Source-Identity")
		if sourceHeader == "" {
			sourceHeader = "external"
		}
		broadcastTrace(TraceEvent{
			Timestamp: time.Now().UTC().Format("15:04:05"),
			Source:    sourceHeader,
			Target:    config.Cell + "/" + config.Identity,
			Method:    r.Method,
			Path:      r.URL.Path,
			Status:    http.StatusOK,
			Latency:   "0ms",
			MTLS:      mtls,
		})

		json.NewEncoder(w).Encode(result)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status":   "online",
		"identity": config.Identity,
		"cell":     config.Cell,
	})
}

// CORS preflight handler
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func truncateJSON(s string) any {
	var obj any
	if err := json.Unmarshal([]byte(s), &obj); err == nil {
		return obj
	}
	return truncate(s, 500)
}

func main() {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config.yaml"
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		log.Fatalf("Failed to read config %s: %v", configPath, err)
	}

	if err := yaml.Unmarshal(data, &config); err != nil {
		log.Fatalf("Failed to parse config: %v", err)
	}

	mux := http.NewServeMux()

	// Health endpoint
	mux.HandleFunc("/health", healthHandler)

	// SSE endpoint for real-time trace
	mux.HandleFunc("/events", sseHandler)

	// Group endpoints by path to avoid duplicate registration
	pathHandlers := make(map[string]map[string]Endpoint)
	for _, ep := range config.Endpoints {
		if _, ok := pathHandlers[ep.Path]; !ok {
			pathHandlers[ep.Path] = make(map[string]Endpoint)
		}
		pathHandlers[ep.Path][ep.Method] = ep
		log.Printf("  %s %s", ep.Method, ep.Path)
	}

	for path, methods := range pathHandlers {
		methods := methods // capture
		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			// Try exact method match first, then wildcard
			if ep, ok := methods[r.Method]; ok {
				makeHandler(ep)(w, r)
				return
			}
			if ep, ok := methods["*"]; ok {
				makeHandler(ep)(w, r)
				return
			}
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		})
	}

	// Serve dashboard UI if configured
	if config.ServeUI {
		uiDir := os.Getenv("UI_DIR")
		if uiDir == "" {
			uiDir = "ui"
		}
		mux.Handle("/ui/", http.StripPrefix("/ui/", http.FileServer(http.Dir(uiDir))))
		// Serve dashboard at root
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/" {
				http.ServeFile(w, r, uiDir+"/dashboard.html")
				return
			}
			// Check if it's a registered API path
			for _, ep := range config.Endpoints {
				if strings.HasPrefix(r.URL.Path, ep.Path) {
					makeHandler(ep)(w, r)
					return
				}
			}
			http.NotFound(w, r)
		})
		log.Printf("  Dashboard UI enabled at /")
	}

	addr := fmt.Sprintf(":%d", config.Port)
	log.Printf("[%s/%s] Starting on %s", config.Cell, config.Identity, addr)

	server := &http.Server{
		Addr:    addr,
		Handler: corsMiddleware(mux),
	}
	log.Fatal(server.ListenAndServe())
}
