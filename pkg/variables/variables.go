package variables

import "context"

type Variables map[string]any

type key int

var varsKey key

func FromContext(ctx context.Context) (Variables, bool) {
	v, ok := ctx.Value(varsKey).(Variables)
	return v, ok
}

func NewContext(ctx context.Context, vars Variables) context.Context {
	return context.WithValue(ctx, varsKey, vars)
}
