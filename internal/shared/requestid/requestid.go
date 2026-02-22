package requestid

import "context"

type ctxKey struct{}

func With(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxKey{}, id)
}

func Get(ctx context.Context) string {
	v := ctx.Value(ctxKey{})
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
