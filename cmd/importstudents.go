
package main

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strings"

	_ "modernc.org/sqlite"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	file, err := os.Open("./students_import.csv")
	if err != nil {
		return fmt.Errorf("opening CSV: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.TrimLeadingSpace = true

	header, err := reader.Read()
	if err != nil {
		return fmt.Errorf("reading CSV header: %w", err)
	}

	if len(header) != 3 ||
		strings.TrimSpace(header[0]) != "class" ||
		strings.TrimSpace(header[1]) != "roll" ||
		strings.TrimSpace(header[2]) != "student_name" {
		return fmt.Errorf("unexpected CSV header; expected class,roll,student_name")
	}

	type Student struct {
		class string
		roll  string
		name  string
	}

	var students []Student
	line := 1

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("reading CSV near line %d: %w", line+1, err)
		}
		line++

		if len(record) != 3 {
			return fmt.Errorf("line %d: expected 3 columns", line)
		}

		student := Student{
			class: strings.TrimSpace(record[0]),
			roll:  strings.TrimSpace(record[1]),
			name:  strings.TrimSpace(record[2]),
		}

		if student.class == "" || student.roll == "" || student.name == "" {
			return fmt.Errorf("line %d: class, roll, and student name are required", line)
		}

		students = append(students, student)
	}

	if len(students) == 0 {
		return fmt.Errorf("CSV contains no student records")
	}

	database, err := sql.Open("sqlite", "./school.db")
	if err != nil {
		return err
	}
	defer database.Close()

	database.SetMaxOpenConns(1)

	if err := database.Ping(); err != nil {
		return err
	}

	tx, err := database.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	imported := 0
	skipped := 0
	classCounts := make(map[string]int)

	for _, student := range students {
		_, err := tx.Exec(
			`INSERT INTO classes (name) VALUES (?) ON CONFLICT(name) DO NOTHING`,
			student.class,
		)
		if err != nil {
			return fmt.Errorf("creating class %q: %w", student.class, err)
		}

		var classID int
		err = tx.QueryRow(
			`SELECT id FROM classes WHERE name = ?`,
			student.class,
		).Scan(&classID)
		if err != nil {
			return fmt.Errorf("finding class %q: %w", student.class, err)
		}

		var existingName string
		err = tx.QueryRow(
			`SELECT name FROM students WHERE class_id = ? AND roll_number = ?`,
			classID, student.roll,
		).Scan(&existingName)

		if err == nil {
			if existingName != student.name {
				return fmt.Errorf(
					"conflict: %s roll %s already belongs to %q, not %q; no CSV changes were committed",
					student.class, student.roll, existingName, student.name,
				)
			}

			skipped++
			continue
		}

		if err != sql.ErrNoRows {
			return fmt.Errorf("checking %s roll %s: %w",
				student.class, student.roll, err)
		}

		_, err = tx.Exec(
			`INSERT INTO students (name, roll_number, class_id) VALUES (?, ?, ?)`,
			student.name, student.roll, classID,
		)
		if err != nil {
			return fmt.Errorf("inserting %s roll %s: %w",
				student.class, student.roll, err)
		}

		imported++
		classCounts[student.class]++
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing import: %w", err)
	}

	fmt.Println("Student import completed successfully.")
	fmt.Printf("CSV records processed: %d\n", len(students))
	fmt.Printf("New students imported: %d\n", imported)
	fmt.Printf("Existing identical records skipped: %d\n", skipped)

	classes := make([]string, 0, len(classCounts))
	for class := range classCounts {
		classes = append(classes, class)
	}
	sort.Strings(classes)

	fmt.Println("\nNew students imported per class:")
	for _, class := range classes {
		fmt.Printf("%s: %d\n", class, classCounts[class])
	}

	return nil
}
