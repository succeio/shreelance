package ui

import (
	"fmt"
	"strings"

	"shreelance/internal/models"

	g "maragu.dev/gomponents"
	html "maragu.dev/gomponents/html"
)

func renderOrderCard(o models.Order) g.Node {
	return html.Div(
		html.Class("bg-panel-bg dark:bg-panel-bg-dark p-6 rounded-2xl shadow-sm border border-panel-border dark:border-panel-border-dark flex flex-col justify-between space-y-3"),
		html.Div(
			html.Div(
				html.Class("flex justify-between items-start mb-2"),
				html.H3(
					html.Class("text-xl font-bold text-app-text dark:text-headline-dark line-clamp-1"),
					html.A(html.Href(fmt.Sprintf("/orders/%d", o.ID)), html.Class("hover:text-brand-primary dark:hover:text-brand-primary-dark"), g.Text(o.Title)),
				),
				html.Span(
					html.Class("text-lg font-extrabold text-emerald-600 dark:text-emerald-400 ml-2 whitespace-nowrap"),
					g.Text(fmt.Sprintf("%.0f ₽", o.Budget)),
				),
			),
			g.If(o.Category != "", html.Div(
				html.Class("mb-3"),
				html.Span(
					html.Class("inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-semibold bg-purple-50 text-purple-700 border border-purple-100 dark:bg-purple-950 dark:text-purple-300 dark:border-purple-800"),
					g.Text(o.Category),
				),
			)),
			html.P(html.Class("text-app-text-muted dark:text-app-text-muted-dark mb-3 line-clamp-3 text-sm leading-relaxed"), g.Text(o.Description)),
			g.If(o.RequiredTech != "", html.Div(
				html.Class("flex flex-wrap gap-1 mb-2"),
				g.Group(func() []g.Node {
					var tags []g.Node
					for _, t := range strings.Split(o.RequiredTech, ",") {
						trimmed := strings.TrimSpace(t)
						if trimmed != "" {
							tags = append(tags, TechBadge(trimmed))
						}
					}
					return tags
				}()),
			)),
		),
		html.Div(
			html.Class("flex justify-between items-center text-xs text-app-text-muted dark:text-app-text-muted-dark border-t border-panel-border dark:border-panel-border-dark pt-3"),
			html.Span(g.Text("Заказчик: "+o.Customer.Username)),
			html.Span(html.Title(o.CreatedAt.Format("02.01.2006 15:04")), g.Text(FormatRelativeTime(o.CreatedAt))),
		),
	)
}

