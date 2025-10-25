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
	sessionUserThemeKey     = "auth:user:theme"
)

var (
	sessionManager *scs.SessionManager
	database       *gorm.DB
)

const defaultUserTheme = "nocturne"

// Configure installs the shared dependencies used by the HTTP handlers.
func Configure(sm *scs.SessionManager, db *gorm.DB) {
	applog.Debug(nil, "configuring handler dependencies", "hasSession", sm != nil, "hasDatabase", db != nil)
	sessionManager = sm
	database = db
	applog.Debug(nil, "handler dependencies configured")
}

func createUser(r *http.Request, email, name, password string) (*models.User, error) {
	if database == nil {
		return nil, gorm.ErrInvalidDB
	}

	applog.Debug(r.Context(), "creating user", "email", strings.ToLower(email))

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		applog.Error(r.Context(), "failed to hash password", "error", err)
		return nil, err
	}

	user := &models.User{
		Email:        strings.ToLower(email),
		Name:         strings.TrimSpace(name),
		PasswordHash: string(hashed),
		Theme:        defaultUserTheme,
	}

	if err := database.WithContext(r.Context()).Create(user).Error; err != nil {
		applog.Error(r.Context(), "failed to persist user", "error", err)
		return nil, err
	}

	applog.Debug(r.Context(), "user created", "userID", user.ID, "email", user.Email)
	return user, nil
}

func findUserByEmail(r *http.Request, email string) (*models.User, error) {
	if database == nil {
		return nil, gorm.ErrInvalidDB
	}

	applog.Debug(r.Context(), "looking up user by email", "email", strings.ToLower(email))
	user := &models.User{}
	err := database.WithContext(r.Context()).Where("lower(email) = ?", strings.ToLower(email)).First(user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			applog.Debug(r.Context(), "user not found", "email", strings.ToLower(email))
		} else {
			applog.Error(r.Context(), "failed to query user", "error", err)
		}
		return nil, err
	}
	applog.Debug(r.Context(), "user located", "userID", user.ID, "email", user.Email)
	return user, nil
}

// authenticate verifies the provided credentials and populates the session if successful.
func authenticate(w http.ResponseWriter, r *http.Request, email, password string) bool {
	if sessionManager == nil {
		applog.Debug(r.Context(), "session manager unavailable during authentication")
		http.Error(w, "authentication not available", http.StatusServiceUnavailable)
		return false
	}

	applog.Debug(r.Context(), "beginning authentication", "email", strings.ToLower(email))
	user, err := findUserByEmail(r, email)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			applog.Debug(r.Context(), "authentication failed: user not found", "email", strings.ToLower(email))
			sessionManager.Put(r.Context(), sessionLoginMessageKey, "Invalid email or password. Please try again.")
		} else {
			applog.Error(r.Context(), "failed to load user during login", "error", err)
			sessionManager.Put(r.Context(), sessionLoginMessageKey, "We were unable to sign you in. Please try again.")
		}
		return false
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		applog.Debug(r.Context(), "authentication failed: invalid password", "userID", user.ID)
		sessionManager.Put(r.Context(), sessionLoginMessageKey, "Invalid email or password. Please try again.")
		return false
	}

	if err := establishSession(r, user); err != nil {
		applog.Error(r.Context(), "failed to establish session", "error", err)
		sessionManager.Put(r.Context(), sessionLoginMessageKey, "We were unable to sign you in. Please try again.")
		return false
	}

	applog.Debug(r.Context(), "authentication complete", "userID", user.ID)
	return true
}

func establishSession(r *http.Request, user *models.User) error {
	if sessionManager == nil {
		applog.Debug(r.Context(), "cannot establish session: session manager missing")
		return errors.New("session manager not configured")
	}
	applog.Debug(r.Context(), "renewing session token", "userID", user.ID)
	if err := sessionManager.RenewToken(r.Context()); err != nil {
		applog.Error(r.Context(), "failed to renew session token", "error", err)
		return err
	}
	applog.Debug(r.Context(), "populating session", "userID", user.ID)
	sessionManager.Put(r.Context(), sessionAuthenticatedKey, true)
	sessionManager.Put(r.Context(), sessionUserIDKey, int(user.ID))
	sessionManager.Put(r.Context(), sessionUserEmailKey, user.Email)
	sessionManager.Put(r.Context(), sessionUserNameKey, user.Name)
	theme := strings.TrimSpace(user.Theme)
	if theme == "" {
		theme = defaultUserTheme
	}
	sessionManager.Put(r.Context(), sessionUserThemeKey, theme)
	applog.Debug(r.Context(), "session established", "userID", user.ID)
	return nil
}

// RequireAuthentication ensures the user has an active session before accessing the resource.
func RequireAuthentication(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !ActiveSession(r) {
			applog.Debug(r.Context(), "request missing active session, redirecting to login", "path", r.URL.Path)
			redirectToLogin(w, r)
			return
		}
		applog.Debug(r.Context(), "authenticated request proceeding", "path", r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

// Logout destroys the current session and redirects the user to the login screen.
func Logout(w http.ResponseWriter, r *http.Request) {
	applog.Debug(r.Context(), "handling logout request", "method", r.Method)
	switch r.Method {
	case http.MethodGet, http.MethodPost:
	default:
		applog.Debug(r.Context(), "logout method not allowed", "method", r.Method)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if sessionManager != nil {
		if err := sessionManager.Destroy(r.Context()); err != nil {
			applog.Error(r.Context(), "failed to destroy session", "error", err)
		} else {
			applog.Debug(r.Context(), "session destroyed successfully")
		}
	}

	redirectToLogin(w, r)
}

func redirectToLogin(w http.ResponseWriter, r *http.Request) {
	applog.Debug(r.Context(), "redirecting to login", "htmx", isHTMX(r))
	if isHTMX(r) {
		w.Header().Set("HX-Redirect", "/login")
		w.WriteHeader(http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func redirectToApp(w http.ResponseWriter, r *http.Request) {
	applog.Debug(r.Context(), "redirecting to app", "htmx", isHTMX(r))
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

func currentUserID(r *http.Request) uint {
	if sessionManager == nil {
		return 0
	}
	id := sessionManager.GetInt(r.Context(), sessionUserIDKey)
	if id <= 0 {
		return 0
	}
	return uint(id)
}

func currentUserTheme(r *http.Request) string {
	if sessionManager == nil {
		return defaultUserTheme
	}
	theme := strings.TrimSpace(sessionManager.GetString(r.Context(), sessionUserThemeKey))
	if theme == "" {
		return defaultUserTheme
	}
	return theme
}

func loadCurrentUser(r *http.Request) (*models.User, error) {
	if database == nil {
		return nil, gorm.ErrInvalidDB
	}
	id := currentUserID(r)
	if id == 0 {
		return nil, errors.New("no authenticated user in session")
	}
	user := &models.User{}
	if err := database.WithContext(r.Context()).First(user, id).Error; err != nil {
		return nil, err
	}
	return user, nil
}

func setSessionTheme(r *http.Request, theme string) {
	if sessionManager == nil {
		return
	}
	sessionManager.Put(r.Context(), sessionUserThemeKey, strings.TrimSpace(theme))
}
