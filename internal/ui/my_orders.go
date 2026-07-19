package ui

import (
	"fmt"
	"strconv"
	"shreelance/internal/models"

	g "maragu.dev/gomponents"
	html "maragu.dev/gomponents/html"
)

func MyOrdersPage(orders []models.Order, unreadCounts map[uint]int, user *models.User, role string, csrfToken string) g.Node {
	var cards []g.Node
	for _, o := range orders {
		var badgeColor string
		var statusText string
		switch o.Status {
		case "open":
			badgeColor = "bg-blue-100 text-blue-800 dark:bg-blue-900/40 dark:text-blue-300 border-blue-200 dark:border-blue-800"
			statusText = "Ищет исполнителя"
		case "in_progress":
			badgeColor = "bg-yellow-100 text-yellow-800 dark:bg-yellow-900/40 dark:text-yellow-300 border-yellow-200 dark:border-yellow-800"
			statusText = "В работе"
		case "completed":
			badgeColor = "bg-green-100 text-green-800 dark:bg-green-900/40 dark:text-green-300 border-green-200 dark:border-green-800"
			statusText = "Выполнен"
		case "cancelled":
			badgeColor = "bg-red-100 text-red-800 dark:bg-red-900/40 dark:text-red-300 border-red-200 dark:border-red-800"
			statusText = "Отменен"
		}

		var actionSection g.Node
		if role == "freelancer" && o.Status == "in_progress" {
			actionSection = html.Form(
				html.Action(fmt.Sprintf("/my-orders/%d/status", o.ID)),
				html.Method("POST"),
				html.Class("flex items-center space-x-2 mt-4 pt-4 border-t border-panel-border dark:border-panel-border-dark"),
				html.Input(html.Type("hidden"), html.Name("csrf_token"), html.Value(csrfToken)),
				html.Label(html.Class("text-xs font-semibold text-app-text-muted dark:text-app-text-muted-dark uppercase tracking-wider"), g.Text("Изменить статус:")),
				html.Select(
					html.Name("status"),
					html.Class("border border-panel-border dark:border-panel-border-dark bg-app-bg dark:bg-app-bg-dark text-app-text dark:text-app-text-dark rounded-xl px-2 py-1 text-xs focus:ring-1 focus:ring-brand-primary"),
					html.Option(g.Attr("value", "in_progress"), g.Attr("selected", "selected"), g.Text("В работе")),
					html.Option(g.Attr("value", "completed"), g.Text("Выполнен")),
				),
				html.Button(
					html.Type("submit"),
					html.Class("bg-brand-primary hover:opacity-90 text-white font-medium text-xs py-1 px-3 rounded-xl transition-colors cursor-pointer"),
					g.Text("Обновить"),
				),
			)
		}

		var freelancerInfo g.Node
		if role == "customer" && o.FreelancerID != nil && o.Freelancer != nil {
			freelancerInfo = html.Div(
				html.Class("text-xs text-app-text-muted dark:text-app-text-muted-dark mt-2 pt-2 border-t border-panel-border dark:border-panel-border-dark flex items-center justify-between"),
				html.Span(g.Text("Исполнитель назначен:")),
				html.Span(html.Class("font-semibold text-brand-primary dark:text-brand-primary-dark"), g.Text(o.Freelancer.Username)),
			)
		}

		unreadMsgCount := unreadCounts[o.ID]
		var unreadBadge g.Node
		if unreadMsgCount > 0 {
			unreadBadge = html.Span(
				html.Class("inline-flex items-center justify-center px-2 py-0.5 ml-2 text-xs font-bold leading-none text-white bg-indigo-600 rounded-full animate-pulse"),
				g.Text(fmt.Sprintf("%d новых сообщ.", unreadMsgCount)),
			)
		}

		// Check if there are new/pending bids for this order
		var bidsBadge g.Node
		if role == "customer" && o.Status == "open" {
			pendingBidsCount := 0
			for _, b := range o.Bids {
				if b.Status == "pending" {
					pendingBidsCount++
				}
			}
			if pendingBidsCount > 0 {
				bidsBadge = html.Span(
					html.Class("inline-flex items-center justify-center px-2 py-0.5 ml-2 text-xs font-bold leading-none text-white bg-emerald-600 rounded-full animate-pulse"),
					g.Text(fmt.Sprintf("%d нов. отклик(ов)", pendingBidsCount)),
				)
			}
		}

		cards = append(cards, html.Div(
			html.Class("bg-panel-bg dark:bg-panel-bg-dark p-6 rounded-2xl shadow-sm border border-panel-border dark:border-panel-border-dark flex flex-col justify-between space-y-4"),
			html.Div(
				html.Class("space-y-2"),
				html.Div(
					html.Class("flex justify-between items-start"),
					html.Div(
						html.Class("flex items-center flex-wrap gap-2"),
						html.A(
							html.Href(fmt.Sprintf("/orders/%d", o.ID)),
							html.Class("text-lg font-bold text-app-text dark:text-headline-dark hover:text-brand-primary dark:hover:text-brand-primary-dark transition-colors"),
							g.Text(o.Title),
						),
						unreadBadge,
						bidsBadge,
					),
					html.Span(
						html.Class("text-xs font-semibold px-2.5 py-1 rounded-full border "+badgeColor),
						g.Text(statusText),
					),
				),
				html.P(html.Class("text-xs text-app-text-muted dark:text-app-text-muted-dark"), g.Text("Бюджет: "+strconv.FormatFloat(o.Budget, 'f', 2, 64)+" ₽")),
				html.P(html.Class("text-sm text-app-text-muted dark:text-app-text-muted-dark line-clamp-3"), g.Text(o.Description)),
				freelancerInfo,
			),
			actionSection,
		))
	}

	if len(cards) == 0 {
		cards = append(cards, html.Div(
			html.Class("col-span-full text-center py-12 text-app-text-muted dark:text-app-text-muted-dark bg-panel-bg dark:bg-panel-bg-dark rounded-2xl border border-panel-border dark:border-panel-border-dark"),
			g.Text(map[string]string{
				"customer":   "Вы еще не разместили ни одного заказа.",
				"freelancer": "У вас пока нет взятых в работу заказов.",
			}[role]),
		))
	}

	titleText := map[string]string{"customer": "Мои размещенные заказы", "freelancer": "Мои работы"}[role]

	return html.Div(
		html.Class("space-y-6 max-w-4xl mx-auto"),
		html.H1(html.Class("text-3xl font-extrabold text-app-text dark:text-headline-dark"), g.Text(titleText)),
		html.Div(
			html.Class("grid grid-cols-1 gap-6"),
			g.Group(cards),
		),
	)
}
