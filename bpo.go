package main

import (
	"bpo/bpo" // generated code via sqlc
	"context"
	"database/sql"
	_ "embed"
	"html/template"
	"log"
	_ "modernc.org/sqlite"
	"net/http"
	"strconv"
	"time"
)

type Server struct {
	queries *bpo.Queries
}

type TemplateContext struct {
	Bpos     []bpo.BloodPressureObservation
	Editable bool
}

//go:embed schema.sql
var ddl string

var templates = template.Must(template.ParseFiles("bpos.html"))

func (s *Server) getHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Executing getHandler()")

	editableString := r.FormValue("editable")
	editable, err := strconv.ParseBool(editableString)

	bpos, err := s.queries.GetBloodPressureObservations(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	templateContext := TemplateContext{Bpos: bpos, Editable: editable}

	err = templates.ExecuteTemplate(w, "bpos.html", templateContext)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) deleteHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Executing deleteHandler()")

	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	err = s.queries.DeleteBloodPressureObservation(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", 303)
}

func (s *Server) postHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Executing postHandler()")

	observedAt, err := time.Parse("2006-01-02T15:04", r.PostFormValue("observedAt"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	systolic, err := strconv.Atoi(r.PostFormValue("systolic"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	diastolic, err := strconv.Atoi(r.PostFormValue("diastolic"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	pulse, err := strconv.Atoi(r.PostFormValue("pulse"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	irregular := r.PostFormValue("irregular") != ""
	comment := r.FormValue("comment")

	var id int = 0
	log.Println(r.URL)
	if idPath := r.PathValue("id"); idPath != "" {
		log.Println(r.PathValue("Detected ID in URL. Converting to int..."))
		id, err = strconv.Atoi(idPath)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
	}

	log.Printf(
		"Parsed id=%d systolic=%d, diastolic=%d, pulse=%d, irregular=%s comment=%s\n",
		id,
		systolic,
		diastolic,
		pulse,
		irregular,
		comment,
	)

	if id == 0 {
		log.Println("Creating new observation...")
		observation := bpo.CreateBloodPressureObservationParams{
			ObservedAt: observedAt,
			Systolic:   systolic,
			Diastolic:  diastolic,
			Pulse:      pulse,
			Irregular:  irregular,
			Comment:    comment,
		}

		log.Println(observation)
		_, err = s.queries.CreateBloodPressureObservation(r.Context(), observation)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		log.Println("Updating observation...")
		observation := bpo.UpdateBloodPressureObservationParams{
			ObservedAt: observedAt,
			ID:         id,
			Systolic:   systolic,
			Diastolic:  diastolic,
			Pulse:      pulse,
			Irregular:  irregular,
			Comment:    comment,
		}

		log.Println(observation)
		_, err = s.queries.UpdateBloodPressureObservation(r.Context(), observation)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	http.Redirect(w, r, "/", 303)
}

func main() {
	ctx := context.Background()
	db, err := sql.Open("sqlite", "bpo.db")
	if err != nil {

		log.Fatal(err)

	}
	defer db.Close() // not sure if actually necessary
	db.Ping()
	db.ExecContext(ctx, ddl)
	queries := bpo.New(db)
	server := &Server{queries: queries}

	http.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	http.HandleFunc("GET /", server.getHandler)
	http.HandleFunc("POST /{id}/delete", server.deleteHandler)
	http.HandleFunc("POST /{id}/update", server.postHandler)
	http.HandleFunc("POST /", server.postHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
