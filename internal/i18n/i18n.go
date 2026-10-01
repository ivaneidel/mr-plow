// Package i18n holds the English and Spanish UI strings.
package i18n

import (
	"os"
	"strings"
)

// Lang is a UI language.
type Lang string

const (
	EN Lang = "en"
	ES Lang = "es"
)

var current = EN

// Text is a translatable string with both languages inline (used by the catalog).
type Text struct{ EN, ES string }

// String returns the text in the current language.
func (t Text) String() string {
	if current == ES && t.ES != "" {
		return t.ES
	}
	return t.EN
}

// Set picks the UI language. "" means detect from LC_ALL / LC_MESSAGES / LANG.
func Set(l string) {
	if l == "" {
		for _, k := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
			if v := os.Getenv(k); v != "" {
				l = v
				break
			}
		}
	}
	if strings.HasPrefix(strings.ToLower(l), "es") {
		current = ES
	} else {
		current = EN
	}
}

// Current returns the active language.
func Current() Lang { return current }

// Toggle switches between English and Spanish.
func Toggle() {
	if current == EN {
		current = ES
	} else {
		current = EN
	}
}

// T looks up a UI string by key; unknown keys are returned as-is.
func T(key string) string {
	if m, ok := strs[key]; ok {
		return m.String()
	}
	return key
}
