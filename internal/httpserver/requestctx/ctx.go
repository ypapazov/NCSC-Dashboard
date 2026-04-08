package requestctx

import (
	"context"

	"fresnel/internal/domain"
)

// WithAuth returns a context carrying AuthContext.
func WithAuth(ctx context.Context, a *domain.AuthContext) context.Context {
	return context.WithValue(ctx, KeyAuth, a)
}

// AuthFrom returns the auth context or nil.
func AuthFrom(ctx context.Context) *domain.AuthContext {
	v := ctx.Value(KeyAuth)
	if v == nil {
		return nil
	}
	a, _ := v.(*domain.AuthContext)
	return a
}

// WithRender returns a context carrying the render kind.
func WithRender(ctx context.Context, k RenderKind) context.Context {
	return context.WithValue(ctx, KeyRender, k)
}

// RenderFrom returns HTML or JSON (default HTML).
func RenderFrom(ctx context.Context) RenderKind {
	v := ctx.Value(KeyRender)
	if v == nil {
		return RenderHTML
	}
	k, _ := v.(RenderKind)
	return k
}

// WithRawAccessToken stores the validated bearer/cookie token string for CSRF.
func WithRawAccessToken(ctx context.Context, raw string) context.Context {
	return context.WithValue(ctx, KeyRawAccessToken, raw)
}

// RawAccessTokenFrom returns the raw access token if set.
func RawAccessTokenFrom(ctx context.Context) string {
	v := ctx.Value(KeyRawAccessToken)
	if v == nil {
		return ""
	}
	s, _ := v.(string)
	return s
}
