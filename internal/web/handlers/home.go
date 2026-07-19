package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/alexedwards/scs/v2"
	"github.com/gorilla/csrf"
	"gorm.io/gorm"
	g "maragu.dev/gomponents"
	html "maragu.dev/gomponents/html"

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
		// Customer view: List of Specialists
		var specialists []models.User
		query := h.DB.Model(&models.User{}).Where("stack IS NOT NULL AND stack != ''")
		
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
		content = h.renderSpecialistsDashboard(specialists, search, tech, minExpStr, sortBy)
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
		content = h.renderOrdersDashboard(orders, search, minBudgetStr, maxBudgetStr, sortBy, user != nil)
	}

	layout := ui.Layout(ui.PageParams{
		Title:       "Панель управления",
		Content:     content,
		User:        user,
		CSRFToken:   csrf.Token(r),
		ContextRole: role,
	})

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = layout.Render(w)
}

func (h *HomeHandler) renderSpecialistsDashboard(specialists []models.User, search, tech, minExp, sort string) g.Node {
	var specCards []g.Node
	for _, s := range specialists {
		specCards = append(specCards, html.Div(
			html.Class("bg-white p-6 rounded-lg shadow-sm border border-gray-100 hover:shadow-md transition-shadow flex items-start space-x-4"),
			html.Img(html.Src(s.AvatarURL), html.Alt(s.Username), html.Class("w-14 h-14 rounded-full border border-gray-100")),
			html.Div(
				html.Class("flex-grow"),
				html.H3(html.Class("text-lg font-bold text-gray-900"), g.Text(s.Username)),
				html.P(html.Class("text-xs text-gray-500 mb-2"), g.Text(fmt.Sprintf("Опыт работы: %d %s", s.ExperienceYears, pluralizeYears(s.ExperienceYears)))),
				g.If(s.Stack != "", html.Div(
					html.Class("flex flex-wrap gap-1 mt-1"),
					g.Group(func() []g.Node {
						var tags []g.Node
						for _, t := range strings.Split(s.Stack, ",") {
							trimmed := strings.TrimSpace(t)
							if trimmed != "" {
								tags = append(tags, html.Span(
									html.Class("inline-flex items-center px-2 py-0.5 rounded text-[10px] font-semibold bg-indigo-50 text-indigo-700 border border-indigo-100"),
									g.Text(trimmed),
								))
							}
						}
						return tags
					}()),
				)),
			),
		))
	}

	if len(specCards) == 0 {
		specCards = append(specCards, html.Div(
			html.Class("col-span-full text-center py-12 text-gray-500 bg-white rounded-lg border border-gray-100"),
			g.Text("Специалисты не найдены по заданным критериям."),
		))
	}

	return html.Div(
		html.Class("grid grid-cols-1 lg:grid-cols-4 gap-8"),
		// Sidebar Filters
		html.Div(
			html.Class("lg:col-span-1 bg-white p-6 rounded-lg shadow-sm border border-gray-100 self-start"),
			html.H2(html.Class("text-lg font-bold text-gray-900 mb-4"), g.Text("Фильтры")),
			html.Form(
				html.Method("GET"),
				html.Class("space-y-4"),
				html.Div(
					html.Label(html.Class("block text-xs font-semibold text-gray-700 mb-1 uppercase tracking-wider"), g.Text("Поиск")),
					html.Input(html.Type("text"), html.Name("search"), html.Value(search), html.Placeholder("Имя или навык..."), html.Class("w-full border border-gray-300 rounded px-3 py-2 text-sm focus:ring-indigo-500 focus:border-indigo-500")),
				),
				html.Div(
					html.Label(html.Class("block text-xs font-semibold text-gray-700 mb-1 uppercase tracking-wider"), g.Text("Конкретная технология")),
					html.Input(html.Type("text"), html.Name("tech"), html.Value(tech), html.Placeholder("Например: Go"), html.Class("w-full border border-gray-300 rounded px-3 py-2 text-sm focus:ring-indigo-500 focus:border-indigo-500")),
				),
				html.Div(
					html.Label(html.Class("block text-xs font-semibold text-gray-700 mb-1 uppercase tracking-wider"), g.Text("Минимальный опыт (лет)")),
					html.Input(html.Type("number"), html.Name("min_exp"), html.Value(minExp), html.Min("0"), html.Class("w-full border border-gray-300 rounded px-3 py-2 text-sm focus:ring-indigo-500 focus:border-indigo-500")),
				),
				html.Div(
					html.Label(html.Class("block text-xs font-semibold text-gray-700 mb-1 uppercase tracking-wider"), g.Text("Сортировка")),
					html.Select(
						html.Name("sort"),
						html.Class("w-full border border-gray-300 rounded px-3 py-2 text-sm focus:ring-indigo-500 focus:border-indigo-500"),
						html.Option(g.Attr("value", "username_asc"), g.If(sort == "username_asc" || sort == "", g.Attr("selected", "selected")), g.Text("По имени (А-Я)")),
						html.Option(g.Attr("value", "username_desc"), g.If(sort == "username_desc", g.Attr("selected", "selected")), g.Text("По имени (Я-А)")),
						html.Option(g.Attr("value", "exp_desc"), g.If(sort == "exp_desc", g.Attr("selected", "selected")), g.Text("По убыванию опыта")),
						html.Option(g.Attr("value", "exp_asc"), g.If(sort == "exp_asc", g.Attr("selected", "selected")), g.Text("По возрастанию опыта")),
					),
				),
				html.Button(
					html.Type("submit"),
					html.Class("w-full bg-indigo-600 hover:bg-indigo-700 text-white font-semibold py-2 rounded text-sm transition-colors"),
					g.Text("Применить"),
				),
				html.A(
					html.Href("/"),
					html.Class("block text-center text-xs text-gray-500 hover:text-indigo-600 mt-2"),
					g.Text("Сбросить все"),
				),
			),
		),
		// Specialists Grid
		html.Div(
			html.Class("lg:col-span-3 space-y-6"),
			html.H1(html.Class("text-3xl font-extrabold text-gray-900"), g.Text("Наши специалисты")),
			html.Div(
				html.Class("grid grid-cols-1 md:grid-cols-2 gap-6"),
				g.Group(specCards),
			),
		),
	)
}

