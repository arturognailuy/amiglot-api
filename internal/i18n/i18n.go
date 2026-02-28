package i18n

import (
	"context"
	"strings"

	"golang.org/x/text/language"
)

const DefaultLocale = "en"

type localeKey struct{}

var supportedTags = []language.Tag{
	language.MustParse("en"),
	language.MustParse("en-AU"),
	language.MustParse("en-CA"),
	language.MustParse("en-GB"),
	language.MustParse("en-US"),
	language.SimplifiedChinese,
	language.TraditionalChinese,
	language.MustParse("zh-CN"),
	language.MustParse("zh-SG"),
	language.MustParse("zh-TW"),
	language.MustParse("zh-HK"),
	language.MustParse("zh-MO"),
	language.MustParse("pt"),
	language.MustParse("pt-BR"),
	language.MustParse("pt-PT"),
}

var matcher = language.NewMatcher(supportedTags)

func ContextWithLocale(ctx context.Context, locale string) context.Context {
	return context.WithValue(ctx, localeKey{}, locale)
}

func LocaleFromContext(ctx context.Context) string {
	value := ctx.Value(localeKey{})
	if locale, ok := value.(string); ok && locale != "" {
		return locale
	}
	return DefaultLocale
}

func LocaleFromHeader(header string) string {
	if header == "" {
		return DefaultLocale
	}
	normalized := strings.ReplaceAll(header, "_", "-")
	tags, _, err := language.ParseAcceptLanguage(normalized)
	if err != nil || len(tags) == 0 {
		return DefaultLocale
	}
	tag, _, _ := matcher.Match(tags...)
	return normalizeTag(tag)
}

func normalizeTag(tag language.Tag) string {
	tagStr := tag.String()
	switch tagStr {
	case "zh-Hans", "zh-CN", "zh-SG":
		return "zh-Hans"
	case "zh-Hant", "zh-HK", "zh-TW", "zh-MO":
		return "zh-Hant"
	case "zh":
		return "zh"
	case "pt-PT":
		return "pt-PT"
	case "pt-BR", "pt":
		return "pt-BR"
	case "en-AU":
		return "en-AU"
	case "en-CA":
		return "en-CA"
	case "en-GB":
		return "en-GB"
	case "en-US":
		return "en-US"
	case "en":
		return "en"
	}
	return DefaultLocale
}

func T(ctx context.Context, key string) string {
	locale := LocaleFromContext(ctx)
	return translate(locale, key)
}
