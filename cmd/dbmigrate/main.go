
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

	var count int
	if err := conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM students").Scan(&count); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Students before migration: %d\n", count)

	if _, err := conn.ExecContext(ctx, "PRAGMA foreign_keys = OFF"); err != nil {
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

	_, err = tx.ExecContext(ctx, `
		CREATE TABLE students_new (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			roll_number TEXT NOT NULL,
			class_id INTEGER NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (class_id) REFERENCES classes(id),
			UNIQUE(class_id, roll_number)
		)
	`)
	if err != nil {
		fail(err)
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO students_new (id, name, roll_number, class_id, created_at)
		SELECT id, name, roll_number, class_id, created_at
		FROM students
	`)
	if err != nil {
		fail(err)
	}

	_, err = tx.ExecContext(ctx, "DROP TABLE students")
	if err != nil {
		fail(err)
	}

	_, err = tx.ExecContext(ctx, "ALTER TABLE students_new RENAME TO students")
	if err != nil {
		fail(err)
	}

	_, err = tx.ExecContext(ctx, "CREATE INDEX idx_students_class ON students(class_id)")
	if err != nil {
		fail(err)
	}

	if err := tx.Commit(); err != nil {
		log.Fatal(err)
	}

	if _, err := conn.ExecContext(ctx, "PRAGMA foreign_keys = ON"); err != nil {
		log.Fatal(err)
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
		log.Fatalf("Foreign key violation: table=%s row=%v parent=%s", table, rowID, parent)
	}

	if err := rows.Err(); err != nil {
		rows.Close()
		log.Fatal(err)
	}
	rows.Close()

	var after int
	if err := conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM students").Scan(&after); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Students after migration: %d\n", after)
	if count != after {
		log.Fatalf("Student count changed; expected %d, got %d", count, after)
	}

	fmt.Println("Migration completed successfully.")
}
