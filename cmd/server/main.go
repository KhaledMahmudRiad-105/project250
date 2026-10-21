package main

import (
    "fmt"
    "html/template"
    "log"
    "net/http"
    "os"

    "github.com/KhaledMahmudRiad-105/project250/internal/db"
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

    log.Println("Database connected successfully")

    page, err := template.ParseFiles("templates/index.html")
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

    log.Printf("Server running at http://localhost:%s\n", port)
    log.Fatal(http.ListenAndServe(":"+port, mux))
}
