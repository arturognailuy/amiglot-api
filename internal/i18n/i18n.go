package i18n

import (
  "context"
  "strings"

  "golang.org/x/text/language"
)

const DefaultLocale = "en"

type localeKey struct{}

var supportedTags = []language.Tag{
  language.English,
  language.SimplifiedChinese,
  language.MustParse("pt-BR"),
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
  base, _ := tag.Base()
  if base.String() == "zh" {
    return "zh"
  }
  if base.String() == "pt" {
    region, _ := tag.Region()
    if region.String() == "BR" {
      return "pt-BR"
    }
  }
  if base.String() == "en" {
    return "en"
  }
  return DefaultLocale
}

func T(ctx context.Context, key string) string {
  locale := LocaleFromContext(ctx)
  return translate(locale, key)
}
