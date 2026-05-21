package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type server struct {
	db     *sql.DB
	tenant string
}

type item struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at,omitempty"`
}

type voteRequest struct {
	VoterID string `json:"voter_id"`
	Score   int    `json:"score"`
	Comment string `json:"comment"`
}

var (
	requests = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "feedback_api_requests_total",
		Help: "Total API requests by tenant, route, method, and status.",
	}, []string{"tenant", "route", "method", "status"})

	latency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "feedback_api_request_duration_seconds",
		Help:    "Request latency by tenant, route, and method.",
		Buckets: prometheus.DefBuckets,
	}, []string{"tenant", "route", "method"})

	votes = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "feedback_votes_total",
		Help: "Votes submitted by tenant.",
	}, []string{"tenant"})

	uniqueVoters = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "feedback_unique_voters",
		Help: "Unique voters observed by tenant.",
	}, []string{"tenant"})
)

func main() {
	tenant := getenv("TENANT_NAME", "local")
	port := getenv("PORT", "8080")
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL is required")
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := migrate(ctx, db); err != nil {
		log.Fatal(err)
	}

	s := &server{db: db, tenant: tenant}
	mux := http.NewServeMux()
	mux.Handle("GET /healthz", s.instrument("healthz", http.HandlerFunc(s.healthz)))
	mux.Handle("POST /items", s.instrument("create_item", http.HandlerFunc(s.createItem)))
	mux.Handle("GET /items", s.instrument("list_items", http.HandlerFunc(s.listItems)))
	mux.Handle("POST /items/{id}/vote", s.instrument("vote", http.HandlerFunc(s.vote)))
	mux.Handle("GET /results", s.instrument("results", http.HandlerFunc(s.results)))
	mux.Handle("GET /metrics", promhttp.Handler())

	log.Printf("feedback API listening on :%s for tenant %s", port, tenant)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

func migrate(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS items (
  id BIGSERIAL PRIMARY KEY,
  name TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS votes (
  id BIGSERIAL PRIMARY KEY,
  item_id BIGINT NOT NULL REFERENCES items(id) ON DELETE CASCADE,
  voter_id TEXT NOT NULL,
  score INT NOT NULL CHECK (score >= 1 AND score <= 5),
  comment TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (item_id, voter_id)
);
`)
	return err
}

func (s *server) healthz(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := s.db.PingContext(ctx); err != nil {
		writeError(w, http.StatusServiceUnavailable, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "tenant": s.tenant})
}

func (s *server) createItem(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, errors.New("name is required"))
		return
	}

	var created item
	err := s.db.QueryRowContext(r.Context(),
		`INSERT INTO items (name) VALUES ($1) RETURNING id, name, created_at`,
		req.Name,
	).Scan(&created.ID, &created.Name, &created.CreatedAt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (s *server) listItems(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.QueryContext(r.Context(), `SELECT id, name, created_at FROM items ORDER BY id`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()

	items := []item{}
	for rows.Next() {
		var it item
		if err := rows.Scan(&it.ID, &it.Name, &it.CreatedAt); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		items = append(items, it)
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *server) vote(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req voteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if strings.TrimSpace(req.VoterID) == "" || req.Score < 1 || req.Score > 5 {
		writeError(w, http.StatusBadRequest, errors.New("voter_id and score 1-5 are required"))
		return
	}

	_, err := s.db.ExecContext(r.Context(),
		`INSERT INTO votes (item_id, voter_id, score, comment)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (item_id, voter_id)
		 DO UPDATE SET score = EXCLUDED.score, comment = EXCLUDED.comment, created_at = now()`,
		id, req.VoterID, req.Score, req.Comment,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	votes.WithLabelValues(s.tenant).Inc()
	s.refreshUniqueVoters(r.Context())
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "recorded"})
}

func (s *server) results(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.QueryContext(r.Context(), `
SELECT i.id, i.name, COUNT(v.id), COALESCE(AVG(v.score), 0)
FROM items i
LEFT JOIN votes v ON v.item_id = i.id
GROUP BY i.id, i.name
ORDER BY i.id`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()

	type result struct {
		ItemID       int64   `json:"item_id"`
		Name         string  `json:"name"`
		Votes        int64   `json:"votes"`
		AverageScore float64 `json:"average_score"`
	}
	results := []result{}
	for rows.Next() {
		var res result
		if err := rows.Scan(&res.ItemID, &res.Name, &res.Votes, &res.AverageScore); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		results = append(results, res)
	}
	s.refreshUniqueVoters(r.Context())
	writeJSON(w, http.StatusOK, results)
}

func (s *server) refreshUniqueVoters(ctx context.Context) {
	var count float64
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(DISTINCT voter_id) FROM votes`).Scan(&count); err == nil {
		uniqueVoters.WithLabelValues(s.tenant).Set(count)
	}
}

func (s *server) instrument(route string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rw, r)
		status := http.StatusText(rw.status)
		requests.WithLabelValues(s.tenant, route, r.Method, status).Inc()
		latency.WithLabelValues(s.tenant, route, r.Method).Observe(time.Since(start).Seconds())
	})
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
