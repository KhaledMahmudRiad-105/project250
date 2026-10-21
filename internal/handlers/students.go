package handlers

import (
    "database/sql"
    "html/template"
    "net/http"
    "strings"
)

type Student struct {
    ID         int
    Name       string
    RollNumber string
    ClassName  string
}

type StudentHandler struct {
    DB   *sql.DB
    Page *template.Template
}

type StudentPageData struct {
    Students []Student
    Error    string
    Message  string
}

func NewStudentHandler(database *sql.DB) (*StudentHandler, error) {
    page, err := template.ParseFiles("templates/students.html")
    if err != nil {
        return nil, err
    }

    return &StudentHandler{
        DB:   database,
        Page: page,
    }, nil
}

func (h *StudentHandler) List(w http.ResponseWriter, r *http.Request) {
    h.render(w, r, "")
}

func (h *StudentHandler) Create(w http.ResponseWriter, r *http.Request) {
    if err := r.ParseForm(); err != nil {
        http.Error(w, "Invalid form data", http.StatusBadRequest)
        return
    }

    name := strings.TrimSpace(r.FormValue("name"))
    roll := strings.TrimSpace(r.FormValue("roll_number"))
    className := strings.TrimSpace(r.FormValue("class_name"))

    if name == "" || roll == "" || className == "" {
        h.render(w, r, "All fields are required.")
        return
    }

    tx, err := h.DB.Begin()
    if err != nil {
        http.Error(w, "Database error", http.StatusInternalServerError)
        return
    }
    defer tx.Rollback()

    _, err = tx.Exec(
        "INSERT INTO classes (name) VALUES (?) ON CONFLICT(name) DO NOTHING",
        className,
    )
    if err != nil {
        http.Error(w, "Could not save class", http.StatusInternalServerError)
        return
    }

    var classID int
    err = tx.QueryRow(
        "SELECT id FROM classes WHERE name = ?",
        className,
    ).Scan(&classID)
    if err != nil {
        http.Error(w, "Could not find class", http.StatusInternalServerError)
        return
    }

    _, err = tx.Exec(
        "INSERT INTO students (name, roll_number, class_id) VALUES (?, ?, ?)",
        name, roll, classID,
    )
    if err != nil {
        h.render(w, r, "Could not register student. The roll number may already exist.")
        return
    }

    if err := tx.Commit(); err != nil {
        http.Error(w, "Could not save student", http.StatusInternalServerError)
        return
    }

    http.Redirect(w, r, "/students?message=Student+registered+successfully", http.StatusSeeOther)
}

func (h *StudentHandler) render(w http.ResponseWriter, r *http.Request, errorMessage string) {
    rows, err := h.DB.Query(`
        SELECT students.id, students.name, students.roll_number, classes.name
        FROM students
        JOIN classes ON students.class_id = classes.id
        ORDER BY students.id DESC
    `)
    if err != nil {
        http.Error(w, "Could not load students", http.StatusInternalServerError)
        return
    }
    defer rows.Close()

    students := []Student{}

    for rows.Next() {
        var student Student

        err := rows.Scan(
            &student.ID,
            &student.Name,
            &student.RollNumber,
            &student.ClassName,
        )
        if err != nil {
            http.Error(w, "Could not read students", http.StatusInternalServerError)
            return
        }

        students = append(students, student)
    }

    if err := rows.Err(); err != nil {
        http.Error(w, "Could not read students", http.StatusInternalServerError)
        return
    }

    data := StudentPageData{
        Students: students,
        Error:    errorMessage,
        Message:  r.URL.Query().Get("message"),
    }

    if err := h.Page.Execute(w, data); err != nil {
        http.Error(w, "Could not render page", http.StatusInternalServerError)
    }
}
