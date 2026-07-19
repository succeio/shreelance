package ui

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

func RegisterPage(csrfToken string, errorMsg string) g.Node {
	return h.Div(
		h.Class("max-w-md mx-auto bg-panel-bg dark:bg-panel-bg-dark p-8 rounded-2xl shadow-md border border-panel-border dark:border-panel-border-dark"),
		h.H1(h.Class("text-2xl font-bold text-center text-app-text dark:text-headline-dark mb-6"), g.Text("Регистрация")),

		g.If(errorMsg != "", h.Div(
			h.Class("bg-red-100 dark:bg-red-900/40 border border-red-400 dark:border-red-600 text-red-700 dark:text-red-300 px-4 py-3 rounded-xl mb-4 text-sm"),
			g.Text(errorMsg),
		)),

		// OAuth Login Options (GitHub & GitLab)
		h.Div(
			h.Class("space-y-3 mb-6"),
			h.A(
				h.Href("/auth/github"),
				h.Class("w-full flex items-center justify-center space-x-2 bg-gray-900 hover:bg-gray-800 text-white font-medium py-2.5 rounded-xl transition-colors text-sm shadow-sm"),
				g.Text("Продолжить через GitHub"),
			),
			h.A(
				h.Href("/auth/gitlab"),
				h.Class("w-full flex items-center justify-center space-x-2 bg-orange-600 hover:bg-orange-700 text-white font-medium py-2.5 rounded-xl transition-colors text-sm shadow-sm"),
				g.Text("Продолжить через GitLab"),
			),
		),

		h.Div(
			h.Class("relative flex py-2 items-center mb-6"),
			h.Div(h.Class("flex-grow border-t border-panel-border dark:border-panel-border-dark")),
			h.Span(h.Class("flex-shrink mx-4 text-xs uppercase tracking-wider text-app-text-muted dark:text-app-text-muted-dark"), g.Text("или по почте")),
			h.Div(h.Class("flex-grow border-t border-panel-border dark:border-panel-border-dark")),
		),

		h.Form(
			h.Action("/register"),
			h.Method("POST"),
			h.Class("space-y-4"),
			h.Input(h.Type("hidden"), h.Name("csrf_token"), h.Value(csrfToken)),

			h.Div(
				h.Label(h.Class("block text-sm font-semibold text-app-text dark:text-headline-dark mb-1"), g.Text("Имя пользователя / Логин")),
				h.Input(
					h.Type("text"),
					h.Name("username"),
					h.Required(),
					h.Placeholder("john_doe"),
					h.Class("w-full border border-panel-border dark:border-panel-border-dark bg-app-bg dark:bg-app-bg-dark text-app-text dark:text-app-text-dark rounded-xl px-3 py-2 focus:ring-brand-primary focus:border-brand-primary text-sm"),
				),
			),

			h.Div(
				h.Label(h.Class("block text-sm font-semibold text-app-text dark:text-headline-dark mb-1"), g.Text("Email")),
				h.Input(
					h.Type("email"),
					h.Name("email"),
					h.Required(),
					h.Placeholder("user@example.com"),
					h.Class("w-full border border-panel-border dark:border-panel-border-dark bg-app-bg dark:bg-app-bg-dark text-app-text dark:text-app-text-dark rounded-xl px-3 py-2 focus:ring-brand-primary focus:border-brand-primary text-sm"),
				),
			),

			h.Div(
				h.Label(h.Class("block text-sm font-semibold text-app-text dark:text-headline-dark mb-1"), g.Text("Пароль")),
				h.Input(
					h.Type("password"),
					h.Name("password"),
					h.Required(),
					h.Min("6"),
					h.Placeholder("••••••••"),
					h.Class("w-full border border-panel-border dark:border-panel-border-dark bg-app-bg dark:bg-app-bg-dark text-app-text dark:text-app-text-dark rounded-xl px-3 py-2 focus:ring-brand-primary focus:border-brand-primary text-sm"),
				),
			),

			h.Button(
				h.Type("submit"),
				h.Class("w-full bg-brand-primary dark:bg-brand-primary-dark hover:opacity-90 text-white dark:text-btn-text-dark font-semibold py-2.5 rounded-xl transition-colors text-sm mt-2"),
				g.Text("Зарегистрироваться"),
			),
		),

		h.P(
			h.Class("text-xs text-center text-app-text-muted dark:text-app-text-muted-dark mt-6"),
			g.Text("Уже есть аккаунт? "),
			h.A(h.Href("/login"), h.Class("text-brand-primary dark:text-brand-primary-dark hover:underline font-semibold"), g.Text("Войти")),
		),
	)
}

