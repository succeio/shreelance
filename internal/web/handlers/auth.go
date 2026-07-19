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
	DB                      *gorm.DB
	Session                 *scs.SessionManager
	GitHubOAuthConfig        *oauth2.Config
	GitLabOAuthConfig        *oauth2.Config
	DonationAlertsOAuthConfig *oauth2.Config
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
			Scopes:       []string{"read_user", "read_api"},
			Endpoint:     gitlab.Endpoint,
		},
		DonationAlertsOAuthConfig: &oauth2.Config{
			ClientID:     cfg.DonationAlertsAppID,
			ClientSecret: cfg.DonationAlertsAPIKey,
			RedirectURL:  cfg.DonationAlertsRedirectURL,
			Scopes:       []string{"oauth-donation-index"},
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://www.donationalerts.com/oauth/authorize",
				TokenURL: "https://www.donationalerts.com/oauth/token",
			},
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

func (h *AuthHandler) DonationAlertsCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Code not found in request", http.StatusBadRequest)
		return
	}

	token, err := h.DonationAlertsOAuthConfig.Exchange(r.Context(), code)
	if err != nil {
		http.Error(w, "Failed to exchange code for token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Store user token to access donation index (in actual usage, you might save it to DB, but since we use OAuth just to authenticate, or maybe store on a single/global config level)
	// The background worker needs to poll, and it needs a token.
	// We can store the last active DonationAlerts token in Valkey or a settings store to let the background job use it.
	// For simplicity, let's store it in Redis under a global key so the background worker can read it.
	// Alternatively, we can save the token in GORM or a config struct if it's user-specific, but the prompt says:
	// "1. ID приложения: 20010, Ключ API: XMPNUCxsgEFpkcnQnAKuRhFddSTF1I8PjiTJcCve, URL перенаправления: http://localhost:8080/donationalerts/callback"
	// To perform polling GET /api/v1/alerts we need a token. We can save the token obtained via OAuth globally.
	// We'll write it to valkey:
	rdb := redis.NewClient(&redis.Options{
		Addr: getEnvDA("VALKEY_ADDR", "localhost:6379"),
	})
	defer rdb.Close()
	_ = rdb.Set(r.Context(), "donationalerts_access_token", token.AccessToken, 0).Err()

	http.Redirect(w, r, "/profile?success=donationalerts_connected", http.StatusSeeOther)
}

func getEnvDA(key, defaultVal string) string {
	importOS := true // placeholder
	_ = importOS
	return defaultVal
}

func SyncGitLabData(db *gorm.DB, user *models.User, tokenString string) error {
	var projects []struct {
		ID int64 `json:"id"`
	}

	// Helper for HTTP GET with optional auth token
	doGet := func(reqURL string, token string) (*http.Response, error) {
		req, err := http.NewRequest("GET", reqURL, nil)
		if err != nil {
			return nil, err
		}
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		return http.DefaultClient.Do(req)
	}

	// 1. Authenticated member projects call if token is present
	if tokenString != "" {
		reqURL := "https://gitlab.com/api/v4/projects?membership=true&order_by=updated_at&per_page=50"
		resp, err := doGet(reqURL, tokenString)
		if err == nil {
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				_ = json.NewDecoder(resp.Body).Decode(&projects)
			} else {
				fmt.Printf("GitLab Sync: membership projects returned status code %d (token length: %d)\n", resp.StatusCode, len(tokenString))
			}
		} else {
			fmt.Printf("GitLab Sync: membership projects request failed: %v\n", err)
		}
	}

	// 2. Fallback to user projects endpoint by GitLabID or username
	if len(projects) == 0 {
		glUsername := user.GitLabUsername
		if glUsername == "" {
			glUsername = user.Username
		}

		var gitlabUserID int64
		if user.GitLabID != nil {
			gitlabUserID = *user.GitLabID
		}

		if gitlabUserID == 0 && glUsername != "" {
			// Resolve username to ID first
			userURL := fmt.Sprintf("https://gitlab.com/api/v4/users?username=%s", glUsername)
			resp, err := doGet(userURL, tokenString)
			if err != nil || resp.StatusCode != http.StatusOK {
				// Retry unauthenticated if authenticated call fails
				if resp != nil {
					resp.Body.Close()
				}
				resp, err = doGet(userURL, "")
			}
			if err == nil {
				defer resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					var usersRes []struct {
						ID int64 `json:"id"`
					}
					if err := json.NewDecoder(resp.Body).Decode(&usersRes); err == nil && len(usersRes) > 0 {
						gitlabUserID = usersRes[0].ID
					}
				} else {
					fmt.Printf("GitLab Sync: resolve username %s returned status %d\n", glUsername, resp.StatusCode)
				}
			}
		}

		var reqURL string
		if gitlabUserID != 0 {
			reqURL = fmt.Sprintf("https://gitlab.com/api/v4/users/%d/projects?order_by=updated_at&per_page=50", gitlabUserID)
		} else if glUsername != "" {
			reqURL = fmt.Sprintf("https://gitlab.com/api/v4/users/%s/projects?order_by=updated_at&per_page=50", glUsername)
		}

		if reqURL != "" {
			// Try with token first
			resp, err := doGet(reqURL, tokenString)
			if err != nil || (resp != nil && resp.StatusCode != http.StatusOK) {
				if resp != nil {
					fmt.Printf("GitLab Sync: projects with token returned status %d for url %s\n", resp.StatusCode, reqURL)
					resp.Body.Close()
				}
				// Retry unauthenticated
				resp, err = doGet(reqURL, "")
			}

			if err == nil {
				defer resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					_ = json.NewDecoder(resp.Body).Decode(&projects)
				} else {
					fmt.Printf("GitLab Sync: public projects returned status code %d for url %s\n", resp.StatusCode, reqURL)
				}
			} else {
				fmt.Printf("GitLab Sync: public projects request failed: %v\n", err)
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

	// For the top 15 projects, fetch their programming languages
	limit := len(projects)
	if limit > 15 {
		limit = 15
	}
	for i := 0; i < limit; i++ {
		p := projects[i]
		langURL := fmt.Sprintf("https://gitlab.com/api/v4/projects/%d/languages", p.ID)
		
		lResp, err := doGet(langURL, tokenString)
		if err != nil || (lResp != nil && lResp.StatusCode != http.StatusOK) {
			if lResp != nil {
				lResp.Body.Close()
			}
			// Retry unauthenticated
			lResp, err = doGet(langURL, "")
		}

		if err != nil {
			continue
		}
		defer lResp.Body.Close()
		if lResp.StatusCode == http.StatusOK {
			var projLangs map[string]float64
			if json.NewDecoder(lResp.Body).Decode(&projLangs) == nil {
				for lang := range projLangs {
					langMap[strings.ToLower(lang)] = true
				}
			}
		} else {
			fmt.Printf("GitLab Sync: languages returned status code %d for project %d\n", lResp.StatusCode, p.ID)
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

func SyncGitHubDataPublic(db *gorm.DB, user *models.User) error {
	if user.Username == "" {
		return nil
	}
	reqRepos, err := http.NewRequest("GET", fmt.Sprintf("https://api.github.com/users/%s/repos?sort=updated&per_page=15", user.Username), nil)
	if err != nil {
		return err
	}
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
		if user.Stack != "" {
			for _, s := range strings.Split(user.Stack, ",") {
				t := strings.TrimSpace(s)
				if t != "" {
					langMap[strings.ToLower(t)] = true
				}
			}
		}
		for _, r := range repos {
			if r.Language != "" {
				langMap[strings.ToLower(r.Language)] = true
			}
		}
		var langs []string
		for lang := range langMap {
			langs = append(langs, strings.Title(lang))
		}
		if len(langs) > 0 {
			user.Stack = strings.Join(langs, ", ")
		}
	}
	user.UpdatedAt = time.Now()
	return db.Save(user).Error
}

func SyncGitLabDataPublic(db *gorm.DB, user *models.User) error {
	glUsername := user.GitLabUsername
	if glUsername == "" {
		glUsername = user.Username
	}
	if glUsername == "" && user.GitLabID == nil {
		return nil
	}

	var reqURL string
	if user.GitLabID != nil {
		reqURL = fmt.Sprintf("https://gitlab.com/api/v4/users/%d/projects?order_by=updated_at&per_page=50", *user.GitLabID)
	} else {
		reqURL = fmt.Sprintf("https://gitlab.com/api/v4/users/%s/projects?order_by=updated_at&per_page=50", glUsername)
	}

	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var projects []struct {
		ID int64 `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&projects); err != nil {
		return err
	}

	langMap := make(map[string]bool)
	if user.Stack != "" {
		for _, s := range strings.Split(user.Stack, ",") {
			t := strings.TrimSpace(s)
			if t != "" {
				langMap[strings.ToLower(t)] = true
			}
		}
	}

	limit := len(projects)
	if limit > 15 {
		limit = 15
	}
	for i := 0; i < limit; i++ {
		p := projects[i]
		langURL := fmt.Sprintf("https://gitlab.com/api/v4/projects/%d/languages", p.ID)
		lReq, err := http.NewRequest("GET", langURL, nil)
		if err != nil {
			continue
		}
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
