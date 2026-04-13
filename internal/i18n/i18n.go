package i18n

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

type Locale string

const (
	EN Locale = "en"
	BG Locale = "bg"
)

const CookieName = "fresnel_lang"

var supported = map[Locale]bool{EN: true, BG: true}

type Messages map[string]string

var translations = map[Locale]Messages{
	EN: enMessages,
	BG: bgMessages,
}

type ctxKey struct{}

func WithLocale(ctx context.Context, l Locale) context.Context {
	return context.WithValue(ctx, ctxKey{}, l)
}

func FromContext(ctx context.Context) Locale {
	if v, ok := ctx.Value(ctxKey{}).(Locale); ok {
		return v
	}
	return EN
}

func T(ctx context.Context, key string) string {
	return Translate(FromContext(ctx), key)
}

func Tn(ctx context.Context, key string, n int) string {
	return TranslateN(FromContext(ctx), key, n)
}

func Translate(l Locale, key string) string {
	if msgs, ok := translations[l]; ok {
		if val, ok := msgs[key]; ok {
			return val
		}
	}
	if val, ok := translations[EN][key]; ok {
		return val
	}
	return key
}

func TranslateN(l Locale, key string, n int) string {
	tpl := Translate(l, key)
	return fmt.Sprintf(tpl, n)
}

func ResolveLocale(r *http.Request) Locale {
	if c, err := r.Cookie(CookieName); err == nil {
		if l := Locale(strings.TrimSpace(c.Value)); supported[l] {
			return l
		}
	}
	return parseAcceptLanguage(r.Header.Get("Accept-Language"))
}

func parseAcceptLanguage(header string) Locale {
	for _, part := range strings.Split(header, ",") {
		tag := strings.TrimSpace(strings.SplitN(part, ";", 2)[0])
		tag = strings.ToLower(tag)
		if strings.HasPrefix(tag, "bg") {
			return BG
		}
		if strings.HasPrefix(tag, "en") {
			return EN
		}
	}
	return EN
}
