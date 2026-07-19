package ui

import (
	"shreelance/internal/models"

	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

func Navbar(p PageParams) g.Node {
	var authSection g.Node
	hasUser := false
	if p.User != nil {
		if u, ok := p.User.(*models.User); ok && u != nil && u.ID > 0 {
			hasUser = true
		}
	}

	themeVal := p.Theme
	if themeVal == "" {
		themeVal = "system"
	}

	var themeLabel string
	switch themeVal {
	case "light":
		themeLabel = "☀️ Светлая"
	case "dark":
		themeLabel = "🌙 Темная"
	default:
		themeLabel = "💻 Системная"
	}

	themeSelector := h.Div(
		h.Class("relative"),
		g.Attr("x-data", `{
			open: false,
			theme: '`+themeVal+`',
			setTheme(val) {
				this.theme = val;
				// Save in localStorage and cookie
				localStorage.setItem('theme', val);
				document.cookie = "theme=" + val + "; path=/; max-age=31536000; SameSite=Lax";
				if (val === 'dark' || (val === 'system' && window.matchMedia('(prefers-color-scheme: dark)').matches)) {
					document.documentElement.classList.add('dark');
				} else {
					document.documentElement.classList.remove('dark');
				}
				this.open = false;
			}
		}`),
		h.Button(
			h.Class("bg-panel-bg dark:bg-panel-bg-dark border border-panel-border dark:border-panel-border-dark rounded-xl px-3 py-1.5 text-sm font-medium text-app-text dark:text-app-text-dark hover:bg-app-bg dark:hover:bg-zinc-800 focus:outline-none flex items-center space-x-1 min-w-[125px] justify-center transition-colors overflow-hidden"),
			g.Attr("@click", "open = !open"),
			g.Attr("aria-label", "Toggle theme"),
			// Render dynamic text on the server to prevent UI shifting and loading lag
			h.Span(
				g.Attr("x-text", "theme === 'light' ? '☀️ Светлая' : (theme === 'dark' ? '🌙 Темная' : '💻 Системная')"),
				g.Text(themeLabel),
			),
		),
		h.Div(
			h.Class("origin-top-right absolute right-0 mt-2 w-40 rounded-xl shadow-lg bg-panel-bg dark:bg-panel-bg-dark ring-1 ring-panel-border dark:ring-panel-border-dark z-50 overflow-hidden"),
			g.Attr("x-show", "open"),
			g.Attr("x-cloak", ""),
			g.Attr("@click.away", "open = false"),
			g.Attr("x-transition", ""),
			h.Div(
				h.Class("py-1"),
				h.Button(
					h.Class("block w-full text-left px-4 py-2 text-sm text-app-text dark:text-app-text-dark hover:bg-app-bg dark:hover:bg-zinc-800"),
					g.Attr("@click", "setTheme('light')"),
					g.Text("☀️ Светлая"),
				),
				h.Button(
					h.Class("block w-full text-left px-4 py-2 text-sm text-app-text dark:text-app-text-dark hover:bg-app-bg dark:hover:bg-zinc-800"),
					g.Attr("@click", "setTheme('dark')"),
					g.Text("🌙 Темная"),
				),
				h.Button(
					h.Class("block w-full text-left px-4 py-2 text-sm text-app-text dark:text-app-text-dark hover:bg-app-bg dark:hover:bg-zinc-800"),
					g.Attr("@click", "setTheme('system')"),
					g.Text("💻 Системная"),
				),
			),
		),
	)

	if hasUser {
		authSection = h.Div(
			h.Class("flex items-center space-x-4"),
			themeSelector,
			h.Div(
				h.Class("relative"),
				g.Attr("x-data", "{ open: false }"),
				h.Button(
					h.Class("bg-panel-bg dark:bg-panel-bg-dark border border-panel-border dark:border-panel-border-dark rounded-xl px-3 py-1.5 text-sm font-medium text-app-text dark:text-app-text-dark hover:bg-app-bg dark:hover:bg-zinc-800 focus:outline-none transition-colors overflow-hidden"),
					g.Attr("@click", "open = !open"),
					g.Text("Режим: "+map[string]string{"customer": "Заказчик", "freelancer": "Исполнитель"}[p.ContextRole]),
				),
				h.Div(
					h.Class("origin-top-right absolute right-0 mt-2 w-48 rounded-xl shadow-lg bg-panel-bg dark:bg-panel-bg-dark ring-1 ring-panel-border dark:ring-panel-border-dark z-50 overflow-hidden"),
					g.Attr("x-show", "open"),
					g.Attr("x-cloak", ""),
					g.Attr("@click.away", "open = false"),
					g.Attr("x-transition", ""),
					h.Div(
						h.Class("py-1"),
						h.Form(
							h.Action("/profile/role"),
							h.Method("POST"),
							h.Input(h.Type("hidden"), h.Name("csrf_token"), h.Value(p.CSRFToken)),
							h.Input(h.Type("hidden"), h.Name("role"), h.Value("customer")),
							h.Button(h.Type("submit"), h.Class("block w-full text-left px-4 py-2 text-sm text-app-text dark:text-app-text-dark hover:bg-app-bg dark:hover:bg-zinc-800"), g.Text("Заказчик")),
						),
						h.Form(
							h.Action("/profile/role"),
							h.Method("POST"),
							h.Input(h.Type("hidden"), h.Name("csrf_token"), h.Value(p.CSRFToken)),
							h.Input(h.Type("hidden"), h.Name("role"), h.Value("freelancer")),
							h.Button(h.Type("submit"), h.Class("block w-full text-left px-4 py-2 text-sm text-app-text dark:text-app-text-dark hover:bg-app-bg dark:hover:bg-zinc-800"), g.Text("Исполнитель")),
						),
					),
				),
			),
			h.A(
				h.Href("/profile"),
				h.Class("text-sm font-semibold hover:text-brand-primary dark:hover:text-brand-primary-dark text-app-text dark:text-app-text-dark"),
				g.Text("Мой Профиль"),
			),
			h.Form(
				h.Action("/auth/logout"),
				h.Method("POST"),
				h.Class("inline"),
				h.Input(h.Type("hidden"), h.Name("csrf_token"), h.Value(p.CSRFToken)),
				h.Button(
					h.Type("submit"),
					h.Class("bg-red-500 hover:bg-red-600 dark:bg-red-600 dark:hover:bg-red-700 text-white px-3 py-1.5 rounded-xl text-sm font-medium"),
					g.Text("Выйти"),
				),
			),
		)
	} else {
		authSection = h.Div(
			h.Class("flex items-center space-x-4"),
			themeSelector,
			h.A(
				h.Href("/auth/github"),
				h.Class("bg-brand-primary dark:bg-brand-primary-dark hover:opacity-90 text-white dark:text-btn-text-dark px-4 py-2 rounded-xl text-sm font-medium flex items-center space-x-2"),
				g.Text("Войти через GitHub"),
			),
		)
	}

	return h.Header(
		h.Class("bg-panel-bg dark:bg-panel-bg-dark border-b border-panel-border dark:border-panel-border-dark shadow-sm"),
		h.Div(
			h.Class("container mx-auto px-4 py-4 flex items-center justify-between"),
			h.A(
				h.Href("/"),
				h.Class("text-2xl font-bold text-brand-primary dark:text-brand-primary-dark tracking-tight"),
				g.Text("Shreelance"),
			),
			h.Nav(
				h.Class("flex items-center space-x-6"),
				h.A(h.Href("/orders"), h.Class("text-app-text-muted dark:text-app-text-muted-dark hover:text-brand-primary dark:hover:text-brand-primary-dark font-medium"), g.Text("Заказы")),
				authSection,
			),
		),
	)
}
