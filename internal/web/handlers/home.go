package handlers

import (
	"net/http"
	"strconv"

	"github.com/alexedwards/scs/v2"
	"github.com/gorilla/csrf"
	"gorm.io/gorm"
	g "maragu.dev/gomponents"

	"shreelance/internal/models"
	"shreelance/internal/ui"
)

type HomeHandler struct {
	DB      *gorm.DB
	Session *scs.SessionManager
}

func NewHomeHandler(db *gorm.DB, session *scs.SessionManager) *HomeHandler {
	return &HomeHandler{
		DB:      db,
		Session: session,
	}
}

func (h *HomeHandler) Show(w http.ResponseWriter, r *http.Request) {
	user, role := GetUserFromSession(h.DB, h.Session, r)

	var content g.Node
	if user != nil && role == "customer" {
		// Customer view: List of Specialists (only users with github_id IS NOT NULL OR gitlab_id IS NOT NULL)
		var specialists []models.User
		query := h.DB.Model(&models.User{}).Where("github_id IS NOT NULL OR gitlab_id IS NOT NULL")
		
		search := r.URL.Query().Get("search")
		if search != "" {
			query = query.Where("username ILIKE ? OR stack ILIKE ?", "%"+search+"%", "%"+search+"%")
		}
		
		tech := r.URL.Query().Get("tech")
		if tech != "" {
			query = query.Where("stack ILIKE ?", "%"+tech+"%")
		}
		
		minExpStr := r.URL.Query().Get("min_exp")
		if minExpStr != "" {
			if minExp, err := strconv.Atoi(minExpStr); err == nil {
				query = query.Where("experience_years >= ?", minExp)
			}
		}
		
		sortBy := r.URL.Query().Get("sort")
		switch sortBy {
		case "exp_desc":
			query = query.Order("experience_years desc")
		case "exp_asc":
			query = query.Order("experience_years asc")
		case "username_desc":
			query = query.Order("username desc")
		default:
			query = query.Order("username asc")
		}
		
		if err := query.Find(&specialists).Error; err != nil {
			http.Error(w, "Failed to load specialists", http.StatusInternalServerError)
			return
		}
		content = ui.SpecialistsDashboard(specialists, search, tech, minExpStr, sortBy)
	} else {
		// Freelancer (Specialist) view or Guest: List of Tasks/Orders
		var orders []models.Order
		query := h.DB.Preload("Customer")
		
		if user != nil {
			query = query.Where("status = ? OR customer_id = ? OR id IN (SELECT order_id FROM bids WHERE freelancer_id = ?)", "open", user.ID, user.ID)
		} else {
			query = query.Where("status = ?", "open")
		}
		
		search := r.URL.Query().Get("search")
		if search != "" {
			query = query.Where("title ILIKE ? OR description ILIKE ?", "%"+search+"%", "%"+search+"%")
		}
		
		minBudgetStr := r.URL.Query().Get("min_budget")
		if minBudgetStr != "" {
			if minBudget, err := strconv.ParseFloat(minBudgetStr, 64); err == nil {
				query = query.Where("budget >= ?", minBudget)
			}
		}
		
		maxBudgetStr := r.URL.Query().Get("max_budget")
		if maxBudgetStr != "" {
			if maxBudget, err := strconv.ParseFloat(maxBudgetStr, 64); err == nil {
				query = query.Where("budget <= ?", maxBudget)
			}
		}
		
		sortBy := r.URL.Query().Get("sort")
		switch sortBy {
		case "budget_desc":
			query = query.Order("budget desc")
		case "budget_asc":
			query = query.Order("budget asc")
		case "created_asc":
			query = query.Order("created_at asc")
		default:
			query = query.Order("created_at desc")
		}
		
		if err := query.Find(&orders).Error; err != nil {
			http.Error(w, "Failed to load orders", http.StatusInternalServerError)
			return
		}
		content = ui.OrdersDashboard(orders, search, minBudgetStr, maxBudgetStr, sortBy, user != nil)
	}

	layout := ui.Layout(ui.PageParams{
		Title:       "Панель управления",
		Content:     content,
		User:        user,
		CSRFToken:   csrf.Token(r),
		ContextRole: role,
		Theme:       GetThemeFromCookie(r),
	})

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = layout.Render(w)
}
