package handlers

import (
	"database/sql"
	"html/template"
	"net/http"
	"strconv"
)

type TeacherSubjectAssignment struct {
	TeacherID   int
	TeacherName string
	SubjectName string
	Assigned    bool
}

type TeacherSubjectData struct {
	Assignments []TeacherSubjectAssignment
	Subjects    []Subject
	Message     string
}

type TeacherSubjectHandler struct {
	DB   *sql.DB
	Page *template.Template
}

func NewTeacherSubjectHandler(database *sql.DB) (*TeacherSubjectHandler, error) {
	page, err := template.ParseFiles("templates/teacher_subjects.html")
	if err != nil {
		return nil, err
	}

	return &TeacherSubjectHandler{
		DB:   database,
		Page: page,
	}, nil
}

func (h *TeacherSubjectHandler) List(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query(`
		SELECT t.id, t.name, COALESCE(s.name, ''), 
		       CASE WHEN ts.teacher_id IS NULL THEN 0 ELSE 1 END
		FROM teachers t
		LEFT JOIN teacher_subjects ts ON ts.teacher_id = t.id
		LEFT JOIN subjects s ON s.id = ts.subject_id
		ORDER BY t.id
	`)
	if err != nil {
		http.Error(w, "Could not load teacher assignments", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	assignments := []TeacherSubjectAssignment{}

	for rows.Next() {
		var a TeacherSubjectAssignment

		if err := rows.Scan(&a.TeacherID, &a.TeacherName, &a.SubjectName, &a.Assigned); err != nil {
			http.Error(w, "Could not read teacher assignments", http.StatusInternalServerError)
			return
		}

		assignments = append(assignments, a)
	}

	if err := rows.Err(); err != nil {
		http.Error(w, "Could not read teacher assignments", http.StatusInternalServerError)
		return
	}

	subjectRows, err := h.DB.Query(`
		SELECT id, name
		FROM subjects
		ORDER BY name
	`)
	if err != nil {
		http.Error(w, "Could not load subjects", http.StatusInternalServerError)
		return
	}
	defer subjectRows.Close()

	subjects := []Subject{}

	for subjectRows.Next() {
		var s Subject

		if err := subjectRows.Scan(&s.ID, &s.Name); err != nil {
			http.Error(w, "Could not read subjects", http.StatusInternalServerError)
			return
		}

		subjects = append(subjects, s)
	}

	if err := subjectRows.Err(); err != nil {
		http.Error(w, "Could not read subjects", http.StatusInternalServerError)
		return
	}

	data := TeacherSubjectData{
		Assignments: assignments,
		Subjects:    subjects,
		Message:     r.URL.Query().Get("message"),
	}

	if err := h.Page.Execute(w, data); err != nil {
		http.Error(w, "Could not render assignments", http.StatusInternalServerError)
	}
}

func (h *TeacherSubjectHandler) Assign(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	teacherID, err := strconv.Atoi(r.FormValue("teacher_id"))
	if err != nil || teacherID < 1 {
		http.Error(w, "Invalid teacher ID", http.StatusBadRequest)
		return
	}

	subjectID, err := strconv.Atoi(r.FormValue("subject_id"))
	if err != nil || subjectID < 1 {
		http.Error(w, "Invalid subject ID", http.StatusBadRequest)
		return
	}

	_, err = h.DB.Exec(`
		INSERT INTO teacher_subjects (teacher_id, subject_id)
		VALUES (?, ?)
		ON CONFLICT(teacher_id) DO UPDATE SET subject_id = excluded.subject_id
	`, teacherID, subjectID)

	if err != nil {
		http.Error(w, "Could not assign subject to teacher", http.StatusBadRequest)
		return
	}

	http.Redirect(w, r, "/teacher-subjects?message=Subject+assigned+successfully", http.StatusSeeOther)
}

func (h *TeacherSubjectHandler) Remove(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	teacherID, err := strconv.Atoi(r.FormValue("teacher_id"))
	if err != nil || teacherID < 1 {
		http.Error(w, "Invalid teacher ID", http.StatusBadRequest)
		return
	}

	result, err := h.DB.Exec(`
		DELETE FROM teacher_subjects
		WHERE teacher_id = ?
	`, teacherID)

	if err != nil {
		http.Error(w, "Could not remove subject assignment", http.StatusConflict)
		return
	}

	affected, err := result.RowsAffected()
	if err != nil {
		http.Error(w, "Could not verify removal", http.StatusInternalServerError)
		return
	}

	if affected == 0 {
		http.NotFound(w, r)
		return
	}

	http.Redirect(w, r, "/teacher-subjects?message=Assignment+removed+successfully", http.StatusSeeOther)
}
