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
			status := http.StatusUnprocessableEntity
			if e, ok := err.(foundation.ErrorFormatter); ok {
				status = e.Status()
			}

			if ctx.WantsJson() {
				ctx.JSON(status, map[string]any{"error": err.Error()})
				return
			}

			// Validation failures on a browser/Inertia form submit flash the
			// errors back to the previous page.
			if status == http.StatusUnprocessableEntity {
				ctx.Errors("status", err.Error())
				ctx.Back()
				return
			}

			// Other client/server errors get a status-coded response that does
			// not depend on a session being present.
			ctx.Text(status, err.Error())
			return
		}

		handler(ctx, &body)
	}
}
