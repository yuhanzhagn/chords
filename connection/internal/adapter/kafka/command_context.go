package kafka

import "context"

type commandTokenKey struct{}

func WithCommandToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, commandTokenKey{}, token)
}
func commandToken(ctx context.Context) string {
	token, _ := ctx.Value(commandTokenKey{}).(string)
	return token
}
