// Package schema owns the golang-migrate v4.20.1 runner for the immutable
// numbered migration pairs in server/migrations (migrations.md, ADR-0065)
// and the hash manifest that makes committed migration files immutable
// (physical_schema_contract.md §6).
package schema

import (
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	pgxmigrate "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// ManifestName is the committed hash manifest (under
// server/internal/durable/schema/, this package's directory) pinning every
// migration file's SHA-256 — committed migration files are immutable
// (migrations.md, physical_schema_contract.md §6).
const ManifestName = "migrations.sha256"

// Migrator is the golang-migrate 4.20.1 entry point over the pinned pgx/v5
// driver. dir is a filesystem path holding NNNNNN_name.up/down.sql pairs.
type Migrator struct{ m *migrate.Migrate }

// NewMigrator wires the iofs source (dir) and the pgx/v5 database driver
// (dsn). golang-migrate's schema_migrations table tracks applied versions.
func NewMigrator(dsn, dir string) (*Migrator, error) {
	if err := CheckManifest(dir); err != nil {
		return nil, err
	}
	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("schema: migrate open: %w", err)
	}
	src, err := iofs.New(os.DirFS(dir), ".")
	if err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("schema: migrate source %s: %w", dir, err)
	}
	drv, err := pgxmigrate.WithInstance(sqlDB, &pgxmigrate.Config{})
	if err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("schema: migrate driver: %w", err)
	}
	m, err := migrate.NewWithInstance("iofs", src, "postgres", drv)
	if err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("schema: migrate init: %w", err)
	}
	return &Migrator{m: m}, nil
}

// Up applies every pending migration.
func (x *Migrator) Up() error {
	if err := x.m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("schema: migrate up: %w", err)
	}
	return nil
}

// Steps moves the schema by n signed versions (negative = down).
func (x *Migrator) Steps(n int) error {
	if err := x.m.Steps(n); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("schema: migrate steps %d: %w", n, err)
	}
	return nil
}

// Down rolls back every applied migration. Local/staging rehearsal only —
// production rollback is forward-fix or PITR (migrations.md § Down/Rollback).
func (x *Migrator) Down() error {
	if err := x.m.Down(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("schema: migrate down: %w", err)
	}
	return nil
}

// Version reports the applied migration version and dirty flag; version 0
// means no migration applied yet.
func (x *Migrator) Version() (version uint, dirty bool, err error) {
	v, d, err := x.m.Version()
	if errors.Is(err, migrate.ErrNilVersion) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("schema: migrate version: %w", err)
	}
	return v, d, nil
}

// Close releases source and database handles.
func (x *Migrator) Close() error {
	srcErr, dbErr := x.m.Close()
	if srcErr != nil {
		return fmt.Errorf("schema: close source: %w", srcErr)
	}
	if dbErr != nil {
		return fmt.Errorf("schema: close db: %w", dbErr)
	}
	return nil
}

// MigrationsDir resolves <repo>/server/migrations relative to this source
// file — valid from any package in this module.
func MigrationsDir() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "migrations"
	}
	// server/internal/durable/schema/schema.go -> server/migrations
	return filepath.Join(filepath.Dir(file), "..", "..", "..", "migrations")
}

// ManifestPath resolves the committed manifest path inside this package's
// directory via the source file location.
func ManifestPath() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return ManifestName
	}
	return filepath.Join(filepath.Dir(file), ManifestName)
}

// WriteManifest writes the manifest for dir: one `<sha256>  <name>` line per
// NNNNNN_*.sql file, sorted by name. Callers run it when authoring a new
// migration pair; CheckManifest enforces it.
func WriteManifest(dir string) error {
	names, err := migrationFiles(dir)
	if err != nil {
		return err
	}
	var b strings.Builder
	for _, name := range names {
		sum, err := hashFile(filepath.Join(dir, name))
		if err != nil {
			return err
		}
		fmt.Fprintf(&b, "%x  %s\n", sum, name)
	}
	return os.WriteFile(ManifestPath(), []byte(b.String()), 0o644)
}

// CheckManifest verifies every committed migration file against
// migrations.sha256: each file's hash must match its recorded entry, every
// migration file on disk must be listed, and the manifest must not reference
// missing files.
func CheckManifest(dir string) error {
	manifest := ManifestPath()
	data, err := os.ReadFile(manifest)
	if err != nil {
		return fmt.Errorf("schema: read %s: %w", manifest, err)
	}
	recorded := map[string]string{}
	for i, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 2 {
			return fmt.Errorf("schema: %s line %d malformed: %q", ManifestName, i+1, line)
		}
		sum, name := fields[0], fields[1]
		if !strings.HasSuffix(name, ".up.sql") && !strings.HasSuffix(name, ".down.sql") {
			return fmt.Errorf("schema: %s line %d: %q is not a migration file", ManifestName, i+1, name)
		}
		if _, dup := recorded[name]; dup {
			return fmt.Errorf("schema: %s lists %s twice", ManifestName, name)
		}
		recorded[name] = sum
	}
	files, err := migrationFiles(dir)
	if err != nil {
		return err
	}
	for _, name := range files {
		want, ok := recorded[name]
		if !ok {
			return fmt.Errorf("schema: %s missing from %s", name, ManifestName)
		}
		got, err := hashFile(filepath.Join(dir, name))
		if err != nil {
			return err
		}
		if fmt.Sprintf("%x", got) != want {
			return fmt.Errorf("schema: %s modified after commit — migrations are immutable (migrations.md)", name)
		}
		delete(recorded, name)
	}
	for name := range recorded {
		return fmt.Errorf("schema: %s lists %s but the file is absent", ManifestName, name)
	}
	return nil
}

// migrationFiles lists the NNNNNN_*.up/down.sql files of dir, sorted.
func migrationFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("schema: read %s: %w", dir, err)
	}
	var names []string
	for _, e := range entries {
		n := e.Name()
		if e.IsDir() || (!strings.HasSuffix(n, ".up.sql") && !strings.HasSuffix(n, ".down.sql")) {
			continue
		}
		names = append(names, n)
	}
	sort.Strings(names)
	return names, nil
}

func hashFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("schema: hash %s: %w", path, err)
	}
	sum := sha256.Sum256(data)
	return sum[:], nil
}
