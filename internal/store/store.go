package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"
	_ "modernc.org/sqlite"
)

type DB struct { db *sql.DB }

type Permits struct {
	ID string `json:"id"`
	PermitType string `json:"permit_type"`
	HolderName string `json:"holder_name"`
	HolderEmail string `json:"holder_email"`
	PermitNumber string `json:"permit_number"`
	IssuedDate string `json:"issued_date"`
	ExpiryDate string `json:"expiry_date"`
	IssuingAuthority string `json:"issuing_authority"`
	Status string `json:"status"`
	Cost float64 `json:"cost"`
	Notes string `json:"notes"`
	CreatedAt string `json:"created_at"`
}

func Open(d string) (*DB, error) {
	if err := os.MkdirAll(d, 0755); err != nil { return nil, err }
	db, err := sql.Open("sqlite", filepath.Join(d, "permit.db")+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil { return nil, err }
	db.SetMaxOpenConns(1)
	db.Exec(`CREATE TABLE IF NOT EXISTS permits(id TEXT PRIMARY KEY, permit_type TEXT NOT NULL, holder_name TEXT NOT NULL, holder_email TEXT DEFAULT '', permit_number TEXT DEFAULT '', issued_date TEXT DEFAULT '', expiry_date TEXT DEFAULT '', issuing_authority TEXT DEFAULT '', status TEXT DEFAULT '', cost REAL DEFAULT 0, notes TEXT DEFAULT '', created_at TEXT DEFAULT(datetime('now')))`)
	return &DB{db: db}, nil
}

func (d *DB) Close() error { return d.db.Close() }
func genID() string { return fmt.Sprintf("%d", time.Now().UnixNano()) }
func now() string { return time.Now().UTC().Format(time.RFC3339) }

func (d *DB) CreatePermits(e *Permits) error {
	e.ID = genID(); e.CreatedAt = now()
	_, err := d.db.Exec(`INSERT INTO permits(id, permit_type, holder_name, holder_email, permit_number, issued_date, expiry_date, issuing_authority, status, cost, notes, created_at) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, e.ID, e.PermitType, e.HolderName, e.HolderEmail, e.PermitNumber, e.IssuedDate, e.ExpiryDate, e.IssuingAuthority, e.Status, e.Cost, e.Notes, e.CreatedAt)
	return err
}

func (d *DB) GetPermits(id string) *Permits {
	var e Permits
	if d.db.QueryRow(`SELECT id, permit_type, holder_name, holder_email, permit_number, issued_date, expiry_date, issuing_authority, status, cost, notes, created_at FROM permits WHERE id=?`, id).Scan(&e.ID, &e.PermitType, &e.HolderName, &e.HolderEmail, &e.PermitNumber, &e.IssuedDate, &e.ExpiryDate, &e.IssuingAuthority, &e.Status, &e.Cost, &e.Notes, &e.CreatedAt) != nil { return nil }
	return &e
}

func (d *DB) ListPermits() []Permits {
	rows, _ := d.db.Query(`SELECT id, permit_type, holder_name, holder_email, permit_number, issued_date, expiry_date, issuing_authority, status, cost, notes, created_at FROM permits ORDER BY created_at DESC`)
	if rows == nil { return nil }; defer rows.Close()
	var o []Permits
	for rows.Next() { var e Permits; rows.Scan(&e.ID, &e.PermitType, &e.HolderName, &e.HolderEmail, &e.PermitNumber, &e.IssuedDate, &e.ExpiryDate, &e.IssuingAuthority, &e.Status, &e.Cost, &e.Notes, &e.CreatedAt); o = append(o, e) }
	return o
}

func (d *DB) UpdatePermits(e *Permits) error {
	_, err := d.db.Exec(`UPDATE permits SET permit_type=?, holder_name=?, holder_email=?, permit_number=?, issued_date=?, expiry_date=?, issuing_authority=?, status=?, cost=?, notes=? WHERE id=?`, e.PermitType, e.HolderName, e.HolderEmail, e.PermitNumber, e.IssuedDate, e.ExpiryDate, e.IssuingAuthority, e.Status, e.Cost, e.Notes, e.ID)
	return err
}

func (d *DB) DeletePermits(id string) error {
	_, err := d.db.Exec(`DELETE FROM permits WHERE id=?`, id)
	return err
}

func (d *DB) CountPermits() int {
	var n int; d.db.QueryRow(`SELECT COUNT(*) FROM permits`).Scan(&n); return n
}

func (d *DB) SearchPermits(q string, filters map[string]string) []Permits {
	where := "1=1"
	args := []any{}
	if q != "" {
		where += " AND (permit_type LIKE ? OR holder_name LIKE ? OR holder_email LIKE ? OR permit_number LIKE ? OR issuing_authority LIKE ? OR notes LIKE ?)"
		args = append(args, "%"+q+"%")
		args = append(args, "%"+q+"%")
		args = append(args, "%"+q+"%")
		args = append(args, "%"+q+"%")
		args = append(args, "%"+q+"%")
		args = append(args, "%"+q+"%")
	}
	if v, ok := filters["status"]; ok && v != "" { where += " AND status=?"; args = append(args, v) }
	rows, _ := d.db.Query(`SELECT id, permit_type, holder_name, holder_email, permit_number, issued_date, expiry_date, issuing_authority, status, cost, notes, created_at FROM permits WHERE `+where+` ORDER BY created_at DESC`, args...)
	if rows == nil { return nil }; defer rows.Close()
	var o []Permits
	for rows.Next() { var e Permits; rows.Scan(&e.ID, &e.PermitType, &e.HolderName, &e.HolderEmail, &e.PermitNumber, &e.IssuedDate, &e.ExpiryDate, &e.IssuingAuthority, &e.Status, &e.Cost, &e.Notes, &e.CreatedAt); o = append(o, e) }
	return o
}
