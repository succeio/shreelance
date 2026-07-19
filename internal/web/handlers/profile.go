package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/gorilla/csrf"
	"gorm.io/gorm"
	g "maragu.dev/gomponents"
	html "maragu.dev/gomponents/html"

	"shreelance/internal/models"
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

	content := h.renderProfilePage(user, role)
	layout := ui.Layout(ui.PageParams{
		Title:       "Профиль",
		Content:     content,
		User:        user,
		CSRFToken:   csrf.Token(r),
		ContextRole: role,
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

func (h *ProfileHandler) renderProfilePage(user interface{}, role string) g.Node {
	u := user.(*models.User)
	
	// Create CSRF token node placeholder if we need it in nested form (we'll fetch it from layout context parameter if needed, but since we are generating form, we can just pass csrf token inside or rely on HTMX header, but standard form post needs hidden input. We can use a trick: in Show, we will render it. Wait! Let's ensure csrf is in the form)
	
	var specialistSection g.Node
	if role == "freelancer" {
		specialistSection = html.Div(
			html.Class("border-t border-gray-200 pt-6 mt-6"),
			html.H2(html.Class("text-xl font-semibold mb-4 text-gray-900"), g.Text("Настройки профиля специалиста")),
			
			// GitHub Sync Button
			html.Form(
				html.Action("/profile/sync"),
				html.Method("POST"),
				html.Class("mb-6 bg-slate-50 p-4 rounded-lg border border-slate-100 flex items-center justify-between"),
				html.Div(
					html.P(html.Class("text-sm font-semibold text-slate-700"), g.Text("Импорт профиля с GitHub")),
					html.P(html.Class("text-xs text-slate-500"), g.Text("Автоматически заполнить стек технологиями из репозиториев и рассчитать опыт на основе даты создания аккаунта GitHub.")),
				),
				html.Button(
					html.Type("submit"),
					html.Class("bg-slate-900 hover:bg-slate-800 text-white font-medium text-xs py-2 px-4 rounded transition-colors flex items-center space-x-1.5"),
					g.Text("Синхронизировать"),
				),
			),

			// Edit Profile Form
			html.Form(
				html.Action("/profile/update"),
				html.Method("POST"),
				html.Class("space-y-4"),
				html.Div(
					html.Label(html.Class("block text-sm font-semibold text-gray-700 mb-1"), g.Text("Технологический стек")),
					html.Input(
						html.Type("text"),
						html.Name("stack"),
						html.Value(u.Stack),
						html.Placeholder("Например: Go, TypeScript, React, PostgreSQL"),
						html.Class("w-full border border-gray-300 rounded px-3 py-2 focus:ring-indigo-500 focus:border-indigo-500"),
					),
					html.P(html.Class("text-xs text-gray-400 mt-1"), g.Text("Перечислите технологии через запятую")),
				),
				html.Div(
					html.Label(html.Class("block text-sm font-semibold text-gray-700 mb-1"), g.Text("Опыт работы (лет)")),
					html.Input(
						html.Type("number"),
						html.Name("experience_years"),
						html.Value(strconv.Itoa(u.ExperienceYears)),
						html.Min("0"),
						html.Class("w-full border border-gray-300 rounded px-3 py-2 focus:ring-indigo-500 focus:border-indigo-500"),
					),
				),
				html.Button(
					html.Type("submit"),
					html.Class("w-full bg-indigo-600 hover:bg-indigo-700 text-white font-semibold py-2.5 rounded transition-colors"),
					g.Text("Сохранить изменения"),
				),
			),
		)
	}

	return html.Div(
		html.Class("max-w-2xl mx-auto bg-white p-8 rounded-lg shadow-md"),
		html.Div(
			html.Class("flex items-center space-x-6 mb-8"),
			html.Img(html.Src(u.AvatarURL), html.Alt(u.Username), html.Class("w-24 h-24 rounded-full border-4 border-indigo-100")),
			html.Div(
				html.H1(html.Class("text-3xl font-bold text-gray-900"), g.Text(u.Username)),
				html.P(html.Class("text-sm text-gray-500"), g.Text(u.Email)),
				g.If(u.Stack != "", html.Div(
					html.Class("mt-2 flex flex-wrap gap-1.5"),
					g.Group(func() []g.Node {
						var tags []g.Node
						for _, s := range strings.Split(u.Stack, ",") {
							trimmed := strings.TrimSpace(s)
							if trimmed != "" {
								tags = append(tags, html.Span(
									html.Class("inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-indigo-50 text-indigo-700 border border-indigo-100"),
									g.Text(trimmed),
								))
							}
						}
						return tags
					}()),
				)),
			),
		),
		html.Div(
			html.Class("border-t border-gray-200 pt-6"),
			html.H2(html.Class("text-xl font-semibold mb-4 text-gray-900"), g.Text("Текущий контекст интерфейса")),
			html.P(html.Class("text-gray-600 mb-4 text-sm"), g.Text("Вы можете свободно переключаться между ролями Заказчика и Исполнителя.")),
			html.Div(
				html.Class("bg-indigo-50 border border-indigo-100 rounded-lg p-4 flex items-center justify-between"),
				html.Div(
					html.P(html.Class("text-xs text-indigo-700 font-semibold uppercase tracking-wider"), g.Text("Активная роль")),
					html.P(html.Class("text-lg font-bold text-indigo-900 mt-0.5"), g.Text(map[string]string{"customer": "Заказчик (Публикация заданий)", "freelancer": "Исполнитель (Отклики на задания)"}[role])),
				),
			),
		),
		specialistSection,
		
		// GitHub Activity Contribution Grid
		html.Div(
			html.Class("border-t border-gray-200 pt-6 mt-6"),
			html.H2(html.Class("text-xl font-semibold mb-4 text-gray-900"), g.Text("Активность на GitHub")),
			html.P(html.Class("text-xs text-gray-500 mb-4"), g.Text("История вкладов (commits, pull requests, issues) за последний год")),
			html.Div(
				html.Class("bg-slate-50 p-4 rounded-lg border border-slate-100 flex justify-center overflow-x-auto"),
				html.Img(
					html.Src("https://ghchart.rshah.org/4f46e5/"+u.Username),
					html.Alt(u.Username+"'s GitHub Contributions Chart"),
					html.Class("max-w-full h-auto min-w-[600px]"),
				),
			),
		),
	)
}
