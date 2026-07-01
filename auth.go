package main

import (
	"bpo/sqlc/bpo"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"log/slog"
	"net/http"
	"os"
	"time"
)

var baseURL = func() string {
	if host := os.Getenv("RENDER_EXTERNAL_HOSTNAME"); host != "" {
		return "https://" + host
	}
	return "http://localhost:" + getEnv("PORT", "8080")
}()

var googleOauthConfig = &oauth2.Config{
	RedirectURL:  baseURL + "/auth/google/callback",
	ClientID:     os.Getenv("GOOGLE_OAUTH_CLIENT_ID"),
	ClientSecret: os.Getenv("GOOGLE_OAUTH_CLIENT_SECRET"),
	Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email"},
	Endpoint:     google.Endpoint,
}

func oauthGoogleLogin(w http.ResponseWriter, r *http.Request) {
	expiration := time.Now().Add(10 * time.Minute)

	b := make([]byte, 16)
	rand.Read(b)
	oauthState := base64.URLEncoding.EncodeToString(b)
	cookie := http.Cookie{
		Name:     "oauthstate",
		Value:    oauthState,
		Expires:  expiration,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}

	// workaround for no TLS on dev for now
	if getEnv("ENVIRONMENT", "development") == "development" {
		cookie.Secure = false
	}

	http.SetCookie(w, &cookie)
	url := googleOauthConfig.AuthCodeURL(oauthState)
	slog.Info("Redirecting to ", "url", url)

	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (s *App) oauthGoogleCallback(w http.ResponseWriter, r *http.Request) {
	oauthState, err := r.Cookie("oauthstate")
	if err != nil {
		slog.Warn("oauthstate cookie missing")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if r.FormValue("state") != oauthState.Value {
		slog.Warn("invalid oauth google state")
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	token, err := googleOauthConfig.Exchange(r.Context(), r.FormValue("code"))
	if err != nil {
		http.Error(w, "failed code exchange", http.StatusInternalServerError)
		return
	}

	client := googleOauthConfig.Client(r.Context(), token)
	response, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		http.Error(w, "failed getting user info", http.StatusInternalServerError)
		return
	}
	defer response.Body.Close()

	var userInfo struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(response.Body).Decode(&userInfo); err != nil {
		http.Error(w, "failed read response", http.StatusInternalServerError)
		return
	}

	// there's theoretically a race condition here
	// if there are two concurrent requests
	// but, that's not going to happen in this app
	// and even if it does happen, all that goes wrong is one weird http error
	// and on one refresh everything OK again
	person, err := s.queries.GetPersonByEmail(r.Context(), userInfo.Email)
	if errors.Is(err, pgx.ErrNoRows) {
		person, err = s.queries.CreatePerson(r.Context(), bpo.CreatePersonParams{
			Email:     userInfo.Email,
			CreatedAt: time.Now(),
		})
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	expiresAt := time.Now().Add(24 * time.Hour)
	sessionId := uuid.New()
	createSessionParameters := bpo.CreateSessionParams{
		ID:        sessionId,
		ExpiresAt: expiresAt,
		PersonID:  person.ID,
	}

	_, err = s.queries.CreateSession(r.Context(), createSessionParameters)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	cookie :=
		http.Cookie{
			Name:     "bpo-session",
			Value:    sessionId.String(),
			Path:     "/",
			Expires:  expiresAt,
			Secure:   true,
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
		}
	// workaround for no TLS on dev for now
	if getEnv("ENVIRONMENT", "development") == "development" {
		cookie.Secure = false
	}
	http.SetCookie(w, &cookie)

	slog.Info("Successfully logged in", "sessionId", sessionId)
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

func (s *App) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("bpo-session")
		slog.Debug("Read session cookie", "cookie", cookie)
		if err != nil || cookie.Value == "" {
			http.Redirect(w, r, "/auth/google/login", http.StatusTemporaryRedirect)
			return
		}

		sessionId, err := uuid.Parse(cookie.Value)
		if err != nil {
			http.Redirect(w, r, "/auth/google/login", http.StatusTemporaryRedirect)
			return
		}

		session, err := s.queries.GetSessionById(r.Context(), sessionId)
		slog.Debug("Read session", "session", session)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, pgx.ErrTooManyRows) {
				http.Redirect(w, r, "/auth/google/login", http.StatusTemporaryRedirect)
				return
			} else {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		slog.Debug("Checking session authentication", "ExpiresAt", session.ExpiresAt)
		if session.ExpiresAt.Before(time.Now()) {
			http.Redirect(w, r, "/auth/google/login", http.StatusTemporaryRedirect)
			return
		}

		ctx := context.WithValue(r.Context(), "PersonID", session.PersonID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
