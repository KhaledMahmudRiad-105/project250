package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"

	"github.com/KhaledMahmudRiad-105/project250/internal/db"
	"github.com/KhaledMahmudRiad-105/project250/internal/handlers"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./school.db"
	}

	database, err := db.Open(dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()

	page, err := template.ParseFiles("templates/index.html")
	if err != nil {
		log.Fatal(err)
	}

	studentHandler, err := handlers.NewStudentHandler(database)
	if err != nil {
		log.Fatal(err)
	}

	teacherHandler, err := handlers.NewTeacherHandler(database)
	if err != nil {
		log.Fatal(err)
	}

	subjectHandler, err := handlers.NewSubjectHandler(database)
	if err != nil {
		log.Fatal(err)
	}

	teacherSubjectHandler, err := handlers.NewTeacherSubjectHandler(database)
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		if err := database.Ping(); err != nil {
			http.Error(w, "Database unavailable", http.StatusServiceUnavailable)
			return
		}

		fmt.Fprint(w, "ok")
	})

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		if err := page.Execute(w, nil); err != nil {
			log.Println("Rendering home page:", err)
		}
	})

	mux.HandleFunc("GET /students", studentHandler.List)
	mux.HandleFunc("POST /students", studentHandler.Create)
	mux.HandleFunc("GET /students/edit", studentHandler.EditForm)
	mux.HandleFunc("POST /students/edit", studentHandler.Update)
	mux.HandleFunc("POST /students/delete", studentHandler.Delete)

	mux.HandleFunc("GET /teachers", teacherHandler.List)
	mux.HandleFunc("POST /teachers", teacherHandler.Create)
	mux.HandleFunc("GET /teachers/edit", teacherHandler.EditForm)
	mux.HandleFunc("POST /teachers/edit", teacherHandler.Update)
	mux.HandleFunc("POST /teachers/delete", teacherHandler.Delete)

	mux.HandleFunc("GET /subjects", subjectHandler.List)
	mux.HandleFunc("POST /subjects", subjectHandler.Create)
	mux.HandleFunc("GET /subjects/edit", subjectHandler.EditForm)
	mux.HandleFunc("POST /subjects/edit", subjectHandler.Update)
	mux.HandleFunc("POST /subjects/delete", subjectHandler.Delete)

	mux.HandleFunc("GET /teacher-subjects", teacherSubjectHandler.List)
	mux.HandleFunc("POST /teacher-subjects/assign", teacherSubjectHandler.Assign)
	mux.HandleFunc("POST /teacher-subjects/remove", teacherSubjectHandler.Remove)

	log.Printf("Server running at http://localhost:%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
