package db

import (
	"database/sql"

	"github.com/HaythmKenway/autoscout/pkg/localUtils"
)

func InitDB(db *sql.DB) error {

	// IMPORTANT: Enable foreign key cascade rules in SQLite
	_, err := db.Exec(`PRAGMA foreign_keys = ON;`)
	if err != nil {
		return err
	}

	// Create tables in correct dependency order
	steps := []func(*sql.DB) error{
		createTargetTableIfNotExists,
		createTargetPatternsIfNotExists,
		createSubsTableIfNotExists,
		createUrlsTableIfNotExist,
		createSpiderTableIfNotExist,
		createProcFuncsTable,
		createProcPathsTable,
		createBranchingRulesTable,
		createProcPathItemsTable,
	}

	for _, step := range steps {
		if err := step(db); err != nil {
			return err
		}
	}

	// Seed workflow only once
	return SeedDefaultWorkflow(db)
}

func createTargetTableIfNotExists(db *sql.DB) error {
	_, err := db.Exec(`
        CREATE TABLE IF NOT EXISTS targets (
            subdomain TEXT PRIMARY KEY,
            lastModified DATE DEFAULT CURRENT_TIMESTAMP,
            lastScanned DATE
		)
    `)
	return err
}

func createTargetPatternsIfNotExists(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS target_patterns (
			pattern_id INTEGER PRIMARY KEY,
			pattern TEXT NOT NULL,
			target_type TEXT CHECK(target_type IN ('INCLUDE', 'EXCLUDE')) NOT NULL,
			subdomain TEXT NOT NULL,
			FOREIGN KEY(subdomain) REFERENCES targets(subdomain) ON DELETE CASCADE
		)
	`)
	return err
}

func createSubsTableIfNotExists(db *sql.DB) error {
	_, err := db.Exec(`
        CREATE TABLE IF NOT EXISTS subdomain (
            subdomain TEXT PRIMARY KEY,
            parent_target TEXT NOT NULL,
            lastModified DATE DEFAULT CURRENT_TIMESTAMP,
            FOREIGN KEY(parent_target) REFERENCES targets(subdomain) ON DELETE CASCADE
        )
    `)
	return err
}

func createUrlsTableIfNotExist(db *sql.DB) error {
	_, err := db.Exec(`
        CREATE TABLE IF NOT EXISTS urls (
			subdomain TEXT NOT NULL,
            url TEXT PRIMARY KEY,
            host TEXT NOT NULL,
            title TEXT,
            scheme TEXT,
            a TEXT,
            cname TEXT,
            tech TEXT,
            ip TEXT,
            port TEXT,
            status_code TEXT,
            lastModified DATE DEFAULT CURRENT_TIMESTAMP,
            FOREIGN KEY(host) REFERENCES targets(subdomain) ON DELETE CASCADE
			FOREIGN KEY(subdomain) REFERENCES subdomain(subdomain) ON DELETE CASCADE
        )
    `)
	return err
}

func createSpiderTableIfNotExist(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS spider (
			url TEXT PRIMARY KEY,
			target TEXT NOT NULL,
			lastModified DATE DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(target) REFERENCES targets(subdomain) ON DELETE CASCADE
		)
	`)
	return err
}

// ------------------- Workflow tables (unchanged logic) -------------------

func createProcFuncsTable(db *sql.DB) error {
	_, err := db.Exec(`
        CREATE TABLE IF NOT EXISTS proc_funcs (
            proc_func_id INTEGER PRIMARY KEY,
            func_name TEXT NOT NULL,
            binary_path TEXT
        )
    `)
	return err
}

func createProcPathsTable(db *sql.DB) error {
	_, err := db.Exec(`
        CREATE TABLE IF NOT EXISTS proc_paths (
            proc_path_id INTEGER PRIMARY KEY,
            path_name TEXT NOT NULL,
            description TEXT
        )
    `)
	return err
}

