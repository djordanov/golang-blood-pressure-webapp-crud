package main

import (
	"bpo/bpo" // generated code via sqlc
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type Server struct {
	queries *bpo.Queries
}

type TemplateContext struct {
	Bpos     []bpo.BloodPressureObservation
	Editable bool
}

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

	err = s.queries.DeleteBloodPressureObservation(r.Context(), int32(id))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", 303)
}

func (s *Server) postHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Executing postHandler()")
	log.Println(r.URL)

	datetimestr := r.PostFormValue("date") + "T" + r.PostFormValue("time")
	log.Println(datetimestr)
	observedAt, err := time.Parse("2006-01-02T15:04", datetimestr) // local time from the users perspective, do not track timezone or convert
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
			ObservedAt: pgtype.Timestamp{Time: observedAt, Valid: true},
			Systolic:   int32(systolic),
			Diastolic:  int32(diastolic),
			Pulse:      int32(pulse),
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
			ObservedAt: pgtype.Timestamp{Time: observedAt, Valid: true},
			ID:         int32(id),
			Systolic:   int32(systolic),
			Diastolic:  int32(diastolic),
			Pulse:      int32(pulse),
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

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func main() {
	ctx := context.Background()
	connStr := fmt.Sprintf(
		"host=%s port=5432 user=%s password=%s dbname=%s sslmode=disable",
		getEnv("POSTGRES_HOST", "localhost"),
		getEnv("POSTGRES_USER", "gbpw"),
		getEnv("POSTGRES_PASSWORD", "gbpwassword"),
		getEnv("POSTGRES_DB", "gbpw"),
	)
	db, err := pgx.Connect(ctx, connStr)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close(ctx)
	db.Ping(ctx)
	queries := bpo.New(db)
	server := &Server{queries: queries}

	http.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	http.HandleFunc("GET /", server.getHandler)
	http.HandleFunc("POST /{id}/delete", server.deleteHandler)
	http.HandleFunc("POST /{id}/update", server.postHandler)
	http.HandleFunc("POST /", server.postHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
