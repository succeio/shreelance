package handlers

import (
	"net/http"

	"github.com/alexedwards/scs/v2"
	"github.com/gorilla/csrf"
	"gorm.io/gorm"
	g "maragu.dev/gomponents"
	html "maragu.dev/gomponents/html"

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

	content := h.renderHomePage(user, role)
	layout := ui.Layout(ui.PageParams{
		Title:       "Главная",
		Content:     content,
		User:        user,
		CSRFToken:   csrf.Token(r),
		ContextRole: role,
	})

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = layout.Render(w)
}

func (h *HomeHandler) renderHomePage(user interface{}, role string) g.Node {
	return html.Div(
		html.Class("text-center py-20 bg-gradient-to-tr from-indigo-50 to-white rounded-2xl shadow-sm border border-indigo-50 px-4"),
		html.H1(html.Class("text-5xl font-extrabold text-gray-900 tracking-tight mb-6"), g.Text("Биржа фриланса нового поколения")),
		html.P(html.Class("text-xl text-gray-600 max-w-2xl mx-auto mb-10 leading-relaxed"), g.Text("Один аккаунт для заказа задач и для их исполнения. Легко переключайтесь между ролями в личном кабинете.")),
		html.Div(
			html.Class("flex justify-center gap-4"),
			html.A(
				html.Href("/orders"),
				html.Class("bg-indigo-600 hover:bg-indigo-700 text-white font-semibold px-8 py-3.5 rounded-lg shadow-lg hover:shadow-indigo-200 transition-all"),
				g.Text("Смотреть заказы"),
			),
		),
	)
}
