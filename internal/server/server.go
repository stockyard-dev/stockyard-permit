package server

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/stockyard-dev/stockyard-permit/internal/store"
)

type Server struct {
	db     *store.DB
	mux    *http.ServeMux
	limits  Limits
	dataDir string
	pCfg    map[string]json.RawMessage
}

func New(db *store.DB, limits Limits, dataDir string) *Server {
	s := &Server{db: db, mux: http.NewServeMux(), limits: limits, dataDir: dataDir}
	s.loadPersonalConfig()
	s.mux.HandleFunc("GET /api/permits", s.listPermits)
	s.mux.HandleFunc("POST /api/permits", s.createPermits)
	s.mux.HandleFunc("GET /api/permits/export.csv", s.exportPermits)
	s.mux.HandleFunc("GET /api/permits/{id}", s.getPermits)
	s.mux.HandleFunc("PUT /api/permits/{id}", s.updatePermits)
	s.mux.HandleFunc("DELETE /api/permits/{id}", s.delPermits)
	s.mux.HandleFunc("GET /api/stats", s.stats)
	s.mux.HandleFunc("GET /api/health", s.health)
	s.mux.HandleFunc("GET /health", s.health)
	s.mux.HandleFunc("GET /ui", s.dashboard)
	s.mux.HandleFunc("GET /ui/", s.dashboard)
	s.mux.HandleFunc("GET /", s.root)
	s.mux.HandleFunc("GET /api/tier", s.tierHandler)
	s.mux.HandleFunc("GET /api/config", s.configHandler)
	s.mux.HandleFunc("GET /api/extras/{resource}", s.listExtras)
	s.mux.HandleFunc("GET /api/extras/{resource}/{id}", s.getExtras)
	s.mux.HandleFunc("PUT /api/extras/{resource}/{id}", s.putExtras)
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.mux.ServeHTTP(w, r) }
func wj(w http.ResponseWriter, c int, v any) { w.Header().Set("Content-Type", "application/json"); w.WriteHeader(c); json.NewEncoder(w).Encode(v) }
func we(w http.ResponseWriter, c int, m string) { wj(w, c, map[string]string{"error": m}) }
func (s *Server) root(w http.ResponseWriter, r *http.Request) { if r.URL.Path != "/" { http.NotFound(w, r); return }; http.Redirect(w, r, "/ui", 302) }
func oe[T any](s []T) []T { if s == nil { return []T{} }; return s }
func init() { log.SetFlags(log.LstdFlags | log.Lshortfile) }

func (s *Server) listPermits(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	filters := map[string]string{}
	if v := r.URL.Query().Get("status"); v != "" { filters["status"] = v }
	if q != "" || len(filters) > 0 { wj(w, 200, map[string]any{"permits": oe(s.db.SearchPermits(q, filters))}); return }
	wj(w, 200, map[string]any{"permits": oe(s.db.ListPermits())})
}

func (s *Server) createPermits(w http.ResponseWriter, r *http.Request) {
	if s.limits.Tier == "none" { we(w, 402, "No license key. Start a 14-day trial at https://stockyard.dev/for/"); return }
	if s.limits.TrialExpired { we(w, 402, "Trial expired. Subscribe at https://stockyard.dev/pricing/"); return }
	var e store.Permits
	json.NewDecoder(r.Body).Decode(&e)
	if e.PermitType == "" { we(w, 400, "permit_type required"); return }
	if e.HolderName == "" { we(w, 400, "holder_name required"); return }
	s.db.CreatePermits(&e)
	wj(w, 201, s.db.GetPermits(e.ID))
}

func (s *Server) getPermits(w http.ResponseWriter, r *http.Request) {
	e := s.db.GetPermits(r.PathValue("id"))
	if e == nil { we(w, 404, "not found"); return }
	wj(w, 200, e)
}