func renderSidebarFilters(search, minBudget, maxBudget, sort, resetHref string) g.Node {
	return html.Div(
		html.Class("lg:col-span-1 bg-panel-bg dark:bg-panel-bg-dark p-6 rounded-2xl shadow-sm border border-panel-border dark:border-panel-border-dark self-start"),
		html.H2(html.Class("text-lg font-bold text-app-text dark:text-headline-dark mb-4"), g.Text("Фильтры")),
		html.Form(
			html.Method("GET"),
			html.Class("space-y-4"),
			html.Div(
				html.Label(html.Class("block text-xs font-semibold text-app-text-muted dark:text-app-text-muted-dark mb-1 uppercase tracking-wider"), g.Text("Поиск")),
				html.Input(html.Type("text"), html.Name("search"), html.Value(search), html.Placeholder("Название или описание..."), html.Class("w-full border border-panel-border dark:border-panel-border-dark bg-app-bg dark:bg-app-bg-dark text-app-text dark:text-app-text-dark rounded-xl px-3 py-2 text-sm focus:ring-brand-primary dark:focus:ring-brand-primary-dark focus:border-brand-primary dark:focus:border-brand-primary-dark")),
			),
			html.Div(
				html.Label(html.Class("block text-xs font-semibold text-app-text-muted dark:text-app-text-muted-dark mb-1 uppercase tracking-wider"), g.Text("Минимальный бюджет (₽)")),
				html.Input(html.Type("number"), html.Name("min_budget"), html.Value(minBudget), html.Class("w-full border border-panel-border dark:border-panel-border-dark bg-app-bg dark:bg-app-bg-dark text-app-text dark:text-app-text-dark rounded-xl px-3 py-2 text-sm focus:ring-brand-primary dark:focus:ring-brand-primary-dark focus:border-brand-primary dark:focus:border-brand-primary-dark")),
			),
			html.Div(
				html.Label(html.Class("block text-xs font-semibold text-app-text-muted dark:text-app-text-muted-dark mb-1 uppercase tracking-wider"), g.Text("Максимальный бюджет (₽)")),
				html.Input(html.Type("number"), html.Name("max_budget"), html.Value(maxBudget), html.Class("w-full border border-panel-border dark:border-panel-border-dark bg-app-bg dark:bg-app-bg-dark text-app-text dark:text-app-text-dark rounded-xl px-3 py-2 text-sm focus:ring-brand-primary dark:focus:ring-brand-primary-dark focus:border-brand-primary dark:focus:border-brand-primary-dark")),
			),
			html.Div(
				html.Label(html.Class("block text-xs font-semibold text-app-text-muted dark:text-app-text-muted-dark mb-1 uppercase tracking-wider"), g.Text("Сортировка")),
				html.Select(
					html.Name("sort"),
					html.Class("w-full border border-panel-border dark:border-panel-border-dark bg-app-bg dark:bg-app-bg-dark text-app-text dark:text-app-text-dark rounded-xl px-3 py-2 text-sm focus:ring-brand-primary dark:focus:ring-brand-primary-dark focus:border-brand-primary dark:focus:border-brand-primary-dark"),
					html.Option(g.Attr("value", "created_desc"), g.If(sort == "created_desc" || sort == "", g.Attr("selected", "selected")), g.Text("Сначала новые")),
					html.Option(g.Attr("value", "created_asc"), g.If(sort == "created_asc", g.Attr("selected", "selected")), g.Text("Сначала старые")),
					html.Option(g.Attr("value", "budget_desc"), g.If(sort == "budget_desc", g.Attr("selected", "selected")), g.Text("Бюджет: по убыванию")),
					html.Option(g.Attr("value", "budget_asc"), g.If(sort == "budget_asc", g.Attr("selected", "selected")), g.Text("Бюджет: по возрастанию")),
				),
			),
			html.Button(
				html.Type("submit"),
				html.Class("w-full bg-brand-primary dark:bg-brand-primary-dark hover:opacity-90 text-white dark:text-btn-text-dark font-semibold py-2 rounded-xl text-sm transition-colors"),
				g.Text("Применить"),
			),
			html.A(
				html.Href(resetHref),
				html.Class("block text-center text-xs text-app-text-muted dark:text-app-text-muted-dark hover:text-brand-primary dark:hover:text-brand-primary-dark mt-2"),
				g.Text("Сбросить все"),
			),
		),
	)
}

func OrdersDashboard(orders []models.Order, search, minBudget, maxBudget, sort string, isLoggedIn bool) g.Node {
	var orderCards []g.Node
	for _, o := range orders {
		orderCards = append(orderCards, renderOrderCard(o))
	}

	if len(orderCards) == 0 {
		orderCards = append(orderCards, html.Div(
			html.Class("col-span-full text-center py-12 text-app-text-muted dark:text-app-text-muted-dark bg-panel-bg dark:bg-panel-bg-dark rounded-2xl border border-panel-border dark:border-panel-border-dark"),
			g.Text("Заказы не найдены по заданным критериям."),
		))
	}

	var headerSection g.Node
	if !isLoggedIn {
		headerSection = html.Div(
			html.Class("text-center py-10 bg-panel-bg dark:bg-panel-bg-dark rounded-2xl shadow-sm border border-panel-border dark:border-panel-border-dark px-4 mb-8"),
			html.H1(html.Class("text-4xl font-extrabold text-app-text dark:text-headline-dark tracking-tight mb-4"), g.Text("Биржа фриланса нового поколения")),
			html.P(html.Class("text-base text-app-text-muted dark:text-app-text-muted-dark max-w-2xl mx-auto mb-6 leading-relaxed"), g.Text("Один аккаунт для заказа задач и для их исполнения. Авторизуйтесь через GitHub или GitLab, чтобы начать работу.")),
			html.Div(
				html.Class("flex justify-center space-x-4"),
				html.A(
					html.Href("/auth/github"),
					html.Class("inline-block bg-brand-primary dark:bg-brand-primary-dark hover:opacity-90 text-white dark:text-btn-text-dark font-semibold px-6 py-2.5 rounded-xl shadow-md transition-all"),
					g.Text("Войти через GitHub"),
				),
				html.A(
					html.Href("/auth/gitlab"),
					html.Class("inline-block bg-orange-600 hover:bg-orange-700 text-white font-semibold px-6 py-2.5 rounded-xl shadow-md transition-all"),
					g.Text("Войти через GitLab"),
				),
			),
		)
	}

	return html.Div(
		html.Class("space-y-6"),
		headerSection,
		html.Div(
			html.Class("grid grid-cols-1 lg:grid-cols-4 gap-8"),
			renderSidebarFilters(search, minBudget, maxBudget, sort, "/"),
			// Orders List
			html.Div(
				html.Class("lg:col-span-3 space-y-6"),
				html.H1(html.Class("text-3xl font-extrabold text-app-text dark:text-headline-dark"), g.Text("Доступные заказы")),
				html.Div(
					html.Class("grid grid-cols-1 md:grid-cols-2 gap-6"),
					g.Group(orderCards),
				),
			),
		),
	)
}

