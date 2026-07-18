package ui

import (
	"shreelance/internal/models"

	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

type PageParams struct {
	Title       string
	Content     g.Node
	User        interface{} // Pass user if logged in
	CSRFToken   string
	ContextRole string // "customer" or "freelancer"
}

func Layout(p PageParams) g.Node {
	return h.HTML(
		h.Lang("ru"),
		h.Head(
			h.Meta(h.Charset("UTF-8")),
			h.Meta(h.Name("viewport"), h.Content("width=device-width, initial-scale=1.0")),
			h.Title(p.Title+" - Shreelance"),
			// Tailwind CSS
			h.Link(h.Rel("stylesheet"), h.Href("/static/style.css")),
			// HTMX v2
			h.Script(h.Src("https://unpkg.com/htmx.org@2.0.0")),
			// Alpine.js CDN
			h.Script(h.Defer(), h.Src("https://unpkg.com/alpinejs@3.x.x/dist/cdn.min.js")),
		),
		h.Body(
			h.Class("bg-gray-50 text-gray-900 min-h-screen flex flex-col"),
			// Pass CSRF token to HTMX requests
			g.Attr("hx-headers", `{"X-CSRF-Token": "`+p.CSRFToken+`"}`),
			
			Navbar(p),
			
			h.Main(
				h.Class("flex-grow container mx-auto px-4 py-8"),
				p.Content,
			),
			
			Footer(),
		),
	)
}

func Navbar(p PageParams) g.Node {
	var authSection g.Node
	// Check if p.User is nil, or if it is a pointer to models.User that is nil
	hasUser := false
	if p.User != nil {
		if u, ok := p.User.(*models.User); ok && u != nil && u.ID > 0 {
			hasUser = true
		}
	}

	if hasUser {
		authSection = h.Div(
			h.Class("flex items-center space-x-4"),
			// Context switch (Customer / Freelancer)
			h.Div(
				h.Class("relative"),
				g.Attr("x-data", "{ open: false }"),
				h.Button(
					h.Class("bg-white border border-gray-300 rounded-md px-3 py-1.5 text-sm font-medium text-gray-700 hover:bg-gray-50 focus:outline-none"),
					g.Attr("@click", "open = !open"),
					g.Text("Режим: "+map[string]string{"customer": "Заказчик", "freelancer": "Исполнитель"}[p.ContextRole]),
				),
				h.Div(
					h.Class("origin-top-right absolute right-0 mt-2 w-48 rounded-md shadow-lg bg-white ring-1 ring-black ring-opacity-5 z-50"),
					g.Attr("x-show", "open"),
					g.Attr("@click.away", "open = false"),
					g.Attr("x-transition", ""),
					h.Div(
						h.Class("py-1"),
						h.Form(
							h.Action("/profile/role"),
							h.Method("POST"),
							h.Input(h.Type("hidden"), h.Name("csrf_token"), h.Value(p.CSRFToken)),
							h.Input(h.Type("hidden"), h.Name("role"), h.Value("customer")),
							h.Button(h.Type("submit"), h.Class("block w-full text-left px-4 py-2 text-sm text-gray-700 hover:bg-gray-100"), g.Text("Заказчик")),
						),
						h.Form(
							h.Action("/profile/role"),
							h.Method("POST"),
							h.Input(h.Type("hidden"), h.Name("csrf_token"), h.Value(p.CSRFToken)),
							h.Input(h.Type("hidden"), h.Name("role"), h.Value("freelancer")),
							h.Button(h.Type("submit"), h.Class("block w-full text-left px-4 py-2 text-sm text-gray-700 hover:bg-gray-100"), g.Text("Исполнитель")),
						),
					),
				),
			),
			h.A(
				h.Href("/profile"),
				h.Class("text-sm font-semibold hover:text-indigo-600"),
				g.Text("Мой Профиль"),
			),
			h.Form(
				h.Action("/auth/logout"),
				h.Method("POST"),
				h.Class("inline"),
				h.Input(h.Type("hidden"), h.Name("csrf_token"), h.Value(p.CSRFToken)),
				h.Button(
					h.Type("submit"),
					h.Class("bg-red-500 hover:bg-red-600 text-white px-3 py-1.5 rounded text-sm font-medium"),
					g.Text("Выйти"),
				),
			),
		)
	} else {
		authSection = h.A(
			h.Href("/auth/github"),
			h.Class("bg-gray-900 hover:bg-gray-800 text-white px-4 py-2 rounded text-sm font-medium flex items-center space-x-2"),
			g.Text("Войти через GitHub"),
		)
	}

	return h.Header(
		h.Class("bg-white border-b border-gray-200 shadow-sm"),
		h.Div(
			h.Class("container mx-auto px-4 py-4 flex items-center justify-between"),
			h.A(
				h.Href("/"),
				h.Class("text-2xl font-bold text-indigo-600 tracking-tight"),
				g.Text("Shreelance"),
			),
			h.Nav(
				h.Class("flex items-center space-x-6"),
				h.A(h.Href("/orders"), h.Class("text-gray-600 hover:text-indigo-600 font-medium"), g.Text("Заказы")),
				authSection,
			),
		),
	)
}

func Footer() g.Node {
	return h.Footer(
		h.Class("bg-white border-t border-gray-200 py-6 mt-12"),
		h.Div(
			h.Class("container mx-auto px-4 text-center text-gray-500 text-sm"),
			g.Text("© 2026 Shreelance. Все права защищены."),
		),
	)
}
