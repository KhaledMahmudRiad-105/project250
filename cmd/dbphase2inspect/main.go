
package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "modernc.org/sqlite"
)

func main() {
	db, err := sql.Open("sqlite", "./school.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	rows, err := db.Query(`
		SELECT name
		FROM sqlite_master
		WHERE type = 'table'
		AND name NOT LIKE 'sqlite_%'
		ORDER BY name
	`)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Database tables:")

	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			rows.Close()
			log.Fatal(err)
		}
		fmt.Println("-", name)
	}

	if err := rows.Err(); err != nil {
		rows.Close()
		log.Fatal(err)
	}
	rows.Close()

	var students int
	if err := db.QueryRow("SELECT COUNT(*) FROM students").Scan(&students); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("\nTotal students: %d\n", students)

	fkRows, err := db.Query("PRAGMA foreign_key_check")
	if err != nil {
		log.Fatal(err)
	}
	defer fkRows.Close()

	violations := 0

	for fkRows.Next() {
		var table, parent string
		var rowID sql.NullInt64
		var fkID int

		if err := fkRows.Scan(&table, &rowID, &parent, &fkID); err != nil {
			log.Fatal(err)
		}

		fmt.Printf("Foreign key violation: %s -> %s\n", table, parent)
		violations++
	}

	if err := fkRows.Err(); err != nil {
		log.Fatal(err)
	}

	if violations == 0 {
		fmt.Println("Foreign key check: PASS")
	} else {
		log.Fatalf("Foreign key check failed: %d violation(s)", violations)
	}
}
