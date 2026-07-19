package ui

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

func Footer() g.Node {
	return h.Footer(
		h.Class("bg-panel-bg dark:bg-panel-bg-dark border-t border-panel-border dark:border-panel-border-dark py-6 mt-12"),
		h.Div(
			h.Class("container mx-auto px-4 text-center text-app-text-muted dark:text-app-text-muted-dark text-sm"),
			g.Text("© 2026 Shreelance. Все права защищены."),
		),
	)
}
