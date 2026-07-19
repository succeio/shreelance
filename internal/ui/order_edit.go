package ui

import (
	"fmt"
	"shreelance/internal/models"

	g "maragu.dev/gomponents"
	html "maragu.dev/gomponents/html"
)

func OrderEditForm(order models.Order, csrfToken string) g.Node {
	return html.Div(
		html.Class("max-w-2xl mx-auto bg-panel-bg dark:bg-panel-bg-dark p-8 rounded-2xl shadow-md border border-panel-border dark:border-panel-border-dark"),
		html.H1(html.Class("text-2xl font-bold mb-6 text-app-text dark:text-headline-dark"), g.Text("Редактировать заказ")),
		html.Form(
			html.Action(fmt.Sprintf("/orders/%d/edit", order.ID)),
			html.Method("POST"),
			html.Class("space-y-6"),
			html.Input(html.Type("hidden"), html.Name("csrf_token"), html.Value(csrfToken)),

			// Title
			html.Div(
				html.Label(html.Class("block text-sm font-semibold text-app-text dark:text-headline-dark mb-1"), g.Text("Название задания")),
				html.Input(
					html.Type("text"),
					html.Name("title"),
					html.Required(),
					html.Value(order.Title),
					html.Class("w-full border border-panel-border dark:border-panel-border-dark bg-app-bg dark:bg-app-bg-dark text-app-text dark:text-app-text-dark rounded-xl px-3.5 py-2.5 focus:ring-2 focus:ring-brand-primary dark:focus:ring-brand-primary-dark focus:border-brand-primary dark:focus:border-brand-primary-dark text-sm"),
				),
			),

			// Category
			html.Div(
				html.Label(html.Class("block text-sm font-semibold text-app-text dark:text-headline-dark mb-1"), g.Text("Категория")),
				html.Input(
					html.Type("text"),
					html.Name("category"),
					html.Required(),
					html.Value(order.Category),
					html.Class("w-full border border-panel-border dark:border-panel-border-dark bg-app-bg dark:bg-app-bg-dark text-app-text dark:text-app-text-dark rounded-xl px-3.5 py-2.5 focus:ring-2 focus:ring-brand-primary dark:focus:ring-brand-primary-dark text-sm"),
				),
			),

			// Budget
			html.Div(
				html.Label(html.Class("block text-sm font-semibold text-app-text dark:text-headline-dark mb-1"), g.Text("Бюджет (₽)")),
				html.Input(
					html.Type("number"),
					html.Name("budget"),
					html.Required(),
					html.Value(fmt.Sprintf("%.0f", order.Budget)),
					html.Class("w-full border border-panel-border dark:border-panel-border-dark bg-app-bg dark:bg-app-bg-dark text-app-text dark:text-app-text-dark rounded-xl px-3.5 py-2.5 focus:ring-2 focus:ring-brand-primary dark:focus:ring-brand-primary-dark text-sm"),
				),
			),

			// Required Tech
			html.Div(
				html.Label(html.Class("block text-sm font-semibold text-app-text dark:text-headline-dark mb-1"), g.Text("Языки и технологии (через запятую)")),
				html.Input(
					html.Type("text"),
					html.Name("required_tech"),
					html.Value(order.RequiredTech),
					html.Placeholder("Go, React, Docker"),
					html.Class("w-full border border-panel-border dark:border-panel-border-dark bg-app-bg dark:bg-app-bg-dark text-app-text dark:text-app-text-dark rounded-xl px-3.5 py-2.5 focus:ring-2 focus:ring-brand-primary dark:focus:ring-brand-primary-dark text-sm"),
				),
			),

			// Description
			html.Div(
				html.Label(html.Class("block text-sm font-semibold text-app-text dark:text-headline-dark mb-1"), g.Text("Описание задачи")),
				html.Textarea(
					html.Name("description"),
					html.Required(),
					html.Rows("6"),
					html.Class("w-full border border-panel-border dark:border-panel-border-dark bg-app-bg dark:bg-app-bg-dark text-app-text dark:text-app-text-dark rounded-xl px-3.5 py-2.5 focus:ring-2 focus:ring-brand-primary dark:focus:ring-brand-primary-dark text-sm"),
					g.Text(order.Description),
				),
			),

			html.Button(
				html.Type("submit"),
				html.Class("w-full bg-brand-primary dark:bg-brand-primary-dark hover:opacity-90 text-white dark:text-btn-text-dark font-semibold py-3 rounded-xl transition-all text-sm shadow-md"),
				g.Text("Сохранить изменения"),
			),
		),
	)
}
