package ui

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

type PageParams struct {
	Title       string
	Content     g.Node
	User        interface{} // Pass user if logged in
	CSRFToken   string
	ContextRole string // "customer" or "freelancer"
	Theme       string // "light", "dark", "system"
}

func Layout(p PageParams) g.Node {
	return h.HTML(
		h.Lang("ru"),
		h.Head(
			h.Meta(h.Charset("UTF-8")),
			h.Meta(h.Name("viewport"), h.Content("width=device-width, initial-scale=1.0")),
			h.Title(p.Title+" - Shreelance"),
			// Inline script to prevent theme flash (FOUC)
			h.Script(g.Raw(`
				(function() {
					// Read theme from cookies first (set by server) or localStorage, defaulting to 'system'
					const getCookie = (name) => {
						const value = "; " + document.cookie;
						const parts = value.split("; " + name + "=");
						if (parts.length === 2) return parts.pop().split(";").shift();
					};
					const theme = getCookie('theme') || localStorage.getItem('theme') || 'system';
					if (theme === 'dark' || (theme === 'system' && window.matchMedia('(prefers-color-scheme: dark)').matches)) {
						document.documentElement.classList.add('dark');
					} else {
						document.documentElement.classList.remove('dark');
					}
				})();
			`)),
			// Tailwind CSS
			h.Link(h.Rel("stylesheet"), h.Href("/static/style.css")),
			// HTMX v2
			h.Script(h.Src("https://unpkg.com/htmx.org@2.0.0")),
			// Alpine.js CDN
			h.Script(h.Defer(), h.Src("https://unpkg.com/alpinejs@3.x.x/dist/cdn.min.js")),
		),
		h.Body(
			h.Class("bg-app-bg text-app-text dark:bg-app-bg-dark dark:text-app-text-dark min-h-screen flex flex-col"),
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
