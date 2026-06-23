package main

import (
	"bpo/sqlc/bpo" // generated code via sqlc
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"
)

type Server struct {
	queries *bpo.Queries
}

type Session struct {
	id              string
	authenticatedAt time.Time
}

type TemplateContext struct {
	Bpos     []bpo.BloodPressureObservation
	Editable bool
}

var templates = template.Must(template.ParseFiles(
	"bpos.html",
	"bpo-row.html",
	"bpo-row-editable.html",
))

var googleOauthConfig = &oauth2.Config{
	RedirectURL:  "http://localhost:8080/auth/google/callback",
	ClientID:     os.Getenv("GOOGLE_OAUTH_CLIENT_ID"),
	ClientSecret: os.Getenv("GOOGLE_OAUTH_CLIENT_SECRET"),
	Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email"},
	Endpoint:     google.Endpoint,
}

var sessions = make(map[string]Session)

func (s *Server) getHandler(w http.ResponseWriter, r *http.Request) {
	slog.Debug("Executing getHandler()")

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
	slog.Debug("Executing deleteHandler()")

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
			ObservedAt: pgtype.Timestamp{Time: observedAt, Valid: true},
			Systolic:   int32(systolic),
			Diastolic:  int32(diastolic),
			Pulse:      int32(pulse),
			Irregular:  irregular,
			Comment:    comment,
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
			ObservedAt: pgtype.Timestamp{Time: observedAt, Valid: true},
			ID:         int32(id),
			Systolic:   int32(systolic),
			Diastolic:  int32(diastolic),
			Pulse:      int32(pulse),
			Irregular:  irregular,
			Comment:    comment,
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

func (s *Server) oauthGoogleCallback(w http.ResponseWriter, r *http.Request) {
	oauthState, _ := r.Cookie("oauthstate")

	if r.FormValue("state") != oauthState.Value {
		slog.Info("invalid oauth google state")
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	token, err := googleOauthConfig.Exchange(context.Background(), r.FormValue("code"))
	if err != nil {
		http.Error(w, "failed code exchange", http.StatusInternalServerError)
		return
	}

	response, err := http.Get("https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + token.AccessToken)
	if err != nil {
		http.Error(w, "failed getting user info", http.StatusInternalServerError)
	}
	defer response.Body.Close()

	var userInfo struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(response.Body).Decode(&userInfo); err != nil {
		http.Error(w, "failed read response", http.StatusInternalServerError)
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	sessionId := uuid.NewString()
	session := Session{
		id:              sessionId,
		authenticatedAt: time.Now().UTC(),
	}
	sessions[sessionId] = session

	cookie :=
		http.Cookie{
			Name:     "bpo-session",
			Value:    sessionId,
			Path:     "/",
			Expires:  time.Now().Add(24 * time.Hour),
			Secure:   false,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		}
	http.SetCookie(w, &cookie)
	slog.Info("Successfully logged in", "email", userInfo.Email)
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

func (s *Server) oauthGoogleLogin(w http.ResponseWriter, r *http.Request) {
	oauthState := generateStateOauthCookie(w)
	url := googleOauthConfig.AuthCodeURL(oauthState)
	slog.Info("Redirecting to ", "url", url)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func generateStateOauthCookie(w http.ResponseWriter) string {
	var expiration = time.Now().Add(365 * 24 * time.Hour)
	b := make([]byte, 16)
	rand.Read(b)
	state := base64.URLEncoding.EncodeToString(b)
	cookie := http.Cookie{Name: "oauthstate", Value: state, Expires: expiration}
	http.SetCookie(w, &cookie)

	return state
}

func loggingMiddleWare(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Info("Received request", "method", r.Method, "url", r.URL)
		next.ServeHTTP(w, r)
	})
}

func loginMiddleWare(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("bpo-session")
		slog.Debug("Read session cookie", "cookie", cookie)
		if err != nil {
			http.Redirect(w, r, "/auth/google/login", http.StatusTemporaryRedirect)
			return
		}

		session, exists := sessions[cookie.Value]
		slog.Debug("Read session", "session", session)
		if !exists {
			http.Redirect(w, r, "/auth/google/login", http.StatusTemporaryRedirect)
			return
		}

		slog.Debug("Checking session authentication", "authenticatedAt", session.authenticatedAt)
		if session.authenticatedAt.Add(24 * time.Hour).Before(time.Now()) {
			http.Redirect(w, r, "/auth/google/login", http.StatusTemporaryRedirect)
			return
		}

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
	server := &Server{queries: queries}

	slog.Info("Attaching HTTP handlers...")
	router := http.NewServeMux()
	router.Handle("GET /auth/google/login", http.HandlerFunc(server.oauthGoogleLogin))
	router.Handle("GET /auth/google/callback", http.HandlerFunc(server.oauthGoogleCallback))
	router.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	router.Handle("GET /", loginMiddleWare(http.HandlerFunc(server.getHandler)))
	router.Handle("POST /{id}/delete", loginMiddleWare(http.HandlerFunc(server.deleteHandler)))
	router.Handle("POST /{id}/update", loginMiddleWare(http.HandlerFunc(server.postHandler)))
	router.Handle("POST /", loginMiddleWare(http.HandlerFunc(server.postHandler)))

	slog.Info("Starting server...")
	http.ListenAndServe(":8080", loggingMiddleWare(router))
}
