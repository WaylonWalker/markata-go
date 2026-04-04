package serveadmin

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrNotAuthenticated = errors.New("not authenticated")
	ErrInvalidSession   = errors.New("invalid session")
	ErrInvalidPassword  = errors.New("invalid password")
)

const (
	sessionCookieName = "markata_session"
	sessionExpiry     = 24 * time.Hour
)

type session struct {
	UserID string
	Expiry time.Time
}

func withAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sess, err := getSession(r)
		if err != nil {
			http.Redirect(w, r, "/__admin/login", http.StatusFound)
			return
		}

		ctx := context.WithValue(r.Context(), "session", sess)
		next(w, r.WithContext(ctx))
	}
}

func getSession(r *http.Request) (*session, error) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return nil, ErrNotAuthenticated
	}

	// Decode and verify session
	data, err := base64.URLEncoding.DecodeString(cookie.Value)
	if err != nil {
		return nil, ErrInvalidSession
	}

	// Session format: expiry_timestamp|user_id|signature
	parts := strings.SplitN(string(data), "|", 3)
	if len(parts) != 3 {
		return nil, ErrInvalidSession
	}

	// Parse RFC3339 timestamp
	expiry, err := time.Parse(time.RFC3339, parts[0])
	if err != nil {
		return nil, ErrInvalidSession
	}

	if time.Now().After(expiry) {
		return nil, ErrNotAuthenticated
	}

	// Verify HMAC signature
	expectedSig := signSession(parts[0], parts[1])
	if !hmac.Equal([]byte(parts[2]), []byte(expectedSig)) {
		return nil, ErrInvalidSession
	}

	return &session{
		UserID: parts[1],
		Expiry: expiry,
	}, nil
}

func createSession(userID string) (string, error) {
	expiry := time.Now().Add(sessionExpiry)
	expiryStr := expiry.Format(time.RFC3339)

	sig := signSession(expiryStr, userID)
	sessionData := expiryStr + "|" + userID + "|" + sig

	return base64.URLEncoding.EncodeToString([]byte(sessionData)), nil
}

func signSession(expiry, userID string) string {
	secrets, _ := LoadSecrets(GetSecretsDir())
	key := "default-session-key-change-in-production"
	if secrets != nil && secrets.SessionKey != "" {
		key = secrets.SessionKey
	}
	h := hmac.New(sha256.New, []byte(key))
	h.Write([]byte(expiry + "|" + userID))
	return base64.URLEncoding.EncodeToString(h.Sum(nil))
}

// HashPassword creates a bcrypt hash of the password
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPassword verifies a password against a hash
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func clearSession(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:   sessionCookieName,
		Value:  "",
		MaxAge: -1,
		Path:   "/",
	})
}

func setSession(w http.ResponseWriter, userID string) error {
	sessionVal, err := createSession(userID)
	if err != nil {
		return err
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    sessionVal,
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // TODO: true in production
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(sessionExpiry),
	})

	return nil
}

// generateCSRF generates a CSRF token for the session
func generateCSRF() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// CSRF middleware for mutating requests
func csrfMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			next(w, r)
			return
		}

		// Check CSRF token
		token := r.Header.Get("X-CSRF-Token")
		if token == "" {
			http.Error(w, "CSRF token required", http.StatusBadRequest)
			return
		}

		// TODO: validate token against session
		next(w, r)
	}
}
