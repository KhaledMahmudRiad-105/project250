package handlers

import (
	"database/sql"
	"html/template"
	"net/http"
	"strconv"
	"strings"
)

type Student struct {
	ID         int
	Name       string
	RollNumber string
	ClassName  string
}

type StudentHandler struct {
	DB       *sql.DB
	Page     *template.Template
	EditPage *template.Template
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

	editPage, err := template.ParseFiles("templates/student_edit.html")
	if err != nil {
		return nil, err
	}

	return &StudentHandler{
		DB:       database,
		Page:     page,
		EditPage: editPage,
	}, nil
}

func (h *StudentHandler) List(w http.ResponseWriter, r *http.Request) {
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
		var s Student
		if err := rows.Scan(&s.ID, &s.Name, &s.RollNumber, &s.ClassName); err != nil {
			http.Error(w, "Could not read students", http.StatusInternalServerError)
			return
		}
		students = append(students, s)
	}

	if err := rows.Err(); err != nil {
		http.Error(w, "Could not read students", http.StatusInternalServerError)
		return
	}

	data := StudentPageData{
		Students: students,
		Message:  r.URL.Query().Get("message"),
	}

	if err := h.Page.Execute(w, data); err != nil {
		http.Error(w, "Could not render students", http.StatusInternalServerError)
	}
}

func (h *StudentHandler) Create(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	roll := strings.TrimSpace(r.FormValue("roll_number"))
	className := strings.TrimSpace(r.FormValue("class_name"))

	if name == "" || roll == "" || className == "" ||
		len(name) > 100 || len(roll) > 50 || len(className) > 100 {
		http.Error(w, "Enter valid values for all fields", http.StatusBadRequest)
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
	if err := tx.QueryRow("SELECT id FROM classes WHERE name = ?", className).Scan(&classID); err != nil {
		http.Error(w, "Could not find class", http.StatusInternalServerError)
		return
	}

	_, err = tx.Exec(
		"INSERT INTO students (name, roll_number, class_id) VALUES (?, ?, ?)",
		name, roll, classID,
	)
	if err != nil {
		http.Error(w, "Could not register student. Check whether the roll number already exists.", http.StatusBadRequest)
		return
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, "Could not save student", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/students?message=Student+registered+successfully", http.StatusSeeOther)
}

func (h *StudentHandler) EditForm(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil || id < 1 {
		http.Error(w, "Invalid student ID", http.StatusBadRequest)
		return
	}

	var s Student
	err = h.DB.QueryRow(`
		SELECT students.id, students.name, students.roll_number, classes.name
		FROM students
		JOIN classes ON students.class_id = classes.id
		WHERE students.id = ?
	`, id).Scan(&s.ID, &s.Name, &s.RollNumber, &s.ClassName)

	if err == sql.ErrNoRows {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "Could not load student", http.StatusInternalServerError)
		return
	}

	if err := h.EditPage.Execute(w, s); err != nil {
		http.Error(w, "Could not render edit page", http.StatusInternalServerError)
	}
}

func (h *StudentHandler) Update(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(r.FormValue("id"))
	if err != nil || id < 1 {
		http.Error(w, "Invalid student ID", http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	roll := strings.TrimSpace(r.FormValue("roll_number"))
	className := strings.TrimSpace(r.FormValue("class_name"))

	if name == "" || roll == "" || className == "" ||
		len(name) > 100 || len(roll) > 50 || len(className) > 100 {
		http.Error(w, "Enter valid values for all fields", http.StatusBadRequest)
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
	if err := tx.QueryRow("SELECT id FROM classes WHERE name = ?", className).Scan(&classID); err != nil {
		http.Error(w, "Could not find class", http.StatusInternalServerError)
		return
	}

	result, err := tx.Exec(`
		UPDATE students
		SET name = ?, roll_number = ?, class_id = ?
		WHERE id = ?
	`, name, roll, classID, id)
	if err != nil {
		http.Error(w, "Could not update student. The roll number may already exist.", http.StatusBadRequest)
		return
	}

	affected, err := result.RowsAffected()
	if err != nil {
		http.Error(w, "Could not verify update", http.StatusInternalServerError)
		return
	}
	if affected == 0 {
		http.Error(w, "Student not found", http.StatusNotFound)
		return
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, "Could not save changes", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/students?message=Student+updated+successfully", http.StatusSeeOther)
}

func (h *StudentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(r.FormValue("id"))
	if err != nil || id < 1 {
		http.Error(w, "Invalid student ID", http.StatusBadRequest)
		return
	}

	tx, err := h.DB.Begin()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	var count int
	err = tx.QueryRow(`
		SELECT
			(SELECT COUNT(*) FROM attendance WHERE student_id = ?) +
			(SELECT COUNT(*) FROM marks WHERE student_id = ?)
	`, id, id).Scan(&count)
	if err != nil {
		http.Error(w, "Could not check student records", http.StatusInternalServerError)
		return
	}

	if count > 0 {
		http.Error(w, "This student has attendance or marks records and cannot be deleted.", http.StatusConflict)
		return
	}

	result, err := tx.Exec("DELETE FROM students WHERE id = ?", id)
	if err != nil {
		http.Error(w, "Could not delete student", http.StatusInternalServerError)
		return
	}

	affected, err := result.RowsAffected()
	if err != nil {
		http.Error(w, "Could not verify deletion", http.StatusInternalServerError)
		return
	}
	if affected == 0 {
		http.NotFound(w, r)
		return
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, "Could not complete deletion", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/students?message=Student+deleted+successfully", http.StatusSeeOther)
}
