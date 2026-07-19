package ui

import (
	"fmt"
	"strings"

	"shreelance/internal/models"

	g "maragu.dev/gomponents"
	html "maragu.dev/gomponents/html"
)

func SpecialistsDashboard(specialists []models.User, search, tech, minExp, sort string) g.Node {
	var specCards []g.Node
	for _, s := range specialists {
		specCards = append(specCards, html.Div(
			html.Class("bg-panel-bg dark:bg-panel-bg-dark p-6 rounded-2xl shadow-sm border border-panel-border dark:border-panel-border-dark flex flex-col justify-between space-y-4"),
			html.Div(
				html.Class("flex items-start space-x-4"),
				html.Img(html.Src(s.AvatarURL), html.Alt(s.Username), html.Class("w-14 h-14 rounded-full border border-panel-border dark:border-panel-border-dark")),
				html.Div(
					html.Class("flex-grow"),
					html.H3(html.Class("text-lg font-bold text-app-text dark:text-headline-dark flex items-center space-x-2"),
						html.Span(g.Text(s.Username)),
						// Render GitHub icon if primary account is GitHub, otherwise GitLab
						g.If(s.GitHubID != nil, html.A(
							html.Href("https://github.com/"+s.Username),
							html.Target("_blank"),
							html.Class("text-app-text-muted hover:text-brand-primary transition-colors flex items-center"),
							g.Raw(`<svg class="w-5 h-5 fill-current" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg"><path d="M12 .297c-6.63 0-12 5.373-12 12 0 5.303 3.438 9.8 8.205 11.385.6.113.82-.258.82-.577 0-.285-.01-1.04-.015-2.04-3.338.724-4.042-1.61-4.042-1.61C4.422 18.07 3.633 17.7 3.633 17.7c-1.087-.744.084-.729.084-.729 1.205.084 1.838 1.236 1.838 1.236 1.07 1.835 2.809 1.305 3.495.998.108-.776.417-1.305.76-1.605-2.665-.3-5.466-1.332-5.466-5.93 0-1.31.465-2.38 1.235-3.22-.135-.303-.54-1.523.105-3.176 0 0 1.005-.322 3.3 1.23.96-.267 1.98-.399 3-.405 1.02.006 2.04.138 3 .405 2.28-1.552 3.285-1.23 3.285-1.23.645 1.653.24 2.873.12 3.176.765.84 1.23 1.91 1.23 3.22 0 4.61-2.805 5.625-5.475 5.92.42.36.81 1.096.81 2.22 0 1.606-.015 2.896-.015 3.286 0 .315.21.69.825.57C20.565 22.092 24 17.592 24 12.297c0-6.627-5.373-12-12-12"/></svg>`),
						)),
						g.If(s.GitHubID == nil && s.GitLabID != nil, html.A(
							html.Href(func() string {
								if s.GitLabUsername != "" {
									return "https://gitlab.com/" + s.GitLabUsername
								}
								return "https://gitlab.com/" + s.Username
							}()),
							html.Target("_blank"),
							html.Class("text-[#FC6D26] hover:opacity-80 transition-opacity flex items-center"),
							g.Raw(`<svg class="w-5 h-5 fill-current" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg"><path d="M23.953 11.83l-1.637-5.037a.625.625 0 0 0-.239-.33.61.61 0 0 0-.398-.088.618.618 0 0 0-.356.173L12 15.385l-9.324-8.837a.622.622 0 0 0-.355-.173.613.613 0 0 0-.4.089.625.625 0 0 0-.238.329L.047 11.83a1.009 1.009 0 0 0 .34 1.127l11.168 8.113a.774.774 0 0 0 .888 0l11.169-8.113a1.009 1.009 0 0 0 .341-1.127z"/></svg>`),
						)),
					),
					html.P(html.Class("text-xs text-app-text-muted dark:text-app-text-muted-dark mb-2"), g.Text(fmt.Sprintf("Опыт работы: %d %s", s.ExperienceYears, PluralizeYears(s.ExperienceYears)))),
					g.If(s.Stack != "", html.Div(
						html.Class("flex flex-wrap gap-1 mt-1"),
						g.Group(func() []g.Node {
							var tags []g.Node
							for _, t := range strings.Split(s.Stack, ",") {
								trimmed := strings.TrimSpace(t)
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
				html.Class("border-t border-panel-border dark:border-panel-border-dark pt-3 overflow-hidden"),
				g.If(s.GitHubID != nil, html.Img(
					html.Src("https://ghchart.rshah.org/4f46e5/"+s.Username),
					html.Alt(s.Username+"'s GitHub Contributions Chart"),
					html.Class("w-full h-auto min-h-[45px] object-cover rounded-xl dark:invert dark:hue-rotate-180 dark:brightness-75"),
				)),
				g.If(s.GitHubID == nil && s.GitLabID != nil, html.Img(
					html.Src("/profile/gitlab-card.svg?username="+(func() string {
						if s.GitLabUsername != "" {
							return s.GitLabUsername
						}
						return s.Username
					})()),
					html.Alt(s.Username+"'s GitLab Contributions Chart"),
					html.Class("w-full h-auto min-h-[45px] object-cover rounded-xl dark:invert dark:hue-rotate-180 dark:brightness-75"),
				)),
			),
		))
	}

	if len(specCards) == 0 {
		specCards = append(specCards, html.Div(
			html.Class("col-span-full text-center py-12 text-app-text-muted dark:text-app-text-muted-dark bg-panel-bg dark:bg-panel-bg-dark rounded-2xl border border-panel-border dark:border-panel-border-dark"),
			g.Text("Специалисты не найдены по заданным критериям."),
		))
	}

	return html.Div(
		html.Class("grid grid-cols-1 lg:grid-cols-4 gap-8"),
		// Sidebar Filters
		html.Div(
			html.Class("lg:col-span-1 bg-panel-bg dark:bg-panel-bg-dark p-6 rounded-2xl shadow-sm border border-panel-border dark:border-panel-border-dark self-start"),
			html.H2(html.Class("text-lg font-bold text-app-text dark:text-headline-dark mb-4"), g.Text("Фильтры")),
			html.Form(
				html.Method("GET"),
				html.Class("space-y-4"),
				html.Div(
					html.Label(html.Class("block text-xs font-semibold text-app-text-muted dark:text-app-text-muted-dark mb-1 uppercase tracking-wider"), g.Text("Поиск")),
					html.Input(html.Type("text"), html.Name("search"), html.Value(search), html.Placeholder("Имя или навык..."), html.Class("w-full border border-panel-border dark:border-panel-border-dark bg-app-bg dark:bg-app-bg-dark text-app-text dark:text-app-text-dark rounded-xl px-3 py-2 text-sm focus:ring-2 focus:ring-brand-primary dark:focus:ring-brand-primary-dark")),
				),
				html.Div(
					html.Label(html.Class("block text-xs font-semibold text-app-text-muted dark:text-app-text-muted-dark mb-1 uppercase tracking-wider"), g.Text("Конкретная технология")),
					html.Input(html.Type("text"), html.Name("tech"), html.Value(tech), html.Placeholder("Например: Go"), html.Class("w-full border border-panel-border dark:border-panel-border-dark bg-app-bg dark:bg-app-bg-dark text-app-text dark:text-app-text-dark rounded-xl px-3 py-2 text-sm focus:ring-2 focus:ring-brand-primary dark:focus:ring-brand-primary-dark")),
				),
				html.Div(
					html.Label(html.Class("block text-xs font-semibold text-app-text-muted dark:text-app-text-muted-dark mb-1 uppercase tracking-wider"), g.Text("Минимальный опыт (лет)")),
					html.Input(html.Type("number"), html.Name("min_exp"), html.Value(minExp), html.Min("0"), html.Class("w-full border border-panel-border dark:border-panel-border-dark bg-app-bg dark:bg-app-bg-dark text-app-text dark:text-app-text-dark rounded-xl px-3 py-2 text-sm focus:ring-2 focus:ring-brand-primary dark:focus:ring-brand-primary-dark")),
				),
				html.Div(
					html.Label(html.Class("block text-xs font-semibold text-app-text-muted dark:text-app-text-muted-dark mb-1 uppercase tracking-wider"), g.Text("Сортировка")),
					html.Select(
						html.Name("sort"),
						html.Class("w-full border border-panel-border dark:border-panel-border-dark bg-app-bg dark:bg-app-bg-dark text-app-text dark:text-app-text-dark rounded-xl px-3 py-2 text-sm focus:ring-2 focus:ring-brand-primary dark:focus:ring-brand-primary-dark"),
						html.Option(g.Attr("value", "username_asc"), g.If(sort == "username_asc" || sort == "", g.Attr("selected", "selected")), g.Text("По имени (А-Я)")),
						html.Option(g.Attr("value", "username_desc"), g.If(sort == "username_desc", g.Attr("selected", "selected")), g.Text("По имени (Я-А)")),
						html.Option(g.Attr("value", "exp_desc"), g.If(sort == "exp_desc", g.Attr("selected", "selected")), g.Text("По убыванию опыта")),
						html.Option(g.Attr("value", "exp_asc"), g.If(sort == "exp_asc", g.Attr("selected", "selected")), g.Text("По возрастанию опыта")),
					),
				),
				html.Button(
					html.Type("submit"),
					html.Class("w-full bg-brand-primary dark:bg-brand-primary-dark hover:opacity-90 text-white dark:text-btn-text-dark font-semibold py-2 rounded-xl text-sm transition-colors cursor-pointer"),
					g.Text("Применить"),
				),
				html.A(
					html.Href("/"),
					html.Class("block text-center text-xs text-app-text-muted dark:text-app-text-muted-dark hover:text-brand-primary dark:hover:text-brand-primary-dark mt-2"),
					g.Text("Сбросить все"),
				),
			),
		),
		// Specialists Grid
		html.Div(
			html.Class("lg:col-span-3 space-y-6"),
			html.H1(html.Class("text-3xl font-extrabold text-app-text dark:text-headline-dark"), g.Text("Наши специалисты")),
			html.Div(
				html.Class("grid grid-cols-1 md:grid-cols-2 gap-6"),
				g.Group(specCards),
			),
		),
	)
}

func PluralizeYears(years int) string {
	if years%10 == 1 && years%100 != 11 {
		return "год"
	}
	if (years%10 >= 2 && years%10 <= 4) && (years%100 < 12 || years%100 > 14) {
		return "года"
	}
	return "лет"
}
