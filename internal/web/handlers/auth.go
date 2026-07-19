package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/gitlab"
	"gorm.io/gorm"

	"shreelance/internal/config"
	"shreelance/internal/models"
)

type AuthHandler struct {
	DB               *gorm.DB
	Session          *scs.SessionManager
	GitHubOAuthConfig *oauth2.Config
	GitLabOAuthConfig *oauth2.Config
}

func NewAuthHandler(db *gorm.DB, session *scs.SessionManager, cfg *config.Config) *AuthHandler {
	return &AuthHandler{
		DB:      db,
		Session: session,
		GitHubOAuthConfig: &oauth2.Config{
			ClientID:     cfg.GitHubClientID,
			ClientSecret: cfg.GitHubClientSecret,
			RedirectURL:  cfg.GitHubRedirectURL,
			Scopes:       []string{"user:email"},
			Endpoint:     github.Endpoint,
		},
		GitLabOAuthConfig: &oauth2.Config{
			ClientID:     cfg.GitLabClientID,
			ClientSecret: cfg.GitLabClientSecret,
			RedirectURL:  cfg.GitLabRedirectURL,
			Scopes:       []string{"read_user", "read_repository"},
			Endpoint:     gitlab.Endpoint,
		},
	}
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	// Simple state token (in a real production app, use a secure random string checked in callback)
	state := "random-state"
	url := h.GitHubOAuthConfig.AuthCodeURL(state)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (h *AuthHandler) Callback(w http.ResponseWriter, r *http.Request) {
	state := r.FormValue("state")
	if state != "random-state" {
		http.Error(w, "Invalid state parameter", http.StatusBadRequest)
		return
	}

	code := r.FormValue("code")
	token, err := h.GitHubOAuthConfig.Exchange(r.Context(), code)
	if err != nil {
		http.Error(w, "Failed to exchange token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	client := h.GitHubOAuthConfig.Client(r.Context(), token)
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
				GitHubID:    &gitHubUser.ID,
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

// GitLab Login & Callback
func (h *AuthHandler) GitLabLogin(w http.ResponseWriter, r *http.Request) {
	state := "random-gitlab-state"
	url := h.GitLabOAuthConfig.AuthCodeURL(state)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (h *AuthHandler) GitLabCallback(w http.ResponseWriter, r *http.Request) {
	state := r.FormValue("state")
	if state != "random-gitlab-state" {
		http.Error(w, "Invalid state parameter", http.StatusBadRequest)
		return
	}

	code := r.FormValue("code")
	token, err := h.GitLabOAuthConfig.Exchange(r.Context(), code)
	if err != nil {
		http.Error(w, "Failed to exchange token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	client := h.GitLabOAuthConfig.Client(r.Context(), token)
	resp, err := client.Get("https://gitlab.com/api/v4/user")
	if err != nil {
		http.Error(w, "Failed to get user info: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	var gitLabUser struct {
		ID        int64  `json:"id"`
		Username  string `json:"username"`
		Email     string `json:"email"`
		AvatarURL string `json:"avatar_url"`
		CreatedAt string `json:"created_at"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&gitLabUser); err != nil {
		http.Error(w, "Failed to decode user info: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Check if logged in user is linking GitLab
	loggedInUser, _ := GetUserFromSession(h.DB, h.Session, r)

	var user models.User
	if loggedInUser != nil {
		user = *loggedInUser
		user.GitLabID = &gitLabUser.ID
		user.GitLabToken = token.AccessToken
		user.GitLabUsername = gitLabUser.Username
		if gitLabUser.CreatedAt != "" {
			if t, err := time.Parse(time.RFC3339, gitLabUser.CreatedAt); err == nil {
				user.GitLabCreatedAt = t
			}
		}
		h.DB.Save(&user)
	} else {
		result := h.DB.Where("gitlab_id = ?", gitLabUser.ID).First(&user)
		if result.Error != nil {
			if result.Error == gorm.ErrRecordNotFound {
				// Also check if email matches existing account
				if gitLabUser.Email != "" {
					var existingByEmail models.User
					if err := h.DB.Where("email = ?", gitLabUser.Email).First(&existingByEmail).Error; err == nil {
						user = existingByEmail
						user.GitLabID = &gitLabUser.ID
						user.GitLabToken = token.AccessToken
						user.GitLabUsername = gitLabUser.Username
						if gitLabUser.CreatedAt != "" {
							if t, err := time.Parse(time.RFC3339, gitLabUser.CreatedAt); err == nil {
								user.GitLabCreatedAt = t
							}
						}
						h.DB.Save(&user)
					} else {
						user = models.User{
							GitLabID:       &gitLabUser.ID,
							GitLabToken:    token.AccessToken,
							GitLabUsername: gitLabUser.Username,
							Username:       gitLabUser.Username,
							Email:          gitLabUser.Email,
							AvatarURL:      gitLabUser.AvatarURL,
							CreatedAt:      time.Now(),
							UpdatedAt:      time.Now(),
						}
						if gitLabUser.CreatedAt != "" {
							if t, err := time.Parse(time.RFC3339, gitLabUser.CreatedAt); err == nil {
								user.GitLabCreatedAt = t
							}
						}
						if err := h.DB.Create(&user).Error; err != nil {
							http.Error(w, "Failed to create user: "+err.Error(), http.StatusInternalServerError)
							return
						}
					}
				} else {
					user = models.User{
						GitLabID:       &gitLabUser.ID,
						GitLabToken:    token.AccessToken,
						GitLabUsername: gitLabUser.Username,
						Username:       gitLabUser.Username,
						Email:          gitLabUser.Email,
						AvatarURL:      gitLabUser.AvatarURL,
						CreatedAt:      time.Now(),
						UpdatedAt:      time.Now(),
					}
					if gitLabUser.CreatedAt != "" {
						if t, err := time.Parse(time.RFC3339, gitLabUser.CreatedAt); err == nil {
							user.GitLabCreatedAt = t
						}
					}
					if err := h.DB.Create(&user).Error; err != nil {
						http.Error(w, "Failed to create user: "+err.Error(), http.StatusInternalServerError)
						return
					}
				}
			} else {
				http.Error(w, "Database error: "+result.Error.Error(), http.StatusInternalServerError)
				return
			}
		} else {
			user.GitLabToken = token.AccessToken
			user.GitLabUsername = gitLabUser.Username
			h.DB.Save(&user)
		}
	}

	_ = SyncGitLabData(h.DB, &user, token.AccessToken)

	h.Session.Put(r.Context(), "userID", user.ID)
	h.Session.Put(r.Context(), "role", "customer")

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func SyncGitLabData(db *gorm.DB, user *models.User, tokenString string) error {
	if tokenString == "" {
		return nil
	}

	// Try using user-specific projects URL first, fall back to membership/owned projects API
	reqURL := fmt.Sprintf("https://gitlab.com/api/v4/users/%d/projects?visibility=public&order_by=updated_at&per_page=50", *user.GitLabID)
	if user.GitLabID == nil {
		reqURL = "https://gitlab.com/api/v4/projects?membership=true&visibility=public&order_by=updated_at&per_page=50"
	}
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+tokenString)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// If we get an error or empty from users projects (some GitLab configurations hide user profiles), fallback to membership
	var projects []struct {
		ID         int64  `json:"id"`
		Visibility string `json:"visibility"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&projects); err != nil || len(projects) == 0 {
		fallbackURL := "https://gitlab.com/api/v4/projects?membership=true&visibility=public&order_by=updated_at&per_page=50"
		reqFB, err := http.NewRequest("GET", fallbackURL, nil)
		if err == nil {
			reqFB.Header.Set("Authorization", "Bearer "+tokenString)
			respFB, err := http.DefaultClient.Do(reqFB)
			if err == nil {
				defer respFB.Body.Close()
				var fbProjects []struct {
					ID         int64  `json:"id"`
					Visibility string `json:"visibility"`
				}
				if json.NewDecoder(respFB.Body).Decode(&fbProjects) == nil {
					projects = fbProjects
				}
			}
		}
	}

	langMap := make(map[string]bool)

	// Retain existing stack languages
	if user.Stack != "" {
		for _, s := range strings.Split(user.Stack, ",") {
			t := strings.TrimSpace(s)
			if t != "" {
				langMap[strings.ToLower(t)] = true
			}
		}
	}

	// For the top 10 public projects, fetch their programming languages
	limit := len(projects)
	if limit > 10 {
		limit = 10
	}
	for i := 0; i < limit; i++ {
		p := projects[i]
		if p.Visibility == "public" {
			langURL := fmt.Sprintf("https://gitlab.com/api/v4/projects/%d/languages", p.ID)
			lReq, err := http.NewRequest("GET", langURL, nil)
			if err != nil {
				continue
			}
			lReq.Header.Set("Authorization", "Bearer "+tokenString)
			lResp, err := http.DefaultClient.Do(lReq)
			if err != nil {
				continue
			}
			var projLangs map[string]float64
			if json.NewDecoder(lResp.Body).Decode(&projLangs) == nil {
				for lang := range projLangs {
					langMap[strings.ToLower(lang)] = true
				}
			}
			lResp.Body.Close()
		}
	}

	var langs []string
	for lang := range langMap {
		langs = append(langs, strings.Title(lang))
	}
	if len(langs) > 0 {
		user.Stack = strings.Join(langs, ", ")
	}

	user.UpdatedAt = time.Now()
	return db.Save(user).Error
}

// Email Registration and Login
func (h *AuthHandler) RegisterEmail(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	username := strings.TrimSpace(r.FormValue("username"))
	email := strings.TrimSpace(strings.ToLower(r.FormValue("email")))
	password := r.FormValue("password")

	if username == "" || email == "" || password == "" {
		http.Redirect(w, r, "/register?error="+url.QueryEscape("Заполните все поля"), http.StatusSeeOther)
		return
	}

	if len(password) < 6 {
		http.Redirect(w, r, "/register?error="+url.QueryEscape("Пароль должен содержать минимум 6 символов"), http.StatusSeeOther)
		return
	}

	var existing models.User
	if err := h.DB.Where("email = ?", email).First(&existing).Error; err == nil {
		http.Redirect(w, r, "/register?error="+url.QueryEscape("Пользователь с таким email уже зарегистрирован"), http.StatusSeeOther)
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Failed to process password", http.StatusInternalServerError)
		return
	}

	// Avatar placeholder using Gravatar/Dicebear
	avatarURL := fmt.Sprintf("https://api.dicebear.com/7.x/identicon/svg?seed=%s", url.QueryEscape(username))

	user := models.User{
		Username:     username,
		Email:        email,
		PasswordHash: string(hashedPassword),
		AvatarURL:    avatarURL,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := h.DB.Create(&user).Error; err != nil {
		http.Redirect(w, r, "/register?error="+url.QueryEscape("Ошибка при создании аккаунта"), http.StatusSeeOther)
		return
	}

	h.Session.Put(r.Context(), "userID", user.ID)
	h.Session.Put(r.Context(), "role", "customer")

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *AuthHandler) LoginEmail(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	email := strings.TrimSpace(strings.ToLower(r.FormValue("email")))
	password := r.FormValue("password")

	if email == "" || password == "" {
		http.Redirect(w, r, "/login?error="+url.QueryEscape("Заполните email и пароль"), http.StatusSeeOther)
		return
	}

	var user models.User
	if err := h.DB.Where("email = ?", email).First(&user).Error; err != nil {
		http.Redirect(w, r, "/login?error="+url.QueryEscape("Неверный email или пароль"), http.StatusSeeOther)
		return
	}

	if user.PasswordHash == "" {
		http.Redirect(w, r, "/login?error="+url.QueryEscape("Этот аккаунт зарегистрирован через OAuth (GitHub/GitLab)"), http.StatusSeeOther)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		http.Redirect(w, r, "/login?error="+url.QueryEscape("Неверный email или пароль"), http.StatusSeeOther)
		return
	}

	h.Session.Put(r.Context(), "userID", user.ID)
	h.Session.Put(r.Context(), "role", "customer")

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// GetUserFromSession helper
func GetUserFromSession(db *gorm.DB, session *scs.SessionManager, r *http.Request) (*models.User, string) {
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer rdb.Close()
	return GetUserFromSessionWithRedis(db, session, rdb, r)
}

func GetUserFromSessionWithRedis(db *gorm.DB, session *scs.SessionManager, rdb *redis.Client, r *http.Request) (*models.User, string) {
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
	if rVal, ok := roleVal.(string); ok && (rVal == "customer" || rVal == "freelancer") {
		role = rVal
	}

	if rdb != nil {
		// Calculate unread chat messages for this user across all their orders
		var orders []models.Order
		if role == "freelancer" {
			db.Where("freelancer_id = ?", user.ID).Find(&orders)
		} else {
			db.Where("customer_id = ?", user.ID).Find(&orders)
		}

		totalUnreadChats := 0
		ctx := r.Context()
		for _, o := range orders {
			streamKey := fmt.Sprintf("chat:order:%d", o.ID)
			lastReadID, err := rdb.Get(ctx, fmt.Sprintf("chat:order:%d:user:%d:last_read", o.ID, user.ID)).Result()
			if err == redis.Nil || lastReadID == "" {
				lastReadID = "-"
			}

			var start string
			if lastReadID == "-" {
				start = "-"
			} else {
				start = "(" + lastReadID
			}

			streams, err := rdb.XRange(ctx, streamKey, start, "+").Result()
			if err == nil {
				for _, s := range streams {
					msgStr, ok := s.Values["message"].(string)
					if !ok {
						continue
					}
					var msg struct {
						SenderID uint `json:"sender_id"`
					}
					if err := json.Unmarshal([]byte(msgStr), &msg); err == nil {
						if msg.SenderID != user.ID {
							totalUnreadChats++
						}
					}
				}
			}
		}

		user.UnreadNotifications += totalUnreadChats
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
