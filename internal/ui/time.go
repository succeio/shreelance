package ui

import (
	"fmt"
	"math"
	"time"
)

// FormatRelativeTime returns a human-readable relative time string in Russian
func FormatRelativeTime(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	// In case of slight clock drifts, treat future times as "только что"
	if diff < 0 {
		return "только что"
	}

	seconds := diff.Seconds()
	if seconds < 60 {
		return "только что"
	}

	minutes := int(math.Floor(seconds / 60))
	if minutes < 60 {
		return formatRussianPlural(minutes, "минуту назад", "минуты назад", "минут назад")
	}

	hours := int(math.Floor(float64(minutes) / 60))
	if hours < 24 {
		return formatRussianPlural(hours, "час назад", "часа назад", "часов назад")
	}

	days := int(math.Floor(float64(hours) / 24))
	if days < 30 {
		return formatRussianPlural(days, "день назад", "дня назад", "дней назад")
	}

	months := int(math.Floor(float64(days) / 30))
	if months < 12 {
		return formatRussianPlural(months, "месяц назад", "месяца назад", "месяцев назад")
	}

	years := int(math.Floor(float64(months) / 12))
	return formatRussianPlural(years, "год назад", "года назад", "лет назад")
}

func formatRussianPlural(n int, one, two, many string) string {
	mod10 := n % 10
	mod100 := n % 100

	if mod10 == 1 && mod100 != 11 {
		if one == "минуту назад" {
			return "минуту назад"
		}
		if one == "час назад" {
			return "час назад"
		}
		if one == "день назад" {
			return "день назад"
		}
		if one == "месяц назад" {
			return "месяц назад"
		}
		if one == "год назад" {
			return "год назад"
		}
		return fmt.Sprintf("%d %s", n, one)
	}

	if mod10 >= 2 && mod10 <= 4 && (mod100 < 10 || mod100 >= 20) {
		return fmt.Sprintf("%d %s", n, two)
	}

	return fmt.Sprintf("%d %s", n, many)
}
