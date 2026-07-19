package ui

import (
	"fmt"
	"strings"

	g "maragu.dev/gomponents"
	html "maragu.dev/gomponents/html"
	"shreelance/internal/models"
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
					html.H3(html.Class("text-lg font-bold text-app-text dark:text-headline-dark"), g.Text(s.Username)),
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
				html.Img(
					html.Src("https://ghchart.rshah.org/4f46e5/"+s.Username),
					html.Alt(s.Username+"'s GitHub Contributions Chart"),
					html.Class("w-full h-auto min-h-[45px] object-cover rounded-xl dark:invert dark:hue-rotate-180 dark:brightness-75"),
				),
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
