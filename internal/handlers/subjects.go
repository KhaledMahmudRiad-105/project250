package handlers

import (
	"database/sql"
	"html/template"
	"net/http"
	"strconv"
	"strings"
)

type Subject struct {
	ID   int
	Name string
}

type SubjectHandler struct {
	DB       *sql.DB
	Page     *template.Template
	EditPage *template.Template
}

type SubjectPageData struct {
	Subjects []Subject
	Message  string
}

func NewSubjectHandler(database *sql.DB) (*SubjectHandler, error) {
	page, err := template.ParseFiles("templates/subjects.html")
	if err != nil {
		return nil, err
	}

	editPage, err := template.ParseFiles("templates/subject_edit.html")
	if err != nil {
		return nil, err
	}

	return &SubjectHandler{
		DB:       database,
		Page:     page,
		EditPage: editPage,
	}, nil
}

func (h *SubjectHandler) List(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query(`
		SELECT id, name
		FROM subjects
		ORDER BY id DESC
	`)
	if err != nil {
		http.Error(w, "Failed to load subjects", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	subjects := []Subject{}

	for rows.Next() {
		var s Subject

		if err := rows.Scan(&s.ID, &s.Name); err != nil {
			http.Error(w, "Failed to read subjects", http.StatusInternalServerError)
			return
		}

		subjects = append(subjects, s)
	}

	if err := rows.Err(); err != nil {
		http.Error(w, "Failed to read subjects", http.StatusInternalServerError)
		return
	}

	data := SubjectPageData{
		Subjects: subjects,
		Message:  r.URL.Query().Get("message"),
	}

	if err := h.Page.Execute(w, data); err != nil {
		http.Error(w, "Failed to render subjects", http.StatusInternalServerError)
	}
}

func (h *SubjectHandler) Create(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))

	if name == "" || len(name) > 100 {
		http.Error(w, "Subject name is required and must be at most 100 characters", http.StatusBadRequest)
		return
	}

	_, err := h.DB.Exec(`
		INSERT INTO subjects (name)
		VALUES (?)
	`, name)

	if err != nil {
		http.Error(w, "Could not add subject. It may already exist.", http.StatusConflict)
		return
	}

	http.Redirect(w, r, "/subjects?message=Subject+added+successfully", http.StatusSeeOther)
}

func (h *SubjectHandler) EditForm(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil || id < 1 {
		http.Error(w, "Invalid subject ID", http.StatusBadRequest)
		return
	}

	var s Subject

	err = h.DB.QueryRow(`
		SELECT id, name
		FROM subjects
		WHERE id = ?
	`, id).Scan(&s.ID, &s.Name)

	if err == sql.ErrNoRows {
		http.NotFound(w, r)
		return
	}

	if err != nil {
		http.Error(w, "Failed to load subject", http.StatusInternalServerError)
		return
	}

	if err := h.EditPage.Execute(w, s); err != nil {
		http.Error(w, "Failed to render edit page", http.StatusInternalServerError)
	}
}

func (h *SubjectHandler) Update(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(r.FormValue("id"))
	if err != nil || id < 1 {
		http.Error(w, "Invalid subject ID", http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))

	if name == "" || len(name) > 100 {
		http.Error(w, "Subject name is required and must be at most 100 characters", http.StatusBadRequest)
		return
	}

	result, err := h.DB.Exec(`
		UPDATE subjects
		SET name = ?
		WHERE id = ?
	`, name, id)

	if err != nil {
		http.Error(w, "Could not update subject. The name may already exist.", http.StatusConflict)
		return
	}

	affected, err := result.RowsAffected()
	if err != nil {
		http.Error(w, "Failed to update subject", http.StatusInternalServerError)
		return
	}

	if affected == 0 {
		http.NotFound(w, r)
		return
	}

	http.Redirect(w, r, "/subjects?message=Subject+updated+successfully", http.StatusSeeOther)
}

func (h *SubjectHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(r.FormValue("id"))
	if err != nil || id < 1 {
		http.Error(w, "Invalid subject ID", http.StatusBadRequest)
		return
	}

	var assignments int

	err = h.DB.QueryRow(`
		SELECT
			(SELECT COUNT(*) FROM teacher_subjects WHERE subject_id = ?) +
			(SELECT COUNT(*) FROM subject_heads WHERE subject_id = ?) +
			(SELECT COUNT(*) FROM class_subjects WHERE subject_id = ?)
	`, id, id, id).Scan(&assignments)

	if err != nil {
		http.Error(w, "Failed to check subject assignments", http.StatusInternalServerError)
		return
	}

	if assignments > 0 {
		http.Error(w, "Cannot delete this subject because it has existing assignments", http.StatusConflict)
		return
	}

	result, err := h.DB.Exec(`
		DELETE FROM subjects
		WHERE id = ?
	`, id)

	if err != nil {
		http.Error(w, "Could not delete subject", http.StatusInternalServerError)
		return
	}

	affected, err := result.RowsAffected()
	if err != nil {
		http.Error(w, "Failed to delete subject", http.StatusInternalServerError)
		return
	}

	if affected == 0 {
		http.NotFound(w, r)
		return
	}

	http.Redirect(w, r, "/subjects?message=Subject+deleted+successfully", http.StatusSeeOther)
}
