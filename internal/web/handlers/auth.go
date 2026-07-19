package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/alexedwards/scs/v2"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"gorm.io/gorm"

	"shreelance/internal/config"
	"shreelance/internal/models"
)

type AuthHandler struct {
	DB           *gorm.DB
	Session      *scs.SessionManager
	OAuthConfig *oauth2.Config
}

func NewAuthHandler(db *gorm.DB, session *scs.SessionManager, cfg *config.Config) *AuthHandler {
	return &AuthHandler{
		DB:      db,
		Session: session,
		OAuthConfig: &oauth2.Config{
			ClientID:     cfg.GitHubClientID,
			ClientSecret: cfg.GitHubClientSecret,
			RedirectURL:  cfg.GitHubRedirectURL,
			Scopes:       []string{"user:email"},
			Endpoint:     github.Endpoint,
		},
	}
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	// Simple state token (in a real production app, use a secure random string checked in callback)
	state := "random-state"
	url := h.OAuthConfig.AuthCodeURL(state)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (h *AuthHandler) Callback(w http.ResponseWriter, r *http.Request) {
	state := r.FormValue("state")
	if state != "random-state" {
		http.Error(w, "Invalid state parameter", http.StatusBadRequest)
		return
	}

	code := r.FormValue("code")
	token, err := h.OAuthConfig.Exchange(r.Context(), code)
	if err != nil {
		http.Error(w, "Failed to exchange token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	client := h.OAuthConfig.Client(r.Context(), token)
	resp, err := client.Get("https://api.github.com/user")
	if err != nil {
		http.Error(w, "Failed to get user info: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	var gitHubUser struct {
		ID        int64  `json:"id"`
		Login     string `json:"login"`
		Email     string `json:"email"`
		AvatarURL string `json:"avatar_url"`
		CreatedAt string `json:"created_at"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&gitHubUser); err != nil {
		http.Error(w, "Failed to decode user info: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var user models.User
	result := h.DB.Where("github_id = ?", gitHubUser.ID).First(&user)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			user = models.User{
				GitHubID:    gitHubUser.ID,
				GitHubToken: token.AccessToken,
				Username:    gitHubUser.Login,
				Email:       gitHubUser.Email,
				AvatarURL:   gitHubUser.AvatarURL,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			}
			if gitHubUser.CreatedAt != "" {
				if t, err := time.Parse(time.RFC3339, gitHubUser.CreatedAt); err == nil {
					user.GitHubCreatedAt = t
					exp := time.Now().Year() - t.Year()
					if exp < 0 {
						exp = 0
					}
					user.ExperienceYears = exp
				}
			}
			if err := h.DB.Create(&user).Error; err != nil {
				http.Error(w, "Failed to create user: "+err.Error(), http.StatusInternalServerError)
				return
			}
		} else {
			http.Error(w, "Database error: "+result.Error.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		// Update token for existing user
		user.GitHubToken = token.AccessToken
		h.DB.Save(&user)
	}

	// Try to sync GitHub data (stack/repos)
	_ = SyncGitHubData(h.DB, &user, token.AccessToken)

	h.Session.Put(r.Context(), "userID", user.ID)
	// Default context role is customer
	h.Session.Put(r.Context(), "role", "customer")

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func SyncGitHubData(db *gorm.DB, user *models.User, tokenString string) error {
	if tokenString == "" {
		return nil
	}
	req, err := http.NewRequest("GET", "https://api.github.com/user", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+tokenString)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var gitHubUser struct {
		CreatedAt string `json:"created_at"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&gitHubUser); err != nil {
		return err
	}

	if gitHubUser.CreatedAt != "" {
		if t, err := time.Parse(time.RFC3339, gitHubUser.CreatedAt); err == nil {
			user.GitHubCreatedAt = t
			exp := time.Now().Year() - t.Year()
			if exp < 0 {
				exp = 0
			}
			user.ExperienceYears = exp
		}
	}

	// Fetch repos
	reqRepos, err := http.NewRequest("GET", "https://api.github.com/user/repos?sort=updated&per_page=15", nil)
	if err != nil {
		return err
	}
	reqRepos.Header.Set("Authorization", "Bearer "+tokenString)
	respRepos, err := http.DefaultClient.Do(reqRepos)
	if err != nil {
		return err
	}
	defer respRepos.Body.Close()

	var repos []struct {
		Language string `json:"language"`
	}
	if err := json.NewDecoder(respRepos.Body).Decode(&repos); err == nil {
		langMap := make(map[string]bool)
		for _, r := range repos {
			if r.Language != "" {
				langMap[strings.ToLower(r.Language)] = true
			}
		}
		var langs []string
		for lang := range langMap {
			// Title case or just lowercase, let's title case them to look nice
			langs = append(langs, strings.Title(lang))
		}
		if len(langs) > 0 {
			user.Stack = strings.Join(langs, ", ")
		}
	}

	user.GitHubToken = tokenString
	user.UpdatedAt = time.Now()
	return db.Save(user).Error
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	h.Session.Destroy(r.Context())
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// GetUserFromSession helper
func GetUserFromSession(db *gorm.DB, session *scs.SessionManager, r *http.Request) (*models.User, string) {
	userIDVal := session.Get(r.Context(), "userID")
	if userIDVal == nil {
		return nil, ""
	}

	userID, ok := userIDVal.(uint)
	if !ok {
		// Try to parse if it's float64 from JSON encoding or int etc
		switch v := userIDVal.(type) {
		case float64:
			userID = uint(v)
		case int:
			userID = uint(v)
		default:
			return nil, ""
		}
	}

	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		return nil, ""
	}

	roleVal := session.Get(r.Context(), "role")
	role := "customer"
	if r, ok := roleVal.(string); ok && (r == "customer" || r == "freelancer") {
		role = r
	}

	return &user, role
}

// GetThemeFromCookie helper
func GetThemeFromCookie(r *http.Request) string {
	if cookie, err := r.Cookie("theme"); err == nil {
		if cookie.Value == "light" || cookie.Value == "dark" || cookie.Value == "system" {
			return cookie.Value
		}
	}
	return "system"
}
