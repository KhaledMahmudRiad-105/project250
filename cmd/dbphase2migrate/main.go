
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	_ "modernc.org/sqlite"
)

func main() {
	ctx := context.Background()

	database, err := sql.Open("sqlite", "./school.db")
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()

	database.SetMaxOpenConns(1)

	conn, err := database.Conn(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	var before int
	err = conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM students").Scan(&before)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Students before migration: %d\n", before)

	_, err = conn.ExecContext(ctx, "PRAGMA foreign_keys = ON")
	if err != nil {
		log.Fatal(err)
	}

	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}

	fail := func(err error) {
		_ = tx.Rollback()
		log.Fatal(err)
	}

	schema := `
	CREATE TABLE IF NOT EXISTS subjects (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE
	);

	CREATE TABLE IF NOT EXISTS school_roles (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		role TEXT NOT NULL UNIQUE
			CHECK(role IN ('principal', 'vice_principal')),
		teacher_id INTEGER NOT NULL UNIQUE,
		FOREIGN KEY (teacher_id) REFERENCES teachers(id)
	);

	CREATE TABLE IF NOT EXISTS teacher_subjects (
		teacher_id INTEGER PRIMARY KEY,
		subject_id INTEGER NOT NULL,
		UNIQUE(teacher_id, subject_id),
		FOREIGN KEY (teacher_id) REFERENCES teachers(id),
		FOREIGN KEY (subject_id) REFERENCES subjects(id)
	);

	CREATE TABLE IF NOT EXISTS subject_heads (
		subject_id INTEGER PRIMARY KEY,
		teacher_id INTEGER NOT NULL UNIQUE,
		FOREIGN KEY (teacher_id, subject_id)
			REFERENCES teacher_subjects(teacher_id, subject_id)
	);

	CREATE TABLE IF NOT EXISTS class_teachers (
		class_id INTEGER PRIMARY KEY,
		teacher_id INTEGER NOT NULL UNIQUE,
		FOREIGN KEY (class_id) REFERENCES classes(id),
		FOREIGN KEY (teacher_id) REFERENCES teachers(id)
	);

	CREATE TABLE IF NOT EXISTS class_subjects (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		class_id INTEGER NOT NULL,
		subject_id INTEGER NOT NULL,
		UNIQUE(class_id, subject_id),
		UNIQUE(id, subject_id),
		FOREIGN KEY (class_id) REFERENCES classes(id),
		FOREIGN KEY (subject_id) REFERENCES subjects(id)
	);

	CREATE TABLE IF NOT EXISTS class_subject_teachers (
		class_subject_id INTEGER NOT NULL,
		teacher_id INTEGER NOT NULL,
		subject_id INTEGER NOT NULL,
		PRIMARY KEY(class_subject_id, teacher_id),
		FOREIGN KEY (class_subject_id, subject_id)
			REFERENCES class_subjects(id, subject_id),
		FOREIGN KEY (teacher_id, subject_id)
			REFERENCES teacher_subjects(teacher_id, subject_id)
	);
	`

	if _, err := tx.ExecContext(ctx, schema); err != nil {
		fail(err)
	}

	if err := tx.Commit(); err != nil {
		log.Fatal(err)
	}

	var after int
	err = conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM students").Scan(&after)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Students after migration: %d\n", after)

	if before != after {
		log.Fatalf("Student count changed: before=%d, after=%d", before, after)
	}

	rows, err := conn.QueryContext(ctx, "PRAGMA foreign_key_check")
	if err != nil {
		log.Fatal(err)
	}

	if rows.Next() {
		var table, parent string
		var rowID sql.NullInt64
		var fkID int

		if err := rows.Scan(&table, &rowID, &parent, &fkID); err != nil {
			rows.Close()
			log.Fatal(err)
		}

		rows.Close()
		log.Fatalf("Foreign key violation: table=%s row=%v parent=%s",
			table, rowID, parent)
	}

	if err := rows.Err(); err != nil {
		rows.Close()
		log.Fatal(err)
	}
	rows.Close()

	fmt.Println("Phase 2 database migration completed successfully.")
}