func LoginPage(csrfToken string, errorMsg string) g.Node {
	return h.Div(
		h.Class("max-w-md mx-auto bg-panel-bg dark:bg-panel-bg-dark p-8 rounded-2xl shadow-md border border-panel-border dark:border-panel-border-dark"),
		h.H1(h.Class("text-2xl font-bold text-center text-app-text dark:text-headline-dark mb-6"), g.Text("Вход в аккаунт")),

		g.If(errorMsg != "", h.Div(
			h.Class("bg-red-100 dark:bg-red-900/40 border border-red-400 dark:border-red-600 text-red-700 dark:text-red-300 px-4 py-3 rounded-xl mb-4 text-sm"),
			g.Text(errorMsg),
		)),

		// OAuth Login Options (GitHub & GitLab side-by-side / stack)
		h.Div(
			h.Class("space-y-3 mb-6"),
			h.A(
				h.Href("/auth/github"),
				h.Class("w-full flex items-center justify-center space-x-2 bg-gray-900 hover:bg-gray-800 text-white font-medium py-2.5 rounded-xl transition-colors text-sm shadow-sm"),
				g.Text("Войти через GitHub"),
			),
			h.A(
				h.Href("/auth/gitlab"),
				h.Class("w-full flex items-center justify-center space-x-2 bg-orange-600 hover:bg-orange-700 text-white font-medium py-2.5 rounded-xl transition-colors text-sm shadow-sm"),
				g.Text("Войти через GitLab"),
			),
		),

		h.Div(
			h.Class("relative flex py-2 items-center mb-6"),
			h.Div(h.Class("flex-grow border-t border-panel-border dark:border-panel-border-dark")),
			h.Span(h.Class("flex-shrink mx-4 text-xs uppercase tracking-wider text-app-text-muted dark:text-app-text-muted-dark"), g.Text("или по почте")),
			h.Div(h.Class("flex-grow border-t border-panel-border dark:border-panel-border-dark")),
		),

		h.Form(
			h.Action("/login"),
			h.Method("POST"),
			h.Class("space-y-4"),
			h.Input(h.Type("hidden"), h.Name("csrf_token"), h.Value(csrfToken)),

			h.Div(
				h.Label(h.Class("block text-sm font-semibold text-app-text dark:text-headline-dark mb-1"), g.Text("Email")),
				h.Input(
					h.Type("email"),
					h.Name("email"),
					h.Required(),
					h.Placeholder("user@example.com"),
					h.Class("w-full border border-panel-border dark:border-panel-border-dark bg-app-bg dark:bg-app-bg-dark text-app-text dark:text-app-text-dark rounded-xl px-3 py-2 focus:ring-brand-primary focus:border-brand-primary text-sm"),
				),
			),

			h.Div(
				h.Label(h.Class("block text-sm font-semibold text-app-text dark:text-headline-dark mb-1"), g.Text("Пароль")),
				h.Input(
					h.Type("password"),
					h.Name("password"),
					h.Required(),
					h.Placeholder("••••••••"),
					h.Class("w-full border border-panel-border dark:border-panel-border-dark bg-app-bg dark:bg-app-bg-dark text-app-text dark:text-app-text-dark rounded-xl px-3 py-2 focus:ring-brand-primary focus:border-brand-primary text-sm"),
				),
			),

			h.Button(
				h.Type("submit"),
				h.Class("w-full bg-brand-primary dark:bg-brand-primary-dark hover:opacity-90 text-white dark:text-btn-text-dark font-semibold py-2.5 rounded-xl transition-colors text-sm mt-2"),
				g.Text("Войти"),
			),
		),

		h.P(
			h.Class("text-xs text-center text-app-text-muted dark:text-app-text-muted-dark mt-6"),
			g.Text("Ещё нет аккаунта? "),
			h.A(h.Href("/register"), h.Class("text-brand-primary dark:text-brand-primary-dark hover:underline font-semibold"), g.Text("Зарегистрироваться")),
		),
	)
}
