package routing

import (
	"net/http"

	"github.com/martin3zra/forge/foundation"
	"github.com/martin3zra/forge/support"
)

// WithRequest allows defining a handler with a JSON-decoded request body.
func WithRequest[T any](handler func(ctx *Context, body *T)) HandlerFunc {
	return func(ctx *Context) {
		var body T

		err := support.ParseRequest(ctx.Request, &body, ctx.Params)
		if err != nil {
			if e, ok := err.(foundation.ErrorFormatter); ok {
				if e.Status() == http.StatusForbidden {
					if ctx.WantsJson() {
						ctx.JSON(e.Status(), map[string]any{"status": e.Error()})
						return
					}
					ctx.Error(err, e.Status())
					return
				}
			}

			if ctx.WantsJson() {
				ctx.JSON(http.StatusUnprocessableEntity, map[string]any{"error": err.Error()})
				return
			}

			// Review this, it was duplicating the errors
			ctx.Errors("status", err.Error())
			ctx.Back()
			return
		}

		handler(ctx, &body)
	}
}
