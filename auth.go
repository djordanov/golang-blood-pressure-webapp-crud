package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"log/slog"
	"net/http"
	"os"
	"time"
)

type Session struct {
	id              string
	authenticatedAt time.Time
	email           string
}

var sessions = make(map[string]Session)

var googleOauthConfig = &oauth2.Config{
	RedirectURL:  "http://localhost:8080/auth/google/callback",
	ClientID:     os.Getenv("GOOGLE_OAUTH_CLIENT_ID"),
	ClientSecret: os.Getenv("GOOGLE_OAUTH_CLIENT_SECRET"),
	Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email"},
	Endpoint:     google.Endpoint,
}

func oauthGoogleLogin(w http.ResponseWriter, r *http.Request) {
	oauthState := generateStateOauthCookie(w)
	url := googleOauthConfig.AuthCodeURL(oauthState)
	slog.Info("Redirecting to ", "url", url)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func generateStateOauthCookie(w http.ResponseWriter) string {
	var expiration = time.Now().Add(10 * time.Minute)
	b := make([]byte, 16)
	rand.Read(b)
	state := base64.URLEncoding.EncodeToString(b)
	cookie := http.Cookie{
		Name:     "oauthstate",
		Value:    state,
		Expires:  expiration,
		Secure:   false,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, &cookie)

	return state
}

func oauthGoogleCallback(w http.ResponseWriter, r *http.Request) {
	oauthState, err := r.Cookie("oauthstate")
	if err != nil {
		slog.Warn("oauthstate cookie missing")
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	if r.FormValue("state") != oauthState.Value {
		slog.Info("invalid oauth google state")
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

	sessionId := uuid.NewString()
	session := Session{
		id:              sessionId,
		authenticatedAt: time.Now().UTC(),
		email:           userInfo.Email,
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
