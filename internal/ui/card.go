package ui

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// Card represents a standard styled container card
func Card(children ...g.Node) g.Node {
	return h.Div(
		h.Class("bg-panel-bg dark:bg-panel-bg-dark shadow-sm rounded-2xl border border-panel-border dark:border-panel-border-dark p-6"),
		g.Group(children),
	)
}
