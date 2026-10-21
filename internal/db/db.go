package db

import (
    "database/sql"
    "fmt"

    _ "modernc.org/sqlite"
)

func Open(path string) (*sql.DB, error) {
    database, err := sql.Open("sqlite", path)
    if err != nil {
        return nil, err
    }

    if err := database.Ping(); err != nil {
        database.Close()
        return nil, err
    }

    if _, err := database.Exec("PRAGMA foreign_keys = ON"); err != nil {
        database.Close()
        return nil, err
    }

    if err := createTables(database); err != nil {
        database.Close()
        return nil, err
    }

    return database, nil
}

func createTables(database *sql.DB) error {
    schema := `
    CREATE TABLE IF NOT EXISTS classes (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        name TEXT NOT NULL UNIQUE
    );

    CREATE TABLE IF NOT EXISTS teachers (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        name TEXT NOT NULL,
        email TEXT UNIQUE
    );

    CREATE TABLE IF NOT EXISTS students (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        name TEXT NOT NULL,
        roll_number TEXT NOT NULL UNIQUE,
        class_id INTEGER,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        FOREIGN KEY (class_id) REFERENCES classes(id)
    );

    CREATE TABLE IF NOT EXISTS attendance (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        student_id INTEGER NOT NULL,
        attendance_date TEXT NOT NULL,
        status TEXT NOT NULL CHECK(status IN ('present', 'absent', 'late')),
        FOREIGN KEY (student_id) REFERENCES students(id),
        UNIQUE(student_id, attendance_date)
    );

    CREATE TABLE IF NOT EXISTS exams (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        name TEXT NOT NULL,
        exam_date TEXT
    );

    CREATE TABLE IF NOT EXISTS marks (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        student_id INTEGER NOT NULL,
        exam_id INTEGER NOT NULL,
        subject TEXT NOT NULL,
        score REAL NOT NULL CHECK(score >= 0),
        max_score REAL NOT NULL CHECK(max_score > 0 AND score <= max_score),
        FOREIGN KEY (student_id) REFERENCES students(id),
        FOREIGN KEY (exam_id) REFERENCES exams(id),
        UNIQUE(student_id, exam_id, subject)
    );

    CREATE INDEX IF NOT EXISTS idx_students_class
        ON students(class_id);

    CREATE INDEX IF NOT EXISTS idx_attendance_date
        ON attendance(attendance_date);
    `

    if _, err := database.Exec(schema); err != nil {
        return fmt.Errorf("creating database tables: %w", err)
    }

    return nil
}
