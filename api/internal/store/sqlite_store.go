package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/EpicMandM/esxi-lab-provider/api/internal/models"
	_ "modernc.org/sqlite"
)

type SQLiteStore struct {
	db *sql.DB
}

func NewSQLiteStore(path string) (*SQLiteStore, error) {
	dbPath, err := resolveDBPath(path)
	if err != nil {
		return nil, err
	}

	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)", dbPath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}

	if err := initSchema(db); err != nil {
		if cerr := db.Close(); cerr != nil {
			return nil, errors.Join(err, cerr)
		}
		return nil, err
	}

	return &SQLiteStore{db: db}, nil
}

func resolveDBPath(path string) (string, error) {
	abs := filepath.Clean(path)
	if strings.HasSuffix(abs, ".db") {
		if err := os.MkdirAll(filepath.Dir(abs), 0o750); err != nil {
			return "", err
		}
		return abs, nil
	}
	if err := os.MkdirAll(abs, 0o750); err != nil {
		return "", err
	}
	return filepath.Join(abs, "store.db"), nil
}

func initSchema(db *sql.DB) error {
	stmts := []string{
		"PRAGMA foreign_keys = ON;",
		"CREATE TABLE IF NOT EXISTS vms (name TEXT PRIMARY KEY, data BLOB NOT NULL);",
		"CREATE TABLE IF NOT EXISTS bookings (id TEXT PRIMARY KEY, vm_name TEXT NOT NULL, start_time TEXT NOT NULL, end_time TEXT NOT NULL, data BLOB NOT NULL);",
		"CREATE INDEX IF NOT EXISTS idx_bookings_vm_time ON bookings(vm_name, start_time, end_time);",
	}
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

func (s *SQLiteStore) GetVMByName(name string) (*models.VM, error) {
	var raw []byte
	err := s.db.QueryRow(`SELECT data FROM vms WHERE name = ?`, name).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	var vm models.VM
	if err := json.Unmarshal(raw, &vm); err != nil {
		return nil, err
	}
	return &vm, nil
}

func (s *SQLiteStore) SaveVM(vm *models.VM) error {
	data, err := json.Marshal(vm)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`INSERT INTO vms (name, data) VALUES (?, ?) ON CONFLICT(name) DO UPDATE SET data = excluded.data`, vm.Name, data)
	return err
}

func (s *SQLiteStore) ListVMs() ([]*models.VM, error) {
	rows, err := s.db.Query(`SELECT data FROM vms ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	var vms []*models.VM
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		var vm models.VM
		if err := json.Unmarshal(raw, &vm); err != nil {
			return nil, err
		}
		vms = append(vms, &vm)
	}
	return vms, rows.Err()
}

func (s *SQLiteStore) CreateBooking(booking *models.Booking) error {
	data, err := json.Marshal(booking)
	if err != nil {
		return err
	}
	start := booking.StartTime.UTC().Format(time.RFC3339Nano)
	end := booking.EndTime.UTC().Format(time.RFC3339Nano)
	_, err = s.db.Exec(`INSERT INTO bookings (id, vm_name, start_time, end_time, data)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET vm_name=excluded.vm_name, start_time=excluded.start_time, end_time=excluded.end_time, data=excluded.data`,
		booking.ID, booking.VMName, start, end, data)
	return err
}

func (s *SQLiteStore) GetActiveBookingForVM(vmName string, at time.Time) (*models.Booking, error) {
	instant := at.UTC().Format(time.RFC3339Nano)
	var raw []byte
	err := s.db.QueryRow(`SELECT data FROM bookings WHERE vm_name = ? AND start_time <= ? AND end_time >= ? LIMIT 1`, vmName, instant, instant).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var booking models.Booking
	if err := json.Unmarshal(raw, &booking); err != nil {
		return nil, err
	}
	return &booking, nil
}

func (s *SQLiteStore) DeleteBooking(id string) error {
	_, err := s.db.Exec(`DELETE FROM bookings WHERE id = ?`, id)
	return err
}