func OrdersList(orders []models.Order, user *models.User, role string, csrfToken string, search, minBudget, maxBudget, sort string) g.Node {
	var createBtn g.Node
	if user != nil && role == "customer" {
		createBtn = html.A(
			html.Href("/orders/new"),
			html.Class("bg-brand-primary dark:bg-brand-primary-dark hover:opacity-90 text-white dark:text-btn-text-dark px-4 py-2 rounded-xl font-semibold text-sm"),
			g.Text("Создать заказ"),
		)
	}

	var orderCards []g.Node
	for _, o := range orders {
		orderCards = append(orderCards, renderOrderCard(o))
	}

	if len(orderCards) == 0 {
		orderCards = append(orderCards, html.Div(
			html.Class("col-span-full text-center py-12 text-app-text-muted dark:text-app-text-muted-dark bg-panel-bg dark:bg-panel-bg-dark rounded-2xl border border-panel-border dark:border-panel-border-dark"),
			g.Text("Заказы не найдены по заданным критериям."),
		))
	}

	return html.Div(
		html.Class("space-y-6"),
		html.Div(
			html.Class("flex justify-between items-center"),
			html.H1(html.Class("text-3xl font-extrabold text-app-text dark:text-headline-dark"), g.Text("Доступные заказы")),
			createBtn,
		),
		html.Div(
			html.Class("grid grid-cols-1 lg:grid-cols-4 gap-8"),
			renderSidebarFilters(search, minBudget, maxBudget, sort, "/orders"),
			// Orders Cards List
			html.Div(
				html.Class("lg:col-span-3 grid grid-cols-1 md:grid-cols-2 gap-6"),
				g.Group(orderCards),
			),
		),
	)
}

