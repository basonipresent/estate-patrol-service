package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// RequestTimeout cancels the request context after d, so a slow query is
// aborted and the client gets a 503 instead of a cut connection.
func RequestTimeout(d time.Duration) echo.MiddlewareFunc {
	return middleware.ContextTimeoutWithConfig(middleware.ContextTimeoutConfig{
		Timeout: d,
		// lib/pq reports a cancelled query as its own error, so check the context.
		ErrorHandler: func(err error, c echo.Context) error {
			if errors.Is(c.Request().Context().Err(), context.DeadlineExceeded) {
				return echo.NewHTTPError(http.StatusServiceUnavailable, "request timed out").SetInternal(err)
			}
			return err
		},
	})
}

// bindJSON decodes exactly one JSON value from the body. Unlike echo's Bind it
// rejects an empty body and anything after the value, e.g. `{...}xyz`.
func bindJSON(ctx echo.Context, dst any, msg string) error {
	if !strings.HasPrefix(ctx.Request().Header.Get(echo.HeaderContentType), echo.MIMEApplicationJSON) {
		return badRequest("Content-Type must be application/json")
	}
	dec := json.NewDecoder(ctx.Request().Body)
	if err := dec.Decode(dst); err != nil {
		var he *echo.HTTPError // e.g. 413 from the body limit
		if errors.As(err, &he) {
			return he
		}
		return badRequest(msg)
	}
	if dec.More() {
		return badRequest(msg)
	}
	return nil
}
