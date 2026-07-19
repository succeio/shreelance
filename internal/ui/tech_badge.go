package ui

import (
	"strings"

	g "maragu.dev/gomponents"
	html "maragu.dev/gomponents/html"
)

// TechBadge renders a styled badge for a specific technology/language
func TechBadge(tech string) g.Node {
	t := strings.TrimSpace(tech)
	if t == "" {
		return nil
	}
	lower := strings.ToLower(t)
	colorClass := "bg-slate-100 text-slate-700 border-slate-200 dark:bg-zinc-800 dark:text-zinc-200 dark:border-zinc-700"

	switch {
	case strings.Contains(lower, "go") || strings.Contains(lower, "golang"):
		colorClass = "bg-cyan-100 text-cyan-800 border-cyan-200 dark:bg-cyan-950 dark:text-cyan-200 dark:border-cyan-800"
	case strings.Contains(lower, "python"):
		colorClass = "bg-amber-100 text-amber-800 border-amber-200 dark:bg-amber-950 dark:text-amber-200 dark:border-amber-800"
	case strings.Contains(lower, "typescript") || strings.EqualFold(lower, "ts"):
		colorClass = "bg-blue-100 text-blue-800 border-blue-200 dark:bg-blue-950 dark:text-blue-200 dark:border-blue-800"
	case strings.Contains(lower, "javascript") || strings.EqualFold(lower, "js"):
		colorClass = "bg-yellow-100 text-yellow-800 border-yellow-200 dark:bg-yellow-950 dark:text-yellow-200 dark:border-yellow-800"
	case strings.Contains(lower, "react") || strings.Contains(lower, "vue") || strings.Contains(lower, "next") || strings.Contains(lower, "htmx"):
		colorClass = "bg-sky-100 text-sky-800 border-sky-200 dark:bg-sky-950 dark:text-sky-200 dark:border-sky-800"
	case strings.Contains(lower, "rust"):
		colorClass = "bg-orange-100 text-orange-800 border-orange-200 dark:bg-orange-950 dark:text-orange-200 dark:border-orange-800"
	case strings.Contains(lower, "docker") || strings.Contains(lower, "kubernetes") || strings.Contains(lower, "k8s") || strings.Contains(lower, "devops") || strings.Contains(lower, "gitops"):
		colorClass = "bg-indigo-100 text-indigo-800 border-indigo-200 dark:bg-indigo-950 dark:text-indigo-200 dark:border-indigo-800"
	case strings.Contains(lower, "postgres") || strings.Contains(lower, "sql") || strings.Contains(lower, "redis"):
		colorClass = "bg-emerald-100 text-emerald-800 border-emerald-200 dark:bg-emerald-950 dark:text-emerald-200 dark:border-emerald-800"
	case strings.Contains(lower, "ml") || strings.Contains(lower, "ai") || strings.Contains(lower, "pytorch"):
		colorClass = "bg-rose-100 text-rose-800 border-rose-200 dark:bg-rose-950 dark:text-rose-200 dark:border-rose-800"
	}

	return html.Span(
		html.Class("inline-flex items-center px-3 py-1 rounded-full text-xs font-semibold border "+colorClass),
		g.Text(t),
	)
}
