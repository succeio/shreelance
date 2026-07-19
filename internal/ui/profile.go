package ui

import (
	"strconv"
	"strings"

	g "maragu.dev/gomponents"
	html "maragu.dev/gomponents/html"
	"shreelance/internal/models"
)

func ProfilePage(u *models.User, role string, csrfToken string) g.Node {
	var specialistSection g.Node
	if role == "freelancer" {
		specialistSection = html.Div(
			html.Class("border-t border-panel-border dark:border-panel-border-dark pt-6 mt-6"),
			html.H2(html.Class("text-xl font-semibold mb-4 text-app-text dark:text-headline-dark"), g.Text("Настройки профиля специалиста")),
			
			// GitHub Sync Button
			html.Form(
				html.Action("/profile/sync"),
				html.Method("POST"),
				html.Class("mb-6 bg-app-bg dark:bg-app-bg-dark p-4 rounded-2xl border border-panel-border dark:border-panel-border-dark flex items-center justify-between"),
				html.Input(html.Type("hidden"), html.Name("csrf_token"), html.Value(csrfToken)),
				html.Div(
					html.P(html.Class("text-sm font-semibold text-app-text dark:text-headline-dark"), g.Text("Импорт профиля с GitHub")),
					html.P(html.Class("text-xs text-app-text-muted dark:text-app-text-muted-dark"), g.Text("Автоматически заполнить стек технологиями из репозиториев и рассчитать опыт на основе даты создания аккаунта GitHub.")),
				),
				html.Button(
					html.Type("submit"),
					html.Class("bg-brand-primary dark:bg-brand-primary-dark hover:opacity-90 text-white dark:text-btn-text-dark font-medium text-xs py-2 px-4 rounded transition-colors flex items-center space-x-1.5"),
					g.Text("Синхронизировать"),
				),
			),

			// Edit Profile Form
			html.Form(
				html.Action("/profile/update"),
				html.Method("POST"),
				html.Class("space-y-4"),
				html.Input(html.Type("hidden"), html.Name("csrf_token"), html.Value(csrfToken)),
				html.Div(
					html.Label(html.Class("block text-sm font-semibold text-app-text dark:text-headline-dark mb-1"), g.Text("Технологический стек")),
					html.Input(
						html.Type("text"),
						html.Name("stack"),
						html.Value(u.Stack),
						html.Placeholder("Например: Go, TypeScript, React, PostgreSQL"),
						html.Class("w-full border border-panel-border dark:border-panel-border-dark bg-app-bg dark:bg-app-bg-dark text-app-text dark:text-app-text-dark rounded px-3 py-2 focus:ring-brand-primary dark:focus:ring-brand-primary-dark focus:border-brand-primary dark:focus:border-brand-primary-dark"),
					),
					html.P(html.Class("text-xs text-app-text-muted dark:text-app-text-muted-dark mt-1"), g.Text("Перечислите технологии через запятую")),
				),
				html.Div(
					html.Label(html.Class("block text-sm font-semibold text-app-text dark:text-headline-dark mb-1"), g.Text("Опыт работы (лет)")),
					html.Input(
						html.Type("number"),
						html.Name("experience_years"),
						html.Value(strconv.Itoa(u.ExperienceYears)),
						html.Min("0"),
						html.Class("w-full border border-panel-border dark:border-panel-border-dark bg-app-bg dark:bg-app-bg-dark text-app-text dark:text-app-text-dark rounded px-3 py-2 focus:ring-brand-primary dark:focus:ring-brand-primary-dark focus:border-brand-primary dark:focus:border-brand-primary-dark"),
					),
				),
				html.Button(
					html.Type("submit"),
					html.Class("w-full bg-brand-primary dark:bg-brand-primary-dark hover:opacity-90 text-white dark:text-btn-text-dark font-semibold py-2.5 rounded transition-colors"),
					g.Text("Сохранить изменения"),
				),
			),
		)
	}

	return html.Div(
		html.Class("max-w-2xl mx-auto bg-panel-bg dark:bg-panel-bg-dark p-8 rounded-2xl shadow-md border border-panel-border dark:border-panel-border-dark"),
		html.Div(
			html.Class("flex items-center space-x-6 mb-8"),
			html.Img(html.Src(u.AvatarURL), html.Alt(u.Username), html.Class("w-24 h-24 rounded-full border-4 border-panel-border dark:border-panel-border-dark")),
			html.Div(
				html.H1(html.Class("text-3xl font-bold text-app-text dark:text-headline-dark"), g.Text(u.Username)),
				html.P(html.Class("text-sm text-app-text-muted dark:text-app-text-muted-dark"), g.Text(u.Email)),
				g.If(u.Stack != "", html.Div(
					html.Class("mt-2 flex flex-wrap gap-1.5"),
					g.Group(func() []g.Node {
						var tags []g.Node
						for _, s := range strings.Split(u.Stack, ",") {
							trimmed := strings.TrimSpace(s)
							if trimmed != "" {
								tags = append(tags, TechBadge(trimmed))
							}
						}
						return tags
					}()),
				)),
			),
		),
		html.Div(
			html.Class("border-t border-panel-border dark:border-panel-border-dark pt-6"),
			html.H2(html.Class("text-xl font-semibold mb-4 text-app-text dark:text-headline-dark"), g.Text("Текущий контекст интерфейса")),
			html.P(html.Class("text-app-text-muted dark:text-app-text-muted-dark mb-4 text-sm"), g.Text("Вы можете свободно переключаться между ролями Заказчика и Исполнителя.")),
			html.Div(
				html.Class("bg-app-bg dark:bg-app-bg-dark border border-panel-border dark:border-panel-border-dark rounded-2xl p-4 flex items-center justify-between"),
				html.Div(
					html.P(html.Class("text-xs text-brand-primary dark:text-brand-primary-dark font-semibold uppercase tracking-wider"), g.Text("Активная роль")),
					html.P(html.Class("text-lg font-bold text-app-text dark:text-headline-dark mt-0.5"), g.Text(map[string]string{"customer": "Заказчик (Публикация заданий)", "freelancer": "Исполнитель (Отклики на задания)"}[role])),
				),
			),
		),
		specialistSection,
		
		// GitHub Activity Contribution Grid
		html.Div(
			html.Class("border-t border-panel-border dark:border-panel-border-dark pt-6 mt-6"),
			html.H2(html.Class("text-xl font-semibold mb-4 text-app-text dark:text-headline-dark"), g.Text("Активность на GitHub")),
			html.P(html.Class("text-xs text-app-text-muted dark:text-app-text-muted-dark mb-4"), g.Text("История вкладов (commits, pull requests, issues) за последний год")),
			html.Div(
				html.Class("bg-app-bg dark:bg-app-bg-dark p-4 rounded-2xl border border-panel-border dark:border-panel-border-dark flex justify-center overflow-hidden"),
				html.Img(
					html.Src("https://ghchart.rshah.org/4f46e5/"+u.Username),
					html.Alt(u.Username+"'s GitHub Contributions Chart"),
					html.Class("w-full h-auto object-contain dark:invert dark:hue-rotate-180 dark:brightness-75"),
				),
			),
		),
	)
}