func createBranchingRulesTable(db *sql.DB) error {
	_, err := db.Exec(`
        CREATE TABLE IF NOT EXISTS branching_rules (
            rule_id INTEGER PRIMARY KEY,
            rule_name TEXT,
            match_type TEXT CHECK(match_type IN ('REGEX', 'EXACT', 'TYPE')),
            match_criteria TEXT NOT NULL,
            priority INTEGER DEFAULT 0,
            target_path_id INTEGER,
            FOREIGN KEY(target_path_id) REFERENCES proc_paths(proc_path_id)
        )
    `)
	return err
}

func createProcPathItemsTable(db *sql.DB) error {
	_, err := db.Exec(`
        CREATE TABLE IF NOT EXISTS proc_path_items (
            item_id INTEGER PRIMARY KEY,
            proc_path_id INTEGER NOT NULL,
            proc_func_id INTEGER NOT NULL,
            exec_order INTEGER NOT NULL,
            input_source TEXT DEFAULT 'PREV_STEP_OUTPUT',
            args TEXT,
            UNIQUE(proc_path_id, exec_order),
            FOREIGN KEY(proc_path_id) REFERENCES proc_paths(proc_path_id) ON DELETE CASCADE,
            FOREIGN KEY(proc_func_id) REFERENCES proc_funcs(proc_func_id)
        )
    `)
	return err
}

// ------------------- Seeder -------------------

func SeedDefaultWorkflow(db *sql.DB) error {
	var count int
	row := db.QueryRow("SELECT COUNT(*) FROM proc_funcs")
	if err := row.Scan(&count); err != nil || count > 0 {
		return nil
	}

	localUtils.Logger("Seeding default workflow data...", 1)

	tools := []string{
		`INSERT INTO proc_funcs (proc_func_id, func_name, binary_path) VALUES (1, 'Subfinder', 'subfinder')`,
		`INSERT INTO proc_funcs (proc_func_id, func_name, binary_path) VALUES (2, 'HTTPX', 'httpx')`,
		`INSERT INTO proc_funcs (proc_func_id, func_name, binary_path) VALUES (3, 'GoSpider', 'gospider')`,
		`INSERT INTO proc_funcs (proc_func_id, func_name, binary_path) VALUES (4, 'DalFox', 'dalfox')`,
	}

	for _, q := range tools {
		if _, err := db.Exec(q); err != nil {
			return err
		}
	}

	_, err := db.Exec(`INSERT INTO proc_paths (proc_path_id, path_name, description) VALUES (10, 'Standard Domain Recon', 'Subfinder -> HTTPX -> GoSpider -> DalFox')`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`INSERT INTO branching_rules (rule_id, rule_name, match_type, match_criteria, target_path_id) VALUES (100, 'Domain Match', 'REGEX', '^[a-zA-Z0-9-]+\.[a-zA-Z]+$', 10)`)
	if err != nil {
		return err
	}

	steps := []string{
		`INSERT INTO proc_path_items (proc_path_id, proc_func_id, exec_order, input_source, args) VALUES (10, 1, 1, 'USER_INPUT', '-silent -d')`,
		`INSERT INTO proc_path_items (proc_path_id, proc_func_id, exec_order, input_source, args) VALUES (10, 2, 2, 'PREV_STEP_OUTPUT', '-silent -status-code -tech-detect')`,
		`INSERT INTO proc_path_items (proc_path_id, proc_func_id, exec_order, input_source, args) VALUES (10, 3, 3, 'PREV_STEP_OUTPUT', '-q -s')`,
		`INSERT INTO proc_path_items (proc_path_id, proc_func_id, exec_order, input_source, args) VALUES (10, 4, 4, 'PREV_STEP_OUTPUT', 'pipe --silence --skip-mining-all')`,
	}

	for _, q := range steps {
		if _, err := db.Exec(q); err != nil {
			return err
		}
	}

	return nil
}
