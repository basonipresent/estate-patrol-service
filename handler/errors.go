package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/SawitProRecruitment/UserService/generated"
	"github.com/labstack/echo/v4"
)

var errEstateNotFound = echo.NewHTTPError(http.StatusNotFound, "estate not found")

func badRequest(msg string) *echo.HTTPError {
	return echo.NewHTTPError(http.StatusBadRequest, msg)
}

// ErrorHandler makes every error response a {"message": ...} body. 5xx details
// are logged with the request ID and never sent to the client.
func ErrorHandler(err error, c echo.Context) {
	if c.Response().Committed {
		return
	}

	code, msg := http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError)
	var he *echo.HTTPError
	if errors.As(err, &he) {
		code, msg = he.Code, publicMessage(he)
	}
	if code >= http.StatusInternalServerError {
		msg = http.StatusText(code)
		c.Logger().Errorf("request_id=%s %s %s: %v",
			c.Response().Header().Get(echo.HeaderXRequestID), c.Request().Method, c.Request().URL.Path, err)
	}

	if c.Request().Method == http.MethodHead {
		err = c.NoContent(code)
	} else {
		err = c.JSON(code, generated.ErrorResponse{Message: msg})
	}
	if err != nil {
		c.Logger().Error(err)
	}
}

// publicMessage hides Go parse errors that the generated parameter binder
// puts in its messages, e.g. "Invalid format for parameter id: ... uuid ...".
func publicMessage(he *echo.HTTPError) string {
	msg, ok := he.Message.(string)
	if !ok {
		return http.StatusText(he.Code)
	}
	if rest, found := strings.CutPrefix(msg, "Invalid format for parameter "); found {
		name, _, _ := strings.Cut(rest, ":")
		return fmt.Sprintf("invalid %s", name)
	}
	return msg
}
