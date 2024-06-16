package contextkey

import (
	"context"
	"log/slog"
)

// AppendSlogAttrs returns a copy of parent in which the value associated with KeySlogAttrs is
// v or is already associated value but appended with v.
//
// AppendSlogAttrs is safe to use concurrently.
//
// v associated with contexts for KeySlogAttrs can later be retrieved by any of Value method,
// ValueSlogAttrs, ValueSlogAttrsFallback or ValueSlogAttrsDefault.
func AppendSlogAttrs(ctx context.Context, v ...slog.Attr) context.Context {
	if len(v) == 0 {
		return ctx
	}
	attrs := ValueSlogAttrsDefault(ctx)
	newAttrs := make([]slog.Attr, len(attrs)+len(v))
	// This should make always a slice where len(s) == cap(s).
	// Does it really matter?
	// At least calling append on result should always trigger growslice,
	// making accidental data races harder to happen.
	copy(newAttrs, attrs)
	copy(newAttrs[len(attrs):], v)
	return WithSlogAttrs(ctx, newAttrs)
}
