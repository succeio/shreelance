package handlers

import (
	"net/http"

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

func (h *ProfileHandler) renderProfilePage(user interface{}, role string) g.Node {
	u := user.(*models.User)
	return html.Div(
		html.Class("max-w-2xl mx-auto bg-white p-8 rounded-lg shadow-md"),
		html.Div(
			html.Class("flex items-center space-x-6 mb-8"),
			html.Img(html.Src(u.AvatarURL), html.Alt(u.Username), html.Class("w-24 h-24 rounded-full border-4 border-indigo-100")),
			html.Div(
				html.H1(html.Class("text-3xl font-bold text-gray-900"), g.Text(u.Username)),
				html.P(html.Class("text-sm text-gray-500"), g.Text(u.Email)),
			),
		),
		html.Div(
			html.Class("border-t border-gray-200 pt-6"),
			html.H2(html.Class("text-xl font-semibold mb-4"), g.Text("Текущий контекст интерфейса")),
			html.P(html.Class("text-gray-600 mb-4"), g.Text("Вы можете свободно переключаться между ролями Заказчика и Исполнителя.")),
			html.Div(
				html.Class("bg-indigo-50 border border-indigo-100 rounded-lg p-4 flex items-center justify-between"),
				html.Div(
					html.P(html.Class("text-sm text-indigo-700 font-semibold"), g.Text("Активная роль")),
					html.P(html.Class("text-lg font-bold text-indigo-900"), g.Text(map[string]string{"customer": "Заказчик (Публикация заданий)", "freelancer": "Исполнитель (Отклики на задания)"}[role])),
				),
			),
		),
	)
}
