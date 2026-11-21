
package handlers

import (
	"database/sql"
	"html/template"
	"net/http"
	"strconv"
	"strings"
)

type Teacher struct {
	ID    int
	Name  string
	Email string
}

type TeacherHandler struct {
	DB       *sql.DB
	Page     *template.Template
	EditPage *template.Template
}

type TeacherPageData struct {
	Teachers []Teacher
	Message  string
}

func NewTeacherHandler(database *sql.DB) (*TeacherHandler, error) {
	page, err := template.ParseFiles("templates/teachers.html")
	if err != nil {
		return nil, err
	}

	editPage, err := template.ParseFiles("templates/teacher_edit.html")
	if err != nil {
		return nil, err
	}

	return &TeacherHandler{
		DB:       database,
		Page:     page,
		EditPage: editPage,
	}, nil
}

func (h *TeacherHandler) List(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query(`
		SELECT id, name, COALESCE(email, '')
		FROM teachers
		ORDER BY id DESC
	`)
	if err != nil {
		http.Error(w, "Could not load teachers", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	teachers := []Teacher{}

	for rows.Next() {
		var t Teacher

		if err := rows.Scan(&t.ID, &t.Name, &t.Email); err != nil {
			http.Error(w, "Could not read teachers", http.StatusInternalServerError)
			return
		}

		teachers = append(teachers, t)
	}

	if err := rows.Err(); err != nil {
		http.Error(w, "Could not read teachers", http.StatusInternalServerError)
		return
	}

	data := TeacherPageData{
		Teachers: teachers,
		Message:  r.URL.Query().Get("message"),
	}

	if err := h.Page.Execute(w, data); err != nil {
		http.Error(w, "Could not render teachers", http.StatusInternalServerError)
	}
}

func (h *TeacherHandler) Create(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	email := strings.TrimSpace(r.FormValue("email"))

	if name == "" || len(name) > 100 || len(email) > 254 {
		http.Error(w, "Enter valid teacher details", http.StatusBadRequest)
		return
	}

	_, err := h.DB.Exec(`
		INSERT INTO teachers (name, email)
		VALUES (?, NULLIF(?, ''))
	`, name, email)

	if err != nil {
		http.Error(w, "Could not save teacher. Check whether the email already exists.", http.StatusBadRequest)
		return
	}

	http.Redirect(w, r, "/teachers?message=Teacher+added+successfully", http.StatusSeeOther)
}

func (h *TeacherHandler) EditForm(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil || id < 1 {
		http.Error(w, "Invalid teacher ID", http.StatusBadRequest)
		return
	}

	var t Teacher

	err = h.DB.QueryRow(`
		SELECT id, name, COALESCE(email, '')
		FROM teachers
		WHERE id = ?
	`, id).Scan(&t.ID, &t.Name, &t.Email)

	if err == sql.ErrNoRows {
		http.NotFound(w, r)
		return
	}

	if err != nil {
		http.Error(w, "Could not load teacher", http.StatusInternalServerError)
		return
	}

	if err := h.EditPage.Execute(w, t); err != nil {
		http.Error(w, "Could not render edit page", http.StatusInternalServerError)
	}
}

func (h *TeacherHandler) Update(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(r.FormValue("id"))
	if err != nil || id < 1 {
		http.Error(w, "Invalid teacher ID", http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	email := strings.TrimSpace(r.FormValue("email"))

	if name == "" || len(name) > 100 || len(email) > 254 {
		http.Error(w, "Enter valid teacher details", http.StatusBadRequest)
		return
	}

	result, err := h.DB.Exec(`
		UPDATE teachers
		SET name = ?, email = NULLIF(?, '')
		WHERE id = ?
	`, name, email, id)

	if err != nil {
		http.Error(w, "Could not update teacher. Check whether the email already exists.", http.StatusBadRequest)
		return
	}

	affected, err := result.RowsAffected()
	if err != nil {
		http.Error(w, "Could not verify update", http.StatusInternalServerError)
		return
	}

	if affected == 0 {
		http.NotFound(w, r)
		return
	}

	http.Redirect(w, r, "/teachers?message=Teacher+updated+successfully", http.StatusSeeOther)
}

func (h *TeacherHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(r.FormValue("id"))
	if err != nil || id < 1 {
		http.Error(w, "Invalid teacher ID", http.StatusBadRequest)
		return
	}

	var count int

	err = h.DB.QueryRow(`
		SELECT
			(SELECT COUNT(*) FROM school_roles WHERE teacher_id = ?) +
			(SELECT COUNT(*) FROM teacher_subjects WHERE teacher_id = ?) +
			(SELECT COUNT(*) FROM subject_heads WHERE teacher_id = ?) +
			(SELECT COUNT(*) FROM class_teachers WHERE teacher_id = ?) +
			(SELECT COUNT(*) FROM class_subject_teachers WHERE teacher_id = ?)
	`, id, id, id, id, id).Scan(&count)

	if err != nil {
		http.Error(w, "Could not check teacher assignments", http.StatusInternalServerError)
		return
	}

	if count > 0 {
		http.Error(w, "This teacher has school roles or academic assignments and cannot be deleted.", http.StatusConflict)
		return
	}

	result, err := h.DB.Exec("DELETE FROM teachers WHERE id = ?", id)
	if err != nil {
		http.Error(w, "Could not delete teacher", http.StatusInternalServerError)
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

	http.Redirect(w, r, "/teachers?message=Teacher+deleted+successfully", http.StatusSeeOther)
}
