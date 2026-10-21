package main

import (
    "html/template"
    "log"
    "net/http"
    "os"
)

func main() {
    port := os.Getenv("PORT")

    if port == "" {
        port = "8080"
    }

    page, err := template.ParseFiles("templates/index.html")
    if err != nil {
        log.Fatal(err)
    }

    mux := http.NewServeMux()

    mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("ok"))
    })

    mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
        if r.URL.Path != "/" {
            http.NotFound(w, r)
            return
        }

        err := page.Execute(w, nil)
        if err != nil {
            log.Println(err)
        }
    })

    log.Printf("Server running at http://localhost:%s\n", port)
    log.Fatal(http.ListenAndServe(":"+port, mux))
}
