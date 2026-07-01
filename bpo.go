package main

import (
	"bpo/sqlc/bpo" // generated code via sqlc
	"context"
	"embed"
	"encoding/csv"
	"github.com/jackc/pgx/v5/pgxpool"
	"html/template"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

type App struct {
	queries *bpo.Queries
}

type TemplateContext struct {
	Bpos     []bpo.GetBloodPressureObservationsRow
	Editable bool
}

//go:embed templates/*.html
var templateFS embed.FS
var templates = template.Must(template.ParseFS(templateFS, "templates/*.html"))

//go:embed static/*
var staticFS embed.FS

func (s *App) getHandler(w http.ResponseWriter, r *http.Request) {
	editableString := r.FormValue("editable")
	editable := false
	if editableString != "" {
		var err error
		editable, err = strconv.ParseBool(editableString)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	personID, exists := r.Context().Value("PersonID").(int)
	if !exists {
		http.Error(w, "failed to find logged-in user", http.StatusInternalServerError)
		return
	}

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

func (s *App) exportHandler(w http.ResponseWriter, r *http.Request) {
	personID, exists := r.Context().Value("PersonID").(int)
	if !exists {
		http.Error(w, "failed to find logged-in user", http.StatusInternalServerError)
		return
	}
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

func (s *App) deleteHandler(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	personID, exists := r.Context().Value("PersonID").(int)
	if !exists {
		http.Error(w, "failed to find logged-in user", http.StatusInternalServerError)
		return
	}
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

func (s *App) postHandler(w http.ResponseWriter, r *http.Request) {
	personID, exists := r.Context().Value("PersonID").(int)
	if !exists {
		http.Error(w, "failed to find logged-in user", http.StatusInternalServerError)
		return
	}

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

func logMiddleware(next http.Handler) http.Handler {
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
	sslmode := "verify-full"
	if getEnv("ENVIRONMENT", "development") == "development" {
		sslmode = "disable"
	}
	postgresHost := os.Getenv("POSTGRES_HOST")
	postgresPort := getEnv("POSTGRES_PORT", "5432")
	dbURL := &url.URL{
		Scheme:   "postgres",
		Host:     net.JoinHostPort(postgresHost, postgresPort),
		User:     url.UserPassword(os.Getenv("POSTGRES_USER"), os.Getenv("POSTGRES_PASSWORD")),
		Path:     os.Getenv("POSTGRES_DB"),
		RawQuery: "sslmode=" + sslmode,
	}
	connStr := dbURL.String()
	slog.Info("Connecting to database", "sslmode", sslmode)

	poolConfig, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		slog.Error("failed to parse db config", "err", err)
		panic(err)
	}
	poolConfig.MaxConns = 8
	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.MaxConnIdleTime = 30 * time.Minute
	poolConfig.ConnConfig.ConnectTimeout = 10 * time.Second
	poolConfig.ConnConfig.RuntimeParams["statement_timeout"] = "5000"

	db, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		slog.Error(err.Error())
		panic(err)
	}
	defer db.Close()

	if err := db.Ping(ctx); err != nil {
		slog.Error("database ping failed", "err", err)
		panic(err)
	}
	queries := bpo.New(db)
	dbConn := &App{queries: queries}

	slog.Info("Attaching HTTP handlers...")
	router := http.NewServeMux()
	router.Handle("GET /auth/google/login", http.HandlerFunc(oauthGoogleLogin))
	router.Handle("GET /auth/google/callback", http.HandlerFunc(dbConn.oauthGoogleCallback))
	router.Handle("GET /static/", http.FileServerFS(staticFS))
	router.Handle("GET /export", dbConn.authMiddleware(http.HandlerFunc(dbConn.exportHandler)))
	router.Handle("GET /", dbConn.authMiddleware(http.HandlerFunc(dbConn.getHandler)))
	router.Handle("POST /{id}/delete", dbConn.authMiddleware(http.HandlerFunc(dbConn.deleteHandler)))
	router.Handle("POST /{id}/update", dbConn.authMiddleware(http.HandlerFunc(dbConn.postHandler)))
	router.Handle("POST /", dbConn.authMiddleware(http.HandlerFunc(dbConn.postHandler)))

	slog.Info("Starting server...")

	port := os.Getenv("PORT")
	handler := logMiddleware(http.NewCrossOriginProtection().Handler(router))
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		slog.Error("server error", "err", err)
		panic(err)
	}
}