func OrderCreateForm(csrfToken string) g.Node {
	presetTechs := []string{"Go", "Python", "TypeScript", "JavaScript", "Rust", "React", "Vue.js", "Docker", "Kubernetes", "PostgreSQL", "Redis", "TailwindCSS", "HTMX", "GitOps", "ML"}

	return html.Div(
		html.Class("max-w-2xl mx-auto bg-panel-bg dark:bg-panel-bg-dark p-8 rounded-2xl shadow-md border border-panel-border dark:border-panel-border-dark"),
		html.H1(html.Class("text-2xl font-bold mb-6 text-app-text dark:text-headline-dark"), g.Text("Создать новый заказ")),
		html.Form(
			html.Action("/orders"),
			html.Method("POST"),
			html.Class("space-y-6"),
			g.Attr("x-data", `{
				category: 'Фронтенд',
				customCategory: '',
				selectedTechs: [],
				customTechInput: '',
				toggleTech(tech) {
					if (this.selectedTechs.includes(tech)) {
						this.selectedTechs = this.selectedTechs.filter(t => t !== tech);
					} else {
						this.selectedTechs.push(tech);
					}
				},
				addCustomTech() {
					let val = this.customTechInput.trim();
					if (val && !this.selectedTechs.includes(val)) {
						this.selectedTechs.push(val);
						this.customTechInput = '';
					}
				},
				removeTech(tech) {
					this.selectedTechs = this.selectedTechs.filter(t => t !== tech);
				},
				get combinedTechs() {
					return this.selectedTechs.join(', ');
				}
			}`),
			html.Input(html.Type("hidden"), html.Name("csrf_token"), html.Value(csrfToken)),
			html.Input(html.Type("hidden"), html.Name("required_tech"), g.Attr(":value", "combinedTechs")),

			// Title
			html.Div(
				html.Label(html.Class("block text-sm font-semibold text-app-text dark:text-headline-dark mb-1"), g.Text("Название задания")),
				html.Input(html.Type("text"), html.Name("title"), html.Required(), html.Placeholder("Например: Разработка REST API на Go"), html.Class("w-full border border-panel-border dark:border-panel-border-dark bg-app-bg dark:bg-app-bg-dark text-app-text dark:text-app-text-dark rounded-xl px-3.5 py-2.5 focus:ring-2 focus:ring-brand-primary dark:focus:ring-brand-primary-dark focus:border-brand-primary dark:focus:border-brand-primary-dark text-sm")),
			),

			// Category Selection (Interactive Pill Buttons)
			html.Div(
				html.Label(html.Class("block text-sm font-semibold text-app-text dark:text-headline-dark mb-2"), g.Text("Область / Категория")),
				html.Input(html.Type("hidden"), html.Name("category"), g.Attr(":value", "category === 'Другое' ? customCategory : category")),
				html.Div(
					html.Class("flex flex-wrap gap-2 mb-2"),
					g.Group(func() []g.Node {
						cats := []string{"Фронтенд", "Бэкенд", "Фулстак", "GitOps", "DevOps", "Machine Learning", "Другое"}
						var nodes []g.Node
						for _, c := range cats {
							nodes = append(nodes, html.Button(
								html.Type("button"),
								g.Attr("@click", fmt.Sprintf("category = '%s'", c)),
								g.Attr(":class", fmt.Sprintf("category === '%s' ? 'bg-brand-primary dark:bg-brand-primary-dark text-white dark:text-btn-text-dark shadow-sm ring-2 ring-brand-primary dark:ring-brand-primary-dark' : 'bg-app-bg dark:bg-app-bg-dark text-app-text dark:text-app-text-dark border border-panel-border dark:border-panel-border-dark hover:bg-panel-border dark:hover:bg-zinc-800 overflow-hidden'", c)),
								html.Class("px-3.5 py-1.5 rounded-full text-xs font-semibold transition-all cursor-pointer"),
								g.Text(c),
							))
						}
						return nodes
					}()),
				),
				html.Div(
					g.Attr("x-show", "category === 'Другое'"),
					g.Attr("style", "display: none;"),
					html.Input(
						html.Type("text"),
						g.Attr("x-model", "customCategory"),
						html.Placeholder("Укажите свою область (например: QA, Blockchain)"),
						html.Class("w-full border border-panel-border dark:border-panel-border-dark bg-app-bg dark:bg-app-bg-dark text-app-text dark:text-app-text-dark rounded-xl px-3.5 py-2 focus:ring-2 focus:ring-brand-primary dark:focus:ring-brand-primary-dark text-sm mt-2"),
					),
				),
			),

			// Tech Stack Selection (Interactive Preset Chips + Custom Addition)
			html.Div(
				html.Label(html.Class("block text-sm font-semibold text-app-text dark:text-headline-dark mb-2"), g.Text("Необходимые языки и технологии")),

				// Selected technologies preview
				html.Div(
					g.Attr("x-show", "selectedTechs.length > 0"),
					g.Attr("style", "display: none;"),
					html.Class("mb-3 flex flex-wrap gap-1.5 p-3 bg-app-bg dark:bg-app-bg-dark rounded-xl border border-panel-border dark:border-panel-border-dark"),
					html.Template(
						g.Attr("x-for", "tech in selectedTechs"),
						g.Attr(":key", "tech"),
						html.Span(
							html.Class("inline-flex items-center space-x-1.5 px-3 py-1 rounded-full text-xs font-bold bg-brand-primary dark:bg-brand-primary-dark text-white dark:text-btn-text-dark shadow-sm"),
							html.Span(g.Attr("x-text", "tech")),
							html.Button(
								html.Type("button"),
								g.Attr("@click", "removeTech(tech)"),
								html.Class("hover:text-red-200 font-bold ml-1 cursor-pointer"),
								g.Text("×"),
							),
						),
					),
				),

				// Preset chips
				html.Div(
					html.Class("flex flex-wrap gap-1.5 mb-3"),
					g.Group(func() []g.Node {
						var nodes []g.Node
						for _, t := range presetTechs {
							nodes = append(nodes, html.Button(
								html.Type("button"),
								g.Attr("@click", fmt.Sprintf("toggleTech('%s')", t)),
								g.Attr(":class", fmt.Sprintf("selectedTechs.includes('%s') ? 'opacity-40 ring-2 ring-brand-primary dark:ring-brand-primary-dark scale-95' : 'hover:scale-105'", t)),
								html.Class("transition-transform cursor-pointer"),
								TechBadge(t),
							))
						}
						return nodes
					}()),
				),

				// Custom tech input field
				html.Div(
					html.Class("flex space-x-2"),
					html.Input(
						html.Type("text"),
						g.Attr("x-model", "customTechInput"),
						g.Attr("@keydown.enter.prevent", "addCustomTech()"),
						html.Placeholder("Добавить свою технологию..."),
						html.Class("flex-grow border border-panel-border dark:border-panel-border-dark bg-app-bg dark:bg-app-bg-dark text-app-text dark:text-app-text-dark rounded-xl px-3.5 py-2 text-sm focus:ring-2 focus:ring-brand-primary dark:focus:ring-brand-primary-dark"),
					),
					html.Button(
						html.Type("button"),
						g.Attr("@click", "addCustomTech()"),
						html.Class("bg-brand-primary dark:bg-brand-primary-dark hover:opacity-90 text-white dark:text-btn-text-dark px-4 py-2 rounded-xl text-xs font-semibold transition-colors cursor-pointer"),
						g.Text("+ Добавить"),
					),
				),
			),

			// Description
			html.Div(
				html.Label(html.Class("block text-sm font-semibold text-app-text dark:text-headline-dark mb-1"), g.Text("Описание задачи")),
				html.Textarea(html.Name("description"), html.Required(), html.Rows("5"), html.Placeholder("Подробно опишите требования к задаче..."), html.Class("w-full border border-panel-border dark:border-panel-border-dark bg-app-bg dark:bg-app-bg-dark text-app-text dark:text-app-text-dark rounded-xl px-3.5 py-2.5 focus:ring-2 focus:ring-brand-primary dark:focus:ring-brand-primary-dark text-sm")),
			),

			// Budget
			html.Div(
				html.Label(html.Class("block text-sm font-semibold text-app-text dark:text-headline-dark mb-1"), g.Text("Бюджет (₽)")),
				html.Input(html.Type("number"), html.Name("budget"), html.Required(), html.Placeholder("50000"), html.Class("w-full border border-panel-border dark:border-panel-border-dark bg-app-bg dark:bg-app-bg-dark text-app-text dark:text-app-text-dark rounded-xl px-3.5 py-2.5 focus:ring-2 focus:ring-brand-primary dark:focus:ring-brand-primary-dark text-sm")),
			),

			// Submit Button
			html.Button(
				html.Type("submit"),
				html.Class("w-full bg-brand-primary dark:bg-brand-primary-dark hover:opacity-90 text-white dark:text-btn-text-dark font-semibold py-3 rounded-xl transition-all text-sm shadow-md"),
				g.Text("Опубликовать заказ"),
			),
		),
	)
}

