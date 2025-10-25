package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/alexedwards/scs/v2"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	applog "perfugo/internal/log"
	"perfugo/models"
)

const (
	sessionAuthenticatedKey = "auth:authenticated"
	sessionLoginMessageKey  = "auth:message"
	sessionUserIDKey        = "auth:user:id"
	sessionUserEmailKey     = "auth:user:email"
	sessionUserNameKey      = "auth:user:name"
)

var (
	sessionManager *scs.SessionManager
	database       *gorm.DB
)

// Configure installs the shared dependencies used by the HTTP handlers.
func Configure(sm *scs.SessionManager, db *gorm.DB) {
	sessionManager = sm
	database = db
}

func createUser(r *http.Request, email, name, password string) (*models.User, error) {
	if database == nil {
		return nil, gorm.ErrInvalidDB
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		Email:        strings.ToLower(email),
		Name:         strings.TrimSpace(name),
		PasswordHash: string(hashed),
	}

	if err := database.WithContext(r.Context()).Create(user).Error; err != nil {
		return nil, err
	}

	return user, nil
}

func findUserByEmail(r *http.Request, email string) (*models.User, error) {
	if database == nil {
		return nil, gorm.ErrInvalidDB
	}

	user := &models.User{}
	err := database.WithContext(r.Context()).Where("lower(email) = ?", strings.ToLower(email)).First(user).Error
	if err != nil {
		return nil, err
	}
	return user, nil
}

// authenticate verifies the provided credentials and populates the session if successful.
func authenticate(w http.ResponseWriter, r *http.Request, email, password string) bool {
	if sessionManager == nil {
		http.Error(w, "authentication not available", http.StatusServiceUnavailable)
		return false
	}

	user, err := findUserByEmail(r, email)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			sessionManager.Put(r.Context(), sessionLoginMessageKey, "Invalid email or password. Please try again.")
		} else {
			applog.Error(r.Context(), "failed to load user during login", "error", err)
			sessionManager.Put(r.Context(), sessionLoginMessageKey, "We were unable to sign you in. Please try again.")
		}
		return false
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		sessionManager.Put(r.Context(), sessionLoginMessageKey, "Invalid email or password. Please try again.")
		return false
	}

	if err := establishSession(r, user); err != nil {
		applog.Error(r.Context(), "failed to establish session", "error", err)
		sessionManager.Put(r.Context(), sessionLoginMessageKey, "We were unable to sign you in. Please try again.")
		return false
	}

	return true
}

func establishSession(r *http.Request, user *models.User) error {
	if sessionManager == nil {
		return errors.New("session manager not configured")
	}
	if err := sessionManager.RenewToken(r.Context()); err != nil {
		return err
	}
	sessionManager.Put(r.Context(), sessionAuthenticatedKey, true)
	sessionManager.Put(r.Context(), sessionUserIDKey, int(user.ID))
	sessionManager.Put(r.Context(), sessionUserEmailKey, user.Email)
	sessionManager.Put(r.Context(), sessionUserNameKey, user.Name)
	return nil
}

// RequireAuthentication ensures the user has an active session before accessing the resource.
func RequireAuthentication(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !ActiveSession(r) {
			redirectToLogin(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Logout destroys the current session and redirects the user to the login screen.
func Logout(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet, http.MethodPost:
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if sessionManager != nil {
		if err := sessionManager.Destroy(r.Context()); err != nil {
			applog.Error(r.Context(), "failed to destroy session", "error", err)
		}
	}

	redirectToLogin(w, r)
}

func redirectToLogin(w http.ResponseWriter, r *http.Request) {
	if isHTMX(r) {
		w.Header().Set("HX-Redirect", "/login")
		w.WriteHeader(http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func redirectToApp(w http.ResponseWriter, r *http.Request) {
	if isHTMX(r) {
		w.Header().Set("HX-Redirect", "/app")
		w.WriteHeader(http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/app", http.StatusSeeOther)
}

// ActiveSession returns true when the current request has an authenticated session.
func ActiveSession(r *http.Request) bool {
	if sessionManager == nil {
		return false
	}
	return sessionManager.GetBool(r.Context(), sessionAuthenticatedKey) && sessionManager.GetInt(r.Context(), sessionUserIDKey) > 0
}
