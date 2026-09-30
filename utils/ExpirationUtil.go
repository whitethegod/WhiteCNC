package utils

import (
	"fmt"
	"time"
)

func CalculateExpiryString(expiryTime time.Time) string {
	loc, _ := time.LoadLocation("Europe/Warsaw")
	now := time.Now().In(loc)
	nowLocal := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second(), now.Nanosecond(), expiryTime.Location())
	timeUntilExpiry := expiryTime.Sub(nowLocal)

	switch {
	case timeUntilExpiry > 24*time.Hour:
		days := int(timeUntilExpiry.Hours() / 24)
		return fmt.Sprintf("%d days", days)
	case timeUntilExpiry > time.Hour:
		hours := int(timeUntilExpiry.Hours())
		return fmt.Sprintf("%d hours", hours)
	case timeUntilExpiry > time.Minute:
		minutes := int(timeUntilExpiry.Minutes())
		return fmt.Sprintf("%d minutes", minutes)
	case timeUntilExpiry > time.Second:
		seconds := int(timeUntilExpiry.Seconds())
		return fmt.Sprintf("%d seconds", seconds)
	default:
		return "Expired"
	}
}

func CalculateInt(value int) string {
	if value == 0 {
		return "\033[31mfalse\033[0m" 
	}
	return "\033[32mtrue\033[0m" 
}
