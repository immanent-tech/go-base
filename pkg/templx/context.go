package templx

import "context"

const (
	fragmentsCtxKey contextKey = "fragments"
)

type contextKey string

type FragmentKey string

func FragmentKeysToCtx(ctx context.Context, keys ...FragmentKey) context.Context {
	if len(keys) > 0 {
		return context.WithValue(ctx, fragmentsCtxKey, keys)
	}
	return ctx
}

func FragmentKeysFromCtx(ctx context.Context) []FragmentKey {
	keys, found := ctx.Value(fragmentsCtxKey).([]FragmentKey)
	if !found {
		return nil
	}
	return keys
}
