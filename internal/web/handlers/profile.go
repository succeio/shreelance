package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/gorilla/csrf"
	"gorm.io/gorm"

	"shreelance/internal/ui"
)

type ProfileHandler struct {
	DB      *gorm.DB
	Session *scs.SessionManager
}

func NewProfileHandler(db *gorm.DB, session *scs.SessionManager) *ProfileHandler {
	return &ProfileHandler{
		DB:      db,
		Session: session,
	}
}

func (h *ProfileHandler) Show(w http.ResponseWriter, r *http.Request) {
	user, role := GetUserFromSession(h.DB, h.Session, r)
	if user == nil {
		http.Redirect(w, r, "/auth/github", http.StatusSeeOther)
		return
	}

	csrfToken := csrf.Token(r)
	content := ui.ProfilePage(user, role, csrfToken)
	layout := ui.Layout(ui.PageParams{
		Title:       "Профиль",
		Content:     content,
		User:        user,
		CSRFToken:   csrfToken,
		ContextRole: role,
		Theme:       GetThemeFromCookie(r),
	})

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = layout.Render(w)
}

func (h *ProfileHandler) SwitchRole(w http.ResponseWriter, r *http.Request) {
	user, _ := GetUserFromSession(h.DB, h.Session, r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	role := r.FormValue("role")
	if role != "customer" && role != "freelancer" {
		http.Error(w, "Invalid role", http.StatusBadRequest)
		return
	}

	h.Session.Put(r.Context(), "role", role)
	http.Redirect(w, r, r.Referer(), http.StatusSeeOther)
}

func (h *ProfileHandler) Update(w http.ResponseWriter, r *http.Request) {
	user, _ := GetUserFromSession(h.DB, h.Session, r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	stack := r.FormValue("stack")
	expYearsStr := r.FormValue("experience_years")
	expYears, _ := strconv.Atoi(expYearsStr)

	user.Stack = stack
	user.ExperienceYears = expYears
	user.UpdatedAt = time.Now()

	if err := h.DB.Save(user).Error; err != nil {
		http.Error(w, "Failed to save profile: "+err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/profile", http.StatusSeeOther)
}

func (h *ProfileHandler) SyncGitHub(w http.ResponseWriter, r *http.Request) {
	user, _ := GetUserFromSession(h.DB, h.Session, r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if user.GitHubToken == "" {
		http.Error(w, "GitHub token not found. Please log in again.", http.StatusBadRequest)
		return
	}

	if err := SyncGitHubData(h.DB, user, user.GitHubToken); err != nil {
		http.Error(w, "Failed to sync with GitHub: "+err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/profile", http.StatusSeeOther)
}