func OrderDetail(order models.Order, user *models.User, role string, csrfToken string) g.Node {
	var bidForm g.Node
	if user != nil && role == "freelancer" && order.CustomerID != user.ID && order.Status == "open" {
		// Check if freelancer has a rejected bid or already bid
		hasAlreadyBid := false
		isRejected := false
		for _, b := range order.Bids {
			if b.FreelancerID == user.ID {
				hasAlreadyBid = true
				if b.Status == "rejected" {
					isRejected = true
				}
				break
			}
		}

		if isRejected {
			bidForm = html.Div(
				html.Class("mt-8 bg-panel-bg dark:bg-panel-bg-dark p-6 rounded-2xl border border-panel-border dark:border-panel-border-dark text-center"),
				html.P(html.Class("text-sm font-semibold text-red-600 dark:text-red-400"), g.Text("Заказчик отклонил ваш отклик на этот заказ")),
			)
		} else if hasAlreadyBid {
			bidForm = html.Div(
				html.Class("mt-8 bg-panel-bg dark:bg-panel-bg-dark p-6 rounded-2xl border border-panel-border dark:border-panel-border-dark text-center"),
				html.P(html.Class("text-sm font-semibold text-brand-primary dark:text-brand-primary-dark"), g.Text("Вы уже откликнулись на этот заказ")),
			)
		} else {
			bidForm = html.Div(
				html.Class("mt-8 bg-panel-bg dark:bg-panel-bg-dark p-6 rounded-2xl border border-panel-border dark:border-panel-border-dark"),
				html.H3(html.Class("text-lg font-bold mb-4 text-app-text dark:text-headline-dark"), g.Text("Откликнуться на заказ")),
				html.Form(
					html.Action(fmt.Sprintf("/orders/%d/bids", order.ID)),
					html.Method("POST"),
					html.Class("space-y-4"),
					html.Input(html.Type("hidden"), html.Name("csrf_token"), html.Value(csrfToken)),
					html.Div(
						html.Label(html.Class("block text-sm font-semibold text-app-text dark:text-headline-dark mb-1"), g.Text("Предлагаемая стоимость (₽)")),
						html.Input(html.Type("number"), html.Name("price"), html.Required(), html.Class("w-full border border-panel-border dark:border-panel-border-dark bg-app-bg dark:bg-app-bg-dark text-app-text dark:text-app-text-dark rounded-xl px-3.5 py-2.5 focus:ring-2 focus:ring-brand-primary dark:focus:ring-brand-primary-dark text-sm")),
					),
					html.Div(
						html.Label(html.Class("block text-sm font-semibold text-app-text dark:text-headline-dark mb-1"), g.Text("Сопроводительное письмо")),
						html.Textarea(html.Name("comment"), html.Required(), html.Rows("3"), html.Class("w-full border border-panel-border dark:border-panel-border-dark bg-app-bg dark:bg-app-bg-dark text-app-text dark:text-app-text-dark rounded-xl px-3.5 py-2.5 focus:ring-2 focus:ring-brand-primary dark:focus:ring-brand-primary-dark text-sm")),
					),
					html.Button(
						html.Type("submit"),
						html.Class("w-full bg-emerald-600 hover:bg-emerald-700 text-white font-semibold py-2 rounded-xl transition-colors cursor-pointer"),
						g.Text("Отправить отклик"),
					),
				),
			)
		}
	}

	var bidsList []g.Node
	for _, b := range order.Bids {
		// Only show the bid to the Order Owner (Customer) or the Author of the bid (Freelancer)
		if user == nil || (order.CustomerID != user.ID && b.FreelancerID != user.ID) {
			continue
		}

		// Don't show rejected bids to customer or freelancer in the list
		if b.Status == "rejected" {
			continue
		}

		// If current user is the customer and the order is open, they should see "Accept" and "Reject" buttons
		var acceptButton g.Node
		var rejectButton g.Node
		if user.ID == order.CustomerID && order.Status == "open" {
			acceptButton = html.Form(
				html.Action(fmt.Sprintf("/orders/%d/bids/%d/accept", order.ID, b.ID)),
				html.Method("POST"),
				html.Class("inline-block"),
				html.Input(html.Type("hidden"), html.Name("csrf_token"), html.Value(csrfToken)),
				html.Button(
					html.Type("submit"),
					html.Class("bg-brand-primary dark:bg-brand-primary-dark hover:opacity-90 text-white dark:text-btn-text-dark text-xs font-semibold py-1.5 px-3 rounded-xl transition-colors cursor-pointer"),
					g.Text("Выбрать исполнителем"),
				),
			)
			rejectButton = html.Form(
				html.Action(fmt.Sprintf("/orders/%d/bids/%d/reject", order.ID, b.ID)),
				html.Method("POST"),
				html.Class("inline-block"),
				html.Input(html.Type("hidden"), html.Name("csrf_token"), html.Value(csrfToken)),
				html.Button(
					html.Type("submit"),
					html.Class("bg-red-600 hover:bg-red-700 text-white text-xs font-semibold py-1.5 px-3 rounded-xl transition-colors cursor-pointer"),
					g.Text("Отклонить"),
				),
			)
		}

		bidsList = append(bidsList, html.Div(
			html.Class("p-4 border-b border-panel-border dark:border-panel-border-dark last:border-0"),
			html.Div(
				html.Class("flex justify-between items-start mb-2"),
				html.Div(
					html.P(html.Class("font-bold text-app-text dark:text-headline-dark"), g.Text(b.Freelancer.Username)),
					html.P(html.Class("text-xs text-app-text-muted dark:text-app-text-muted-dark"), html.Title(b.CreatedAt.Format("02.01.2006 15:04")), g.Text(FormatRelativeTime(b.CreatedAt))),
				),
				html.Div(
					html.Class("flex items-center space-x-3"),
					html.Span(html.Class("font-bold text-emerald-600 dark:text-emerald-400 mr-2"), g.Text(fmt.Sprintf("%.0f ₽", b.Price))),
					acceptButton,
					rejectButton,
				),
			),
			html.P(html.Class("text-sm text-app-text-muted dark:text-app-text-muted-dark"), g.Text(b.Comment)),
		))
	}

	var bidsContainer g.Node
	if order.Status == "open" {
		if len(bidsList) > 0 {
			bidsContainer = html.Div(
				html.Class("mt-8 bg-panel-bg dark:bg-panel-bg-dark rounded-2xl shadow-sm border border-panel-border dark:border-panel-border-dark overflow-hidden"),
				html.Div(html.Class("p-4 border-b border-panel-border dark:border-panel-border-dark bg-app-bg dark:bg-app-bg-dark rounded-t-2xl"), html.H3(html.Class("font-bold text-app-text dark:text-headline-dark"), g.Text("Отклики исполнителей"))),
				html.Div(g.Group(bidsList)),
			)
		} else {
			bidsContainer = html.Div(
				html.Class("mt-8 text-center text-app-text-muted dark:text-app-text-muted-dark py-6"),
				g.Text("Откликов на это задание пока нет."),
			)
		}
	}

	var chatContainer g.Node
	if (order.Status == "in_progress" || order.Status == "completed") && user != nil && (order.CustomerID == user.ID || (order.FreelancerID != nil && *order.FreelancerID == user.ID)) {
		var chatTitle string
		if order.Freelancer != nil {
			chatTitle = fmt.Sprintf("Чат по заказу (Исполнитель: %s)", order.Freelancer.Username)
		} else {
			chatTitle = "Чат по заказу"
		}

		chatContainer = html.Div(
			html.Class("mt-8 bg-panel-bg dark:bg-panel-bg-dark rounded-2xl shadow-sm border border-panel-border dark:border-panel-border-dark p-6"),
			html.H3(html.Class("text-lg font-bold text-app-text dark:text-headline-dark mb-4"), g.Text(chatTitle)),
			html.Div(
				html.ID("chat-messages"),
				html.Class("h-64 overflow-y-auto p-4 bg-app-bg dark:bg-app-bg-dark rounded-xl border border-panel-border dark:border-panel-border-dark mb-4 space-y-2"),
				g.Attr("hx-get", fmt.Sprintf("/orders/%d/chat/messages", order.ID)),
				g.Attr("hx-trigger", "load, every 2s"),
				g.Text("Загрузка сообщений..."),
			),
			html.Form(
				g.Attr("hx-post", fmt.Sprintf("/orders/%d/chat/send", order.ID)),
				g.Attr("hx-target", "#chat-messages"),
				g.Attr("hx-swap", "beforeend"),
				g.Attr("hx-on::after-request", "this.reset()"),
				html.Class("flex space-x-2"),
				html.Input(html.Type("hidden"), html.Name("csrf_token"), html.Value(csrfToken)),
				html.Input(
					html.Type("text"),
					html.Name("message"),
					html.Required(),
					html.Placeholder("Введите сообщение..."),
					html.Class("flex-grow border border-panel-border dark:border-panel-border-dark bg-app-bg dark:bg-app-bg-dark text-app-text dark:text-app-text-dark rounded-xl px-3.5 py-2 focus:ring-2 focus:ring-brand-primary dark:focus:ring-brand-primary-dark text-sm"),
				),
				html.Button(
					html.Type("submit"),
					html.Class("bg-brand-primary dark:bg-brand-primary-dark hover:opacity-90 text-white dark:text-btn-text-dark font-semibold px-4 py-2 rounded-xl transition-colors cursor-pointer text-sm"),
					g.Text("Отправить"),
				),
			),
		)
	}

	var actionButtons g.Node
	if user != nil && (order.CustomerID == user.ID || (order.FreelancerID != nil && *order.FreelancerID == user.ID)) {
		var buttons []g.Node
		if order.Status == "in_progress" {
			buttons = append(buttons, html.Form(
				html.Action(fmt.Sprintf("/orders/%d/cancel", order.ID)),
				html.Method("POST"),
				html.Input(html.Type("hidden"), html.Name("csrf_token"), html.Value(csrfToken)),
				html.Button(
					html.Type("submit"),
					html.Class("bg-red-600 hover:bg-red-700 text-white font-semibold py-2 px-4 rounded-xl transition-colors cursor-pointer text-sm"),
					g.Text("Отказаться от работы (Вернуть в список / Отменить)"),
				),
			))

			if user.ID == order.CustomerID {
				buttons = append(buttons, html.Form(
					html.Action(fmt.Sprintf("/my-orders/%d/status", order.ID)),
					html.Method("POST"),
					html.Input(html.Type("hidden"), html.Name("csrf_token"), html.Value(csrfToken)),
					html.Input(html.Type("hidden"), html.Name("status"), html.Value("completed")),
					html.Button(
						html.Type("submit"),
						html.Class("bg-emerald-600 hover:bg-emerald-700 text-white font-semibold py-2 px-4 rounded-xl transition-colors cursor-pointer text-sm"),
						g.Text("Завершить заказ"),
					),
				))
			}
		}

		if order.Status == "open" && user.ID == order.CustomerID {
			buttons = append(buttons, html.A(
				html.Href(fmt.Sprintf("/orders/%d/edit", order.ID)),
				html.Class("bg-brand-primary dark:bg-brand-primary-dark hover:opacity-90 text-white dark:text-btn-text-dark font-semibold py-2 px-4 rounded-xl transition-colors text-sm"),
				g.Text("Редактировать заказ"),
			))
		}

		if len(buttons) > 0 {
			actionButtons = html.Div(
				html.Class("flex space-x-4 mt-6"),
				g.Group(buttons),
			)
		}
	}

	return html.Div(
		html.Class("max-w-3xl mx-auto space-y-6"),
		html.Div(
			html.Class("bg-panel-bg dark:bg-panel-bg-dark p-8 rounded-2xl shadow-sm border border-panel-border dark:border-panel-border-dark"),
			html.Div(
				html.Class("flex justify-between items-start mb-6"),
				html.Div(
					html.H1(html.Class("text-3xl font-extrabold text-app-text dark:text-headline-dark"), g.Text(order.Title)),
					html.Div(
						html.Class("flex items-center space-x-2 mt-2"),
						html.Span(
							html.Class("inline-block px-2.5 py-0.5 rounded-full text-xs font-semibold "+
								map[string]string{
									"open":        "bg-green-100 text-green-800 dark:bg-green-950 dark:text-green-300",
									"in_progress": "bg-blue-100 text-blue-800 dark:bg-blue-950 dark:text-blue-300",
									"completed":   "bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-300",
									"cancelled":   "bg-red-100 text-red-800 dark:bg-red-950 dark:text-red-300",
								}[order.Status]),
							g.Text(map[string]string{
								"open":        "Открыт",
								"in_progress": "В работе",
								"completed":   "Завершен",
								"cancelled":   "Отменен",
							}[order.Status]),
						),
						g.If(order.Category != "", html.Span(
							html.Class("inline-block px-2.5 py-0.5 rounded-full text-xs font-semibold bg-purple-100 text-purple-800 dark:bg-purple-950 dark:text-purple-300"),
							g.Text(order.Category),
						)),
					),
				),
				html.Span(html.Class("text-2xl font-extrabold text-emerald-600 dark:text-emerald-400"), g.Text(fmt.Sprintf("%.0f ₽", order.Budget))),
			),
			g.If(order.RequiredTech != "", html.Div(
				html.Class("mb-6 border-b border-panel-border dark:border-panel-border-dark pb-4"),
				html.H4(html.Class("text-xs font-semibold text-app-text-muted dark:text-app-text-muted-dark uppercase tracking-wider mb-2"), g.Text("Необходимые технологии")),
				html.Div(
					html.Class("flex flex-wrap gap-2"),
					g.Group(func() []g.Node {
						var tags []g.Node
						for _, t := range strings.Split(order.RequiredTech, ",") {
							trimmed := strings.TrimSpace(t)
							if trimmed != "" {
								tags = append(tags, TechBadge(trimmed))
							}
						}
						return tags
					}()),
				),
			)),
			html.Div(
				html.Class("prose max-w-none text-app-text dark:text-app-text-dark mb-6 leading-relaxed"),
				g.Text(order.Description),
			),
			html.Div(
				html.Class("flex justify-between items-center text-xs text-app-text-muted dark:text-app-text-muted-dark border-t border-panel-border dark:border-panel-border-dark pt-4"),
				html.Span(g.Text("Заказчик: "+order.Customer.Username)),
				html.Span(html.Title(order.CreatedAt.Format("02.01.2006 15:04")), g.Text("Дата публикации: "+FormatRelativeTime(order.CreatedAt))),
			),
			actionButtons,
		),
		bidForm,
		bidsContainer,
		chatContainer,
	)
}