func (s *Server) updatePermits(w http.ResponseWriter, r *http.Request) {
	existing := s.db.GetPermits(r.PathValue("id"))
	if existing == nil { we(w, 404, "not found"); return }
	var patch store.Permits
	json.NewDecoder(r.Body).Decode(&patch)
	patch.ID = existing.ID; patch.CreatedAt = existing.CreatedAt
	if patch.PermitType == "" { patch.PermitType = existing.PermitType }
	if patch.HolderName == "" { patch.HolderName = existing.HolderName }
	if patch.HolderEmail == "" { patch.HolderEmail = existing.HolderEmail }
	if patch.PermitNumber == "" { patch.PermitNumber = existing.PermitNumber }
	if patch.IssuedDate == "" { patch.IssuedDate = existing.IssuedDate }
	if patch.ExpiryDate == "" { patch.ExpiryDate = existing.ExpiryDate }
	if patch.IssuingAuthority == "" { patch.IssuingAuthority = existing.IssuingAuthority }
	if patch.Status == "" { patch.Status = existing.Status }
	if patch.Notes == "" { patch.Notes = existing.Notes }
	s.db.UpdatePermits(&patch)
	wj(w, 200, s.db.GetPermits(patch.ID))
}

func (s *Server) delPermits(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id"); s.db.DeletePermits(id); s.db.DeleteExtras("permits", id)
	wj(w, 200, map[string]string{"deleted": "ok"})
}

func (s *Server) exportPermits(w http.ResponseWriter, r *http.Request) {
	items := s.db.ListPermits()
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=permits.csv")
	cw := csv.NewWriter(w)
	cw.Write([]string{"id", "permit_type", "holder_name", "holder_email", "permit_number", "issued_date", "expiry_date", "issuing_authority", "status", "cost", "notes", "created_at"})
	for _, e := range items {
		cw.Write([]string{e.ID, fmt.Sprintf("%v", e.PermitType), fmt.Sprintf("%v", e.HolderName), fmt.Sprintf("%v", e.HolderEmail), fmt.Sprintf("%v", e.PermitNumber), fmt.Sprintf("%v", e.IssuedDate), fmt.Sprintf("%v", e.ExpiryDate), fmt.Sprintf("%v", e.IssuingAuthority), fmt.Sprintf("%v", e.Status), fmt.Sprintf("%v", e.Cost), fmt.Sprintf("%v", e.Notes), e.CreatedAt})
	}
	cw.Flush()
}

func (s *Server) stats(w http.ResponseWriter, r *http.Request) {
	m := map[string]any{}
	m["permits_total"] = s.db.CountPermits()
	wj(w, 200, m)
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	m := map[string]any{"status": "ok", "service": "permit"}
	m["permits"] = s.db.CountPermits()
	wj(w, 200, m)
}

// loadPersonalConfig reads config.json from the data directory.
func (s *Server) loadPersonalConfig() {
	path := filepath.Join(s.dataDir, "config.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var cfg map[string]json.RawMessage
	if err := json.Unmarshal(data, &cfg); err != nil {
		log.Printf("warning: could not parse config.json: %v", err)
		return
	}
	s.pCfg = cfg
	log.Printf("loaded personalization from %s", path)
}

func (s *Server) configHandler(w http.ResponseWriter, r *http.Request) {
	if s.pCfg == nil {
		wj(w, 200, map[string]any{})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.pCfg)
}

// listExtras returns all extras for a resource type as {record_id: {...fields...}}
func (s *Server) listExtras(w http.ResponseWriter, r *http.Request) {
	resource := r.PathValue("resource")
	all := s.db.AllExtras(resource)
	out := make(map[string]json.RawMessage, len(all))
	for id, data := range all {
		out[id] = json.RawMessage(data)
	}
	wj(w, 200, out)
}

// getExtras returns the extras blob for a single record.
func (s *Server) getExtras(w http.ResponseWriter, r *http.Request) {
	resource := r.PathValue("resource")
	id := r.PathValue("id")
	data := s.db.GetExtras(resource, id)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(data))
}

// putExtras stores the extras blob for a single record.
func (s *Server) putExtras(w http.ResponseWriter, r *http.Request) {
	resource := r.PathValue("resource")
	id := r.PathValue("id")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		we(w, 400, "read body")
		return
	}
	var probe map[string]any
	if err := json.Unmarshal(body, &probe); err != nil {
		we(w, 400, "invalid json")
		return
	}
	if err := s.db.SetExtras(resource, id, string(body)); err != nil {
		we(w, 500, "save failed")
		return
	}
	wj(w, 200, map[string]string{"ok": "saved"})
}
