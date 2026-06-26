package main

import (
	"bpo/sqlc/bpo" // generated code via sqlc
	"context"
	"encoding/csv"
	"fmt"
	"github.com/jackc/pgx/v5"
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"
)

type DbConn struct {
	queries *bpo.Queries
}

type TemplateContext struct {
	Bpos     []bpo.GetBloodPressureObservationsRow
	Editable bool
}

var templates = template.Must(template.ParseFiles(
	"bpos.html",
	"bpo-row.html",
	"bpo-row-editable.html",
))

func (s *DbConn) getHandler(w http.ResponseWriter, r *http.Request) {
	editableString := r.FormValue("editable")
	editable, err := strconv.ParseBool(editableString)

	personID := r.Context().Value("PersonID").(int)
	bpos, err := s.queries.GetBloodPressureObservations(r.Context(), personID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	templateContext := TemplateContext{Bpos: bpos, Editable: editable}

	err = templates.ExecuteTemplate(w, "bpos.html", templateContext)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *DbConn) exportHandler(w http.ResponseWriter, r *http.Request) {
	personID := r.Context().Value("PersonID").(int)
	bpos, err := s.queries.GetBloodPressureObservations(r.Context(), personID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	slog.Debug("Fetched bpos, writing csv")
	writer := csv.NewWriter(w)
	writer.Comma = ';'
	err = writer.Write([]string{
		"ObservedAt",
		"Irregular Heartbeat",
		"Systolic",
		"Diastolic",
		"Pulse",
		"Comment",
	})

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for _, bpo := range bpos {
		err = writer.Write([]string{
			bpo.ObservedAt.Format("2006-01-02 15:04"),
			strconv.FormatBool(bpo.Irregular),
			strconv.Itoa(int(bpo.Systolic)),
			strconv.Itoa(int(bpo.Diastolic)),
			strconv.Itoa(int(bpo.Pulse)),
			bpo.Comment,
		})

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	slog.Debug("Setting csv headers...")
	w.Header().Set("Content-Disposition", "attachment; filename=blood_pressure.csv")
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Transfer-Encoding", "chunked")

	writer.Flush()
}

func (s *DbConn) deleteHandler(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	personID := r.Context().Value("PersonID").(int)
	err = s.queries.DeleteBloodPressureObservation(r.Context(), bpo.DeleteBloodPressureObservationParams{
		ID:       id,
		PersonID: personID,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", 303)
}

func (s *DbConn) postHandler(w http.ResponseWriter, r *http.Request) {
	personID := r.Context().Value("PersonID").(int)

	slog.Debug("Parsing POST arguments...")
	datetimestr := r.PostFormValue("date") + "T" + r.PostFormValue("time")
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
		slog.Debug("Detected ID in URL. Converting to int...", "idPath", idPath)
		id, err = strconv.Atoi(idPath)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}
	slog.Debug(
		"Parsed POST arguments",
		"id",
		id,
		"systolic",
		systolic,
		"diastolic",
		diastolic,
		"pulse",
		pulse,
		"irregular",
		irregular,
		"comment",
		comment,
	)

	if id == 0 {
		observation := bpo.CreateBloodPressureObservationParams{
			ObservedAt: observedAt,
			Systolic:   systolic,
			Diastolic:  diastolic,
			Pulse:      pulse,
			Irregular:  irregular,
			Comment:    comment,
			PersonID:   personID,
		}

		slog.Debug("Creating new observation", "observation", observation)
		_, err = s.queries.CreateBloodPressureObservation(r.Context(), observation)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		slog.Debug("Created new observation", "observation", observation)
	} else {
		observation := bpo.UpdateBloodPressureObservationParams{
			ObservedAt: observedAt,
			ID:         id,
			Systolic:   systolic,
			Diastolic:  diastolic,
			Pulse:      pulse,
			Irregular:  irregular,
			Comment:    comment,
			PersonID:   personID,
		}

		slog.Debug("Updating observation", "observation", observation)
		_, err = s.queries.UpdateBloodPressureObservation(r.Context(), observation)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		slog.Debug("Updated observation", "observation", observation)
	}

	http.Redirect(w, r, "/", 303)
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func logMid(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Info("Received request", "method", r.Method, "url", r.URL)
		next.ServeHTTP(w, r)
	})
}

func main() {
	ctx := context.Background()

	logLevel := slog.LevelInfo
	if getEnv("ENVIRONMENT", "development") == "development" {
		logLevel = slog.LevelDebug
	}
	logger := slog.New(
		slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: logLevel,
		}),
	)
	slog.SetDefault(logger)

	slog.Info("Starting program...")

	slog.Info("Connecting to database...")
	connStr := fmt.Sprintf(
		"host=%s port=5432 user=%s password=%s dbname=%s sslmode=disable",
		getEnv("POSTGRES_HOST", "localhost"),
		getEnv("POSTGRES_USER", "gbpw"),
		getEnv("POSTGRES_PASSWORD", "gbpwassword"),
		getEnv("POSTGRES_DB", "gbpw"),
	)
	db, err := pgx.Connect(ctx, connStr)
	if err != nil {
		slog.Error(err.Error())
		panic(err)
	}
	defer db.Close(ctx)
	db.Ping(ctx)
	queries := bpo.New(db)
	dbConn := &DbConn{queries: queries}

	slog.Info("Attaching HTTP handlers...")
	router := http.NewServeMux()
	router.Handle("GET /auth/google/login", http.HandlerFunc(oauthGoogleLogin))
	router.Handle("GET /auth/google/callback", http.HandlerFunc(dbConn.oauthGoogleCallback))
	router.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	router.Handle("GET /export", dbConn.authMid(http.HandlerFunc(dbConn.exportHandler)))
	router.Handle("GET /", dbConn.authMid(http.HandlerFunc(dbConn.getHandler)))
	router.Handle("POST /{id}/delete", dbConn.authMid(http.HandlerFunc(dbConn.deleteHandler)))
	router.Handle("POST /{id}/update", dbConn.authMid(http.HandlerFunc(dbConn.postHandler)))
	router.Handle("POST /", dbConn.authMid(http.HandlerFunc(dbConn.postHandler)))

	slog.Info("Starting server...")
	http.ListenAndServe(":8080", logMid(router))
}
