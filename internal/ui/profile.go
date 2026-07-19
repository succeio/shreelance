package ui

import (
	"fmt"
	"strconv"
	"strings"

	g "maragu.dev/gomponents"
	html "maragu.dev/gomponents/html"
	"shreelance/internal/models"
)

func ProfilePage(u *models.User, role string, csrfToken string, errorMsg string) g.Node {
	var specialistSection g.Node
	if u.GitHubID != nil || u.GitLabID != nil {
		specialistSection = html.Div(
			html.Class("border-t border-panel-border dark:border-panel-border-dark pt-6 mt-6"),
			html.H2(html.Class("text-xl font-semibold mb-4 text-app-text dark:text-headline-dark"), g.Text("Настройки профиля специалиста")),
			
			// GitHub & GitLab Sync Button
			html.Div(
				html.Class("mb-6 space-y-4"),
				html.Form(
					html.Action("/profile/sync"),
					html.Method("POST"),
					html.Class("bg-app-bg dark:bg-app-bg-dark p-4 rounded-2xl border border-panel-border dark:border-panel-border-dark flex items-center justify-between"),
					html.Input(html.Type("hidden"), html.Name("csrf_token"), html.Value(csrfToken)),
					html.Div(
						html.P(html.Class("text-sm font-semibold text-app-text dark:text-headline-dark"), g.Text("Импорт профиля")),
						html.P(html.Class("text-xs text-app-text-muted dark:text-app-text-muted-dark"), g.Text("Автоматически заполнить стек технологиями из репозиториев (GitHub и GitLab).")),
					),
					html.Button(
						html.Type("submit"),
						html.Class("bg-brand-primary dark:bg-brand-primary-dark hover:opacity-90 text-white dark:text-btn-text-dark font-medium text-xs py-2 px-4 rounded transition-colors flex items-center space-x-1.5"),
						g.Text("Синхронизировать"),
					),
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
	} else {
		specialistSection = html.Div(
			html.Class("border-t border-panel-border dark:border-panel-border-dark pt-6 mt-6"),
			html.Div(
				html.Class("bg-app-bg dark:bg-app-bg-dark p-4 rounded-2xl border border-panel-border dark:border-panel-border-dark flex items-center justify-between"),
				html.Div(
					html.P(html.Class("text-sm font-semibold text-app-text dark:text-headline-dark"), g.Text("Хотите откликаться на заказы в качестве исполнителя?")),
					html.P(html.Class("text-xs text-app-text-muted dark:text-app-text-muted-dark"), g.Text("Авторизуйтесь или привяжите аккаунт GitHub или GitLab для подтверждения компетенций.")),
				),
				html.Div(
					html.Class("flex space-x-2"),
					html.A(
						html.Href("/auth/github"),
						html.Class("bg-gray-900 hover:bg-gray-800 text-white font-medium text-xs py-2 px-3 rounded-xl transition-colors"),
						g.Text("GitHub"),
					),
					html.A(
						html.Href("/auth/gitlab"),
						html.Class("bg-orange-600 hover:bg-orange-700 text-white font-medium text-xs py-2 px-3 rounded-xl transition-colors"),
						g.Text("GitLab"),
					),
				),
			),
		)
	}

	return html.Div(
		html.Class("max-w-2xl mx-auto bg-panel-bg dark:bg-panel-bg-dark p-8 rounded-2xl shadow-md border border-panel-border dark:border-panel-border-dark"),
		
		g.If(errorMsg != "", html.Div(
			html.Class("bg-red-100 dark:bg-red-900/40 border border-red-400 dark:border-red-600 text-red-700 dark:text-red-300 px-4 py-3 rounded-xl mb-4 text-sm"),
			g.Text(func() string {
				if errorMsg == "star_not_found" {
					return "Вы не поставили звезду репозиторию. Пожалуйста, поставьте звезду и попробуйте снова!"
				}
				return errorMsg
			}()),
		)),

		// Add Reward Task Banner
		g.If(!u.HasStarredRepo, html.Div(
			html.Class("mb-6 p-5 rounded-2xl bg-gradient-to-r from-purple-500/20 via-pink-500/10 to-indigo-500/20 dark:from-purple-950/40 dark:to-indigo-950/40 border border-purple-500/30 dark:border-purple-500/20 shadow-md animate-pulse-slow"),
			html.Div(
				html.Class("flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4"),
				html.Div(
					html.Class("space-y-1"),
					html.H3(html.Class("text-base font-bold text-purple-900 dark:text-purple-300 flex items-center space-x-1.5"),
						g.Raw(`<svg class="w-5 h-5 text-amber-500 animate-bounce" fill="currentColor" viewBox="0 0 20 20"><path d="M9.049 2.927c.3-.921 1.603-.921 1.902 0l1.07 3.292a1 1 0 00.95.69h3.462c.969 0 1.371 1.24.588 1.81l-2.8 2.034a1 1 0 00-.364 1.118l1.07 3.292c.3.921-.755 1.688-1.54 1.118l-2.8-2.034a1 1 0 00-1.175 0l-2.8 2.034c-.784.57-1.838-.197-1.539-1.118l1.07-3.292a1 1 0 00-.364-1.118L2.98 8.72c-.783-.57-.38-1.81.588-1.81h3.461a1 1 0 00.951-.69l1.07-3.292z"></path></svg>`),
						g.Text("Получи PRO аккаунт бесплатно на 3 дня!"),
					),
					html.H3(html.Class("text-xs text-purple-750 dark:text-purple-400 font-semibold"),
						g.Text(fmt.Sprintf("Ваш PRO_ID для доната: PRO_ID_%d", u.ID)),
					),
					html.P(html.Class("text-xs text-purple-750 dark:text-purple-400 leading-relaxed"),
						g.Text("Поставьте звезду нашему проекту на GitHub или GitLab, а затем нажмите кнопку проверки, чтобы активировать PRO-статус."),
					),
					html.Div(
						html.Class("flex flex-wrap gap-3 mt-2 text-xs font-semibold"),
						g.If(u.GitHubID != nil, html.A(
							html.Href("https://github.com/succeio/shreelance"),
							html.Target("_blank"),
							html.Class("text-blue-600 dark:text-blue-400 hover:underline flex items-center space-x-1"),
							g.Text("★ GitHub репозиторий"),
						)),
						g.If(u.GitLabID != nil || u.GitLabUsername != "", html.A(
							html.Href("https://gitlab.com/blackteka/hikkasay"),
							html.Target("_blank"),
							html.Class("text-orange-600 dark:text-orange-400 hover:underline flex items-center space-x-1"),
							g.Text("★ GitLab репозиторий"),
						)),
					),
				),
				html.Form(
					html.Action("/profile/verify-star"),
					html.Method("POST"),
					html.Input(html.Type("hidden"), html.Name("csrf_token"), html.Value(csrfToken)),
					html.Button(
						html.Type("submit"),
						html.Class("w-full sm:w-auto bg-purple-600 dark:bg-purple-700 hover:bg-purple-700 dark:hover:bg-purple-800 text-white font-bold text-xs py-2.5 px-4 rounded-xl shadow-lg hover:shadow-purple-500/20 transition-all flex items-center justify-center space-x-1.5"),
						g.Text("Проверить звезду"),
					),
				),
			),
		)),

		// DonationAlerts Section
		html.Div(
			html.Class("mb-6 p-5 rounded-2xl bg-gradient-to-r from-orange-500/10 via-yellow-500/5 to-orange-500/10 dark:from-orange-950/20 dark:to-yellow-950/20 border border-orange-500/20 dark:border-orange-500/10 shadow-sm"),
			html.Div(
				html.Class("flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4"),
				html.Div(
					html.Class("space-y-1"),
					html.H3(html.Class("text-base font-bold text-orange-900 dark:text-orange-300 flex items-center space-x-1.5"),
						g.Text("Поддержка проекта (Купить PRO)"),
					),
					html.P(html.Class("text-xs text-orange-750 dark:text-orange-400 leading-relaxed"),
						g.Text("Вы можете продлить PRO-статус через пожертвование (100 рублей = 10 дней PRO)."),
					),
					html.P(html.Class("text-xs font-semibold text-orange-900 dark:text-orange-300"),
						g.Text(fmt.Sprintf("Обязательно укажите в сообщении доната: PRO_ID_%d", u.ID)),
					),
				),
				html.A(
					html.Href(fmt.Sprintf("https://www.donationalerts.com/r/shrelance?amount=100&comment=PRO_ID_%d", u.ID)),
					html.Target("_blank"),
					html.Class("w-full sm:w-auto bg-orange-500 hover:bg-orange-650 text-white font-bold text-xs py-2.5 px-4 rounded-xl shadow-md transition-all text-center"),
					g.Text("Пожертвовать"),
				),
			),
		),

		html.Div(
			html.Class("flex items-center space-x-6 mb-8"),
			html.Img(html.Src(u.AvatarURL), html.Alt(u.Username), html.Class("w-24 h-24 rounded-full border-4 border-panel-border dark:border-panel-border-dark")),
			html.Div(
				html.H1(html.Class("text-3xl font-bold text-app-text dark:text-headline-dark flex items-center space-x-3"),
					html.Span(g.Text(u.Username)),
					g.If(u.IsPro(), html.Span(
						html.Class("bg-gradient-to-r from-amber-500 to-yellow-400 text-white font-extrabold text-xs px-2.5 py-0.5 rounded-full shadow-sm tracking-wide border border-amber-300 dark:border-amber-600"),
						html.Title("PRO аккаунт активен"),
						g.Text("PRO"),
					)),
					// Render GitHub icon if primary account is GitHub (GitHubID != nil), otherwise render GitLab icon if GitLabID != nil
					g.If(u.GitHubID != nil, html.A(
						html.Href("https://github.com/"+u.Username),
						html.Target("_blank"),
						html.Class("text-app-text-muted hover:text-brand-primary transition-colors"),
						g.Raw(`<svg class="w-6 h-6 fill-current" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg"><path d="M12 .297c-6.63 0-12 5.373-12 12 0 5.303 3.438 9.8 8.205 11.385.6.113.82-.258.82-.577 0-.285-.01-1.04-.015-2.04-3.338.724-4.042-1.61-4.042-1.61C4.422 18.07 3.633 17.7 3.633 17.7c-1.087-.744.084-.729.084-.729 1.205.084 1.838 1.236 1.838 1.236 1.07 1.835 2.809 1.305 3.495.998.108-.776.417-1.305.76-1.605-2.665-.3-5.466-1.332-5.466-5.93 0-1.31.465-2.38 1.235-3.22-.135-.303-.54-1.523.105-3.176 0 0 1.005-.322 3.3 1.23.96-.267 1.98-.399 3-.405 1.02.006 2.04.138 3 .405 2.28-1.552 3.285-1.23 3.285-1.23.645 1.653.24 2.873.12 3.176.765.84 1.23 1.91 1.23 3.22 0 4.61-2.805 5.625-5.475 5.92.42.36.81 1.096.81 2.22 0 1.606-.015 2.896-.015 3.286 0 .315.21.69.825.57C20.565 22.092 24 17.592 24 12.297c0-6.627-5.373-12-12-12"/></svg>`),
					)),
					g.If(u.GitHubID == nil && u.GitLabID != nil, html.A(
						html.Href(func() string {
							if u.GitLabUsername != "" {
								return "https://gitlab.com/" + u.GitLabUsername
							}
							return "https://gitlab.com/" + u.Username
						}()),
						html.Target("_blank"),
						html.Class("text-[#FC6D26] hover:opacity-80 transition-opacity"),
						g.Raw(`<svg class="w-6 h-6 fill-current" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg"><path d="M23.953 11.83l-1.637-5.037a.625.625 0 0 0-.239-.33.61.61 0 0 0-.398-.088.618.618 0 0 0-.356.173L12 15.385l-9.324-8.837a.622.622 0 0 0-.355-.173.613.613 0 0 0-.4.089.625.625 0 0 0-.238.329L.047 11.83a1.009 1.009 0 0 0 .34 1.127l11.168 8.113a.774.774 0 0 0 .888 0l11.169-8.113a1.009 1.009 0 0 0 .341-1.127z"/></svg>`),
					)),
				),
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
		
		// Developer Activity Section (GitHub & GitLab)
		g.If(u.GitHubID != nil, html.Div(
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
		)),

		g.If(u.GitLabID != nil, html.Div(
			html.Class("border-t border-panel-border dark:border-panel-border-dark pt-6 mt-6"),
			html.H2(html.Class("text-xl font-semibold mb-4 text-app-text dark:text-headline-dark flex items-center space-x-2"),
				html.Span(g.Text("Активность на GitLab")),
				html.Span(html.Class("text-xs font-medium px-2 py-0.5 rounded-full bg-orange-100 dark:bg-orange-900/40 text-orange-600 dark:text-orange-300 border border-orange-200 dark:border-orange-800"), g.Text("GitLab")),
			),
			html.P(html.Class("text-xs text-app-text-muted dark:text-app-text-muted-dark mb-4"), g.Text("Статистика публичных проектов и репозиториев GitLab")),
			html.Div(
				html.Class("bg-app-bg dark:bg-app-bg-dark p-4 rounded-2xl border border-panel-border dark:border-panel-border-dark flex flex-col items-center justify-center space-y-4 overflow-hidden"),
				html.Img(
					html.Src("/profile/gitlab-card.svg?username="+(func() string {
						if u.GitLabUsername != "" {
							return u.GitLabUsername
						}
						return u.Username
					})()),
					html.Alt(u.GitLabUsername+"'s GitLab Stats"),
					html.Class("w-full h-auto min-h-[45px] object-cover rounded-xl dark:invert dark:hue-rotate-180 dark:brightness-75"),
				),
			),
		)),
	)
}
