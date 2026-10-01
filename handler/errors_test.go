package handler

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

func TestErrorHandler(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode int
		wantBody string
	}{
		{
			name:     "binder uuid error is reworded",
			err:      echo.NewHTTPError(http.StatusBadRequest, "Invalid format for parameter id: error unmarshaling 'x' text as *uuid.UUID: invalid UUID length: 1"),
			wantCode: http.StatusBadRequest,
			wantBody: `{"message":"invalid id"}`,
		},
		{
			name:     "binder int error is reworded",
			err:      echo.NewHTTPError(http.StatusBadRequest, `Invalid format for parameter max_distance: strconv.ParseInt: parsing "abc": invalid syntax`),
			wantCode: http.StatusBadRequest,
			wantBody: `{"message":"invalid max_distance"}`,
		},
		{
			name:     "own message kept",
			err:      echo.NewHTTPError(http.StatusNotFound, "estate not found"),
			wantCode: http.StatusNotFound,
			wantBody: `{"message":"estate not found"}`,
		},
		{
			name:     "unknown route",
			err:      echo.ErrNotFound,
			wantCode: http.StatusNotFound,
			wantBody: `{"message":"Not Found"}`,
		},
		{
			name:     "non-string message falls back to status text",
			err:      echo.NewHTTPError(http.StatusBadRequest, errors.New("raw")),
			wantCode: http.StatusBadRequest,
			wantBody: `{"message":"Bad Request"}`,
		},
		{
			name:     "plain error is a generic 500",
			err:      errors.New("pq: password authentication failed"),
			wantCode: http.StatusInternalServerError,
			wantBody: `{"message":"Internal Server Error"}`,
		},
		{
			name:     "5xx HTTPError message is not leaked",
			err:      echo.NewHTTPError(http.StatusServiceUnavailable, "db pool exhausted"),
			wantCode: http.StatusServiceUnavailable,
			wantBody: `{"message":"Service Unavailable"}`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			c := quietEcho().NewContext(httptest.NewRequest(http.MethodGet, "/", nil), rec)

			ErrorHandler(tc.err, c)

			require.Equal(t, tc.wantCode, rec.Code)
			require.JSONEq(t, tc.wantBody, rec.Body.String())
		})
	}
}

func TestErrorHandlerLogs5xxWithRequestID(t *testing.T) {
	e := echo.New()
	var logs bytes.Buffer
	e.Logger.SetOutput(&logs)
	rec := httptest.NewRecorder()
	c := e.NewContext(httptest.NewRequest(http.MethodGet, "/estate", nil), rec)
	c.Response().Header().Set(echo.HeaderXRequestID, "req-123")

	ErrorHandler(errors.New("pq: connection refused"), c)

	require.Contains(t, logs.String(), "req-123")
	require.Contains(t, logs.String(), "pq: connection refused")
	require.NotContains(t, rec.Body.String(), "pq:")
}

func TestErrorHandlerHeadHasNoBody(t *testing.T) {
	rec := httptest.NewRecorder()
	c := quietEcho().NewContext(httptest.NewRequest(http.MethodHead, "/", nil), rec)

	ErrorHandler(echo.ErrNotFound, c)

	require.Equal(t, http.StatusNotFound, rec.Code)
	require.Empty(t, rec.Body.String())
}

func TestErrorHandlerSkipsCommittedResponse(t *testing.T) {
	rec := httptest.NewRecorder()
	c := quietEcho().NewContext(httptest.NewRequest(http.MethodGet, "/", nil), rec)
	require.NoError(t, c.String(http.StatusOK, "partial"))

	ErrorHandler(errors.New("late failure"), c)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "partial", rec.Body.String())
}
