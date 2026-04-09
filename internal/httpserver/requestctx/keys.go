package requestctx

type ctxKey int

const (
	KeyAuth ctxKey = iota
	KeyRender
)

// RenderKind selects response representation.
type RenderKind int

const (
	RenderHTML RenderKind = iota
	RenderJSON
)