func (h *HomeHandler) renderOrdersDashboard(orders []models.Order, search, minBudget, maxBudget, sort string, isLoggedIn bool) g.Node {
	var orderCards []g.Node
	for _, o := range orders {
		orderCards = append(orderCards, html.Div(
			html.Class("bg-white p-6 rounded-lg shadow-sm border border-gray-100 hover:shadow-md transition-shadow"),
			html.Div(
				html.Class("flex justify-between items-start mb-4"),
				html.H3(
					html.Class("text-xl font-bold text-gray-900"),
					html.A(html.Href(fmt.Sprintf("/orders/%d", o.ID)), html.Class("hover:text-indigo-600"), g.Text(o.Title)),
				),
				html.Span(
					html.Class("text-lg font-extrabold text-green-600"),
					g.Text(fmt.Sprintf("%.0f ₽", o.Budget)),
				),
			),
			html.P(html.Class("text-gray-600 mb-4 line-clamp-3 text-sm"), g.Text(o.Description)),
			html.Div(
				html.Class("flex justify-between items-center text-xs text-gray-400"),
				html.Span(g.Text("Заказчик: "+o.Customer.Username)),
				html.Span(g.Text(o.CreatedAt.Format("02.01.2006 15:04"))),
			),
		))
	}

	if len(orderCards) == 0 {
		orderCards = append(orderCards, html.Div(
			html.Class("col-span-full text-center py-12 text-gray-500 bg-white rounded-lg border border-gray-100"),
			g.Text("Заказы не найдены по заданным критериям."),
		))
	}

	var headerSection g.Node
	if !isLoggedIn {
		headerSection = html.Div(
			html.Class("text-center py-10 bg-gradient-to-tr from-indigo-50 to-white rounded-2xl shadow-sm border border-indigo-50 px-4 mb-8"),
			html.H1(html.Class("text-4xl font-extrabold text-gray-900 tracking-tight mb-4"), g.Text("Биржа фриланса нового поколения")),
			html.P(html.Class("text-base text-gray-600 max-w-2xl mx-auto mb-6 leading-relaxed"), g.Text("Один аккаунт для заказа задач и для их исполнения. Авторизуйтесь через GitHub, чтобы начать работу.")),
			html.A(
				html.Href("/auth/github"),
				html.Class("inline-block bg-indigo-600 hover:bg-indigo-700 text-white font-semibold px-6 py-2.5 rounded-lg shadow-md hover:shadow-indigo-200 transition-all"),
				g.Text("Войти через GitHub"),
			),
		)
	}

	return html.Div(
		html.Class("space-y-6"),
		headerSection,
		html.Div(
			html.Class("grid grid-cols-1 lg:grid-cols-4 gap-8"),
			// Sidebar Filters
			html.Div(
				html.Class("lg:col-span-1 bg-white p-6 rounded-lg shadow-sm border border-gray-100 self-start"),
				html.H2(html.Class("text-lg font-bold text-gray-900 mb-4"), g.Text("Фильтры")),
				html.Form(
					html.Method("GET"),
					html.Class("space-y-4"),
					html.Div(
						html.Label(html.Class("block text-xs font-semibold text-gray-700 mb-1 uppercase tracking-wider"), g.Text("Поиск")),
						html.Input(html.Type("text"), html.Name("search"), html.Value(search), html.Placeholder("Название или описание..."), html.Class("w-full border border-gray-300 rounded px-3 py-2 text-sm focus:ring-indigo-500 focus:border-indigo-500")),
					),
					html.Div(
						html.Label(html.Class("block text-xs font-semibold text-gray-700 mb-1 uppercase tracking-wider"), g.Text("Минимальный бюджет (₽)")),
						html.Input(html.Type("number"), html.Name("min_budget"), html.Value(minBudget), html.Class("w-full border border-gray-300 rounded px-3 py-2 text-sm focus:ring-indigo-500 focus:border-indigo-500")),
					),
					html.Div(
						html.Label(html.Class("block text-xs font-semibold text-gray-700 mb-1 uppercase tracking-wider"), g.Text("Максимальный бюджет (₽)")),
						html.Input(html.Type("number"), html.Name("max_budget"), html.Value(maxBudget), html.Class("w-full border border-gray-300 rounded px-3 py-2 text-sm focus:ring-indigo-500 focus:border-indigo-500")),
					),
					html.Div(
						html.Label(html.Class("block text-xs font-semibold text-gray-700 mb-1 uppercase tracking-wider"), g.Text("Сортировка")),
						html.Select(
							html.Name("sort"),
							html.Class("w-full border border-gray-300 rounded px-3 py-2 text-sm focus:ring-indigo-500 focus:border-indigo-500"),
							html.Option(g.Attr("value", "created_desc"), g.If(sort == "created_desc" || sort == "", g.Attr("selected", "selected")), g.Text("Сначала новые")),
							html.Option(g.Attr("value", "created_asc"), g.If(sort == "created_asc", g.Attr("selected", "selected")), g.Text("Сначала старые")),
							html.Option(g.Attr("value", "budget_desc"), g.If(sort == "budget_desc", g.Attr("selected", "selected")), g.Text("Бюджет: по убыванию")),
							html.Option(g.Attr("value", "budget_asc"), g.If(sort == "budget_asc", g.Attr("selected", "selected")), g.Text("Бюджет: по возрастанию")),
						),
					),
					html.Button(
						html.Type("submit"),
						html.Class("w-full bg-indigo-600 hover:bg-indigo-700 text-white font-semibold py-2 rounded text-sm transition-colors"),
						g.Text("Применить"),
					),
					html.A(
						html.Href("/"),
						html.Class("block text-center text-xs text-gray-500 hover:text-indigo-600 mt-2"),
						g.Text("Сбросить все"),
					),
				),
			),
			// Orders List
			html.Div(
				html.Class("lg:col-span-3 space-y-6"),
				html.H1(html.Class("text-3xl font-extrabold text-gray-900"), g.Text("Доступные заказы")),
				html.Div(
					html.Class("grid grid-cols-1 md:grid-cols-2 gap-6"),
					g.Group(orderCards),
				),
			),
		),
	)
}

func pluralizeYears(years int) string {
	if years%10 == 1 && years%100 != 11 {
		return "год"
	}
	if (years%10 >= 2 && years%10 <= 4) && (years%100 < 12 || years%100 > 14) {
		return "года"
	}
	return "лет"
}
