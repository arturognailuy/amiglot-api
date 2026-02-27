package i18n

import (
	"context"
	"testing"
)

func TestLocaleFromHeader(t *testing.T) {
	cases := []struct {
		name   string
		header string
		want   string
	}{
		{"empty", "", DefaultLocale},
		{"english", "en-GB", "en"},
		{"portuguese", "pt-BR", "pt-BR"},
		{"portuguese_underscore", "pt_BR", "pt-BR"},
		{"chinese", "zh-CN", "zh"},
		{"fallback", "fr-FR", DefaultLocale},
		{"invalid", "???", DefaultLocale},
	}

	for _, tc := range cases {
		if got := LocaleFromHeader(tc.header); got != tc.want {
			t.Fatalf("%s: expected %s, got %s", tc.name, tc.want, got)
		}
	}
}

func TestContextLocale(t *testing.T) {
	ctx := context.Background()
	if got := LocaleFromContext(ctx); got != DefaultLocale {
		t.Fatalf("expected default locale, got %s", got)
	}

	ctx = ContextWithLocale(ctx, "zh")
	if got := LocaleFromContext(ctx); got != "zh" {
		t.Fatalf("expected zh locale, got %s", got)
	}

	ctx = ContextWithLocale(ctx, "")
	if got := LocaleFromContext(ctx); got != DefaultLocale {
		t.Fatalf("expected default locale for empty value, got %s", got)
	}
}

func TestTranslate(t *testing.T) {
	if got := translate("zh", "errors.database_unavailable"); got == "errors.database_unavailable" {
		t.Fatalf("expected translated string, got %s", got)
	}

	if got := translate("pt-BR", "errors.database_unavailable"); got == "errors.database_unavailable" {
		t.Fatalf("expected translated string, got %s", got)
	}

	if got := translate("fr", "errors.database_unavailable"); got != messages[DefaultLocale]["errors.database_unavailable"] {
		t.Fatalf("expected fallback to default locale, got %s", got)
	}

	if got := translate("fr", "errors.unknown_key"); got != "errors.unknown_key" {
		t.Fatalf("expected fallback to key, got %s", got)
	}
}

func TestT(t *testing.T) {
	ctx := ContextWithLocale(context.Background(), "pt-BR")
	if got := T(ctx, "errors.token_invalid"); got == "errors.token_invalid" {
		t.Fatalf("expected translated string, got %s", got)
	}
}
