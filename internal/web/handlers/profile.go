package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/gorilla/csrf"
	"gorm.io/gorm"

	"shreelance/internal/config"
	"shreelance/internal/models"
	"shreelance/internal/ui"
)

type ProfileHandler struct {
	DB      *gorm.DB
	Session *scs.SessionManager
	Cfg     *config.Config
}

func NewProfileHandler(db *gorm.DB, session *scs.SessionManager, cfg *config.Config) *ProfileHandler {
	return &ProfileHandler{
		DB:      db,
		Session: session,
		Cfg:     cfg,
	}
}

func (h *ProfileHandler) Show(w http.ResponseWriter, r *http.Request) {
	user, role := GetUserFromSession(h.DB, h.Session, r)
	if user == nil {
		http.Redirect(w, r, "/auth/github", http.StatusSeeOther)
		return
	}

	csrfToken := csrf.Token(r)
	errorMsg := r.URL.Query().Get("error")
	content := ui.ProfilePage(user, role, csrfToken, errorMsg)
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

	// Email-registered users without OAuth (GitHub/GitLab) cannot switch to freelancer role
	if role == "freelancer" && user.GitHubID == nil && user.GitLabID == nil {
		http.Redirect(w, r, "/profile?error="+url.QueryEscape("Роль исполнителя доступна только пользователям, авторизованным через GitHub или GitLab"), http.StatusSeeOther)
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

	if user.GitHubToken != "" {
		if err := SyncGitHubData(h.DB, user, user.GitHubToken); err != nil {
			http.Error(w, "Failed to sync with GitHub: "+err.Error(), http.StatusInternalServerError)
			return
		}
	} else if user.GitHubID != nil {
		// Fallback to public GitHub API sync if token is missing
		_ = SyncGitHubDataPublic(h.DB, user)
	}

	if user.GitLabToken != "" {
		if err := SyncGitLabData(h.DB, user, user.GitLabToken); err != nil {
			http.Error(w, "Failed to sync with GitLab: "+err.Error(), http.StatusInternalServerError)
			return
		}
	} else if user.GitLabID != nil {
		// Fallback to public GitLab API sync if token is missing
		_ = SyncGitLabDataPublic(h.DB, user)
	}

	http.Redirect(w, r, "/profile", http.StatusSeeOther)
}

func (h *ProfileHandler) GitLabSVGCard(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("username")
	if username == "" {
		http.Error(w, "Username required", http.StatusBadRequest)
		return
	}

	var user models.User
	if err := h.DB.Where("git_lab_username = ?", username).First(&user).Error; err != nil {
		// Fallback search by general username if git_lab_username is empty
		if err := h.DB.Where("username = ? AND git_lab_id > 0", username).First(&user).Error; err != nil {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
	}

	// Fetch recent events to build a contribution grid (last 365 days)
	contributions := make(map[string]int)
	if user.GitLabToken != "" && user.GitLabID != nil {
		eventsURL := fmt.Sprintf("https://gitlab.com/api/v4/users/%d/events?per_page=100", *user.GitLabID)
		req, err := http.NewRequest("GET", eventsURL, nil)
		if err == nil {
			req.Header.Set("Authorization", "Bearer "+user.GitLabToken)
			resp, err := http.DefaultClient.Do(req)
			if err == nil {
				defer resp.Body.Close()
				var events []struct {
					CreatedAt string `json:"created_at"`
				}
				if json.NewDecoder(resp.Body).Decode(&events) == nil {
					for _, ev := range events {
						if t, err := time.Parse(time.RFC3339, ev.CreatedAt); err == nil {
							dayStr := t.Format("2006-01-02")
							contributions[dayStr]++
						}
					}
				}
			}
		}
	}

	// Align to build a full calendar grid of 53 weeks (371 days)
	// Match ghchart dimensions: step=12 (10 rect + 2 gap), X_PAD = 27, Y_PAD = 20
	// width: 12 * 53 + 27 = 663, height: 12 * 7 + 20 = 104
	now := time.Now()
	startDate := now.AddDate(0, 0, -364)
	for startDate.Weekday() != time.Sunday {
		startDate = startDate.AddDate(0, 0, -1)
	}

	var gridSVG strings.Builder
	var monthLabels strings.Builder
	lastMonth := ""

	currDate := startDate
	for col := 0; col < 53; col++ {
		// Draw columns with step=12 and X_PAD=27
		gridSVG.WriteString(fmt.Sprintf("<g transform=\"translate(%d, 0)\">", col*12+27))
		for row := 0; row < 7; row++ {
			dayStr := currDate.Format("2006-01-02")
			count := contributions[dayStr]

			// Month label handling: exact placement from ghchart
			// Only show month labels from the second column onwards to avoid duplication/clipping at the very left edge,
			// or if the month changes.
			if row == 0 && col > 0 && currDate.Format("Jan") != lastMonth {
				lastMonth = currDate.Format("Jan")
				monthLabels.WriteString(fmt.Sprintf(`<text x="%d" y="10" class="month-label">%s</text>`, col*12+27, lastMonth))
			} else if row == 0 && col == 0 {
				// Store the initial month so we don't duplicate it immediately
				lastMonth = currDate.Format("Jan")
			}

			rectClass := "day-empty"
			fillColor := "#eeeeee"
			if count == 0 {
				rectClass = "day-empty"
				fillColor = "#eeeeee"
			} else if count <= 2 {
				rectClass = "day-low"
				fillColor = "#9be9a8"
			} else if count <= 5 {
				rectClass = "day-med"
				fillColor = "#40c463"
			} else if count <= 10 {
				rectClass = "day-high"
				fillColor = "#30a14e"
			} else {
				rectClass = "day-max"
				fillColor = "#216e39"
			}

			// Square 10x10, Y_PAD=20
			// Use CSS class fill instead of hardcoded fill attribute for day-empty to allow styling overrides
			rectFill := fillColor
			// Make empty cells fill with emptyFill color by default inside SVG
			gridSVG.WriteString(fmt.Sprintf(`<rect width="10" height="10" y="%d" class="%s" fill="%s" data-count="%d" data-date="%s"><title>%s: %d contributions</title></rect>`, row*12+20, rectClass, rectFill, count, dayStr, dayStr, count))
			currDate = currDate.AddDate(0, 0, 1)
		}
		gridSVG.WriteString("</g>")
	}

	emptyFill := "#EEEEEE"

	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")

	labelColor := "#767676"

	svg := fmt.Sprintf(`<svg width="663" height="104" viewBox="0 0 663 104" xmlns="http://www.w3.org/2000/svg">
  <style>
    .month-label {
      font-size: 10px;
      fill: %s;
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Helvetica, Arial, sans-serif;
    }

    .wday-label {
      font-size: 9px;
      fill: %s;
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Helvetica, Arial, sans-serif;
    }

    .day-empty {
      fill: %s;
    }
  </style>

  %s

  <!-- Weekday Labels: aligned precisely to match row y coordinates (Mon: row 1 (y:32), Wed: row 3 (y:56), Fri: row 5 (y:80)) with slightly lower baseline alignment (+7px) -->
  <text x="5" y="39" class="wday-label">Mon</text>
  <text x="5" y="63" class="wday-label">Wed</text>
  <text x="5" y="87" class="wday-label">Fri</text>

  %s
	</svg>`,
		labelColor,
		labelColor,
		emptyFill,
		monthLabels.String(),
		gridSVG.String(),
	)
	_, _ = w.Write([]byte(svg))
}

func (h *ProfileHandler) VerifyStar(w http.ResponseWriter, r *http.Request) {
	user, _ := GetUserFromSession(h.DB, h.Session, r)
	if user == nil {
		http.Redirect(w, r, "/auth/github", http.StatusSeeOther)
		return
	}

	if user.HasStarredRepo {
		http.Redirect(w, r, "/profile", http.StatusSeeOther)
		return
	}

	starred := false

	// Check GitHub star if GitHub account connected
	if user.GitHubID != nil {
		reqURL := fmt.Sprintf("https://api.github.com/user/starred/%s", h.Cfg.RewardGitHubRepo)
		req, err := http.NewRequest("GET", reqURL, nil)
		if err == nil {
			if user.GitHubToken != "" {
				req.Header.Set("Authorization", "Bearer "+user.GitHubToken)
			}
			req.Header.Set("User-Agent", "ShreelanceApp")
			resp, err := http.DefaultClient.Do(req)
			if err == nil {
				defer resp.Body.Close()
				fmt.Printf("VerifyStar GitHub: checking %s with token len %d, response status: %d\n", reqURL, len(user.GitHubToken), resp.StatusCode)
				if resp.StatusCode == http.StatusNoContent {
					starred = true
				}
			} else {
				fmt.Printf("VerifyStar GitHub: request failed: %v\n", err)
			}
		}
	}

	// Check GitLab star if not starred on GitHub and GitLab username exists
	if !starred && (user.GitLabID != nil || user.GitLabUsername != "") {
		glUser := user.GitLabUsername
		if glUser == "" {
			glUser = user.Username
		}
		reqURL := fmt.Sprintf("https://gitlab.com/api/v4/users/%s/starred_projects", glUser)
		req, err := http.NewRequest("GET", reqURL, nil)
		if err == nil {
			if user.GitLabToken != "" {
				req.Header.Set("Authorization", "Bearer "+user.GitLabToken)
			}
			resp, err := http.DefaultClient.Do(req)
			if err == nil {
				defer resp.Body.Close()
				fmt.Printf("VerifyStar GitLab: checking %s with token len %d, response status: %d\n", reqURL, len(user.GitLabToken), resp.StatusCode)
				
				// Fallback if starred_projects is forbidden (e.g. private starred projects list or scope restrictions)
				if resp.StatusCode == http.StatusForbidden && user.GitLabToken != "" {
					escapedProject := strings.ReplaceAll(h.Cfg.RewardGitLabRepo, "/", "%2F")
					projectURL := fmt.Sprintf("https://gitlab.com/api/v4/projects/%s", escapedProject)
					pReq, pErr := http.NewRequest("GET", projectURL, nil)
					if pErr == nil {
						pReq.Header.Set("Authorization", "Bearer "+user.GitLabToken)
						pResp, pErr2 := http.DefaultClient.Do(pReq)
						if pErr2 == nil {
							defer pResp.Body.Close()
							fmt.Printf("VerifyStar GitLab fallback: checking project %s status: %d\n", projectURL, pResp.StatusCode)
							if pResp.StatusCode == http.StatusOK {
								var pDetails struct {
									Starred bool `json:"starred"`
								}
								if json.NewDecoder(pResp.Body).Decode(&pDetails) == nil {
									if pDetails.Starred {
										starred = true
									}
								}
							}
						}
					}
				}

				if resp.StatusCode == http.StatusOK {
					var starredProjects []struct {
						PathWithNamespace string `json:"path_with_namespace"`
					}
					if json.NewDecoder(resp.Body).Decode(&starredProjects) == nil {
						target := strings.ToLower(h.Cfg.RewardGitLabRepo)
						for _, p := range starredProjects {
							if strings.ToLower(p.PathWithNamespace) == target {
								starred = true
								break
							}
						}
					}
				}
			} else {
				fmt.Printf("VerifyStar GitLab: request failed: %v\n", err)
			}
		}
	}

	if starred {
		user.HasStarredRepo = true
		now := time.Now()
		var proBase time.Time
		if user.ProUntil != nil && user.ProUntil.After(now) {
			proBase = *user.ProUntil
		} else {
			proBase = now
		}
		newProUntil := proBase.Add(3 * 24 * time.Hour)
		user.ProUntil = &newProUntil
		h.DB.Save(user)
		http.Redirect(w, r, "/profile?success=pro_granted", http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/profile?error=star_not_found", http.StatusSeeOther)
}
