package handler

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

func TestBindJSON(t *testing.T) {
	type body struct {
		Length int `json:"length"`
	}
	tests := []struct {
		name        string
		contentType string
		body        io.Reader
		wantCode    int // 0 means no error
	}{
		{name: "valid", contentType: "application/json", body: strings.NewReader(`{"length":5}`)},
		{name: "charset param", contentType: "application/json; charset=utf-8", body: strings.NewReader(`{"length":5}`)},
		{name: "trailing whitespace", contentType: "application/json", body: strings.NewReader("{\"length\":5}\n ")},
		{name: "empty body", contentType: "application/json", body: strings.NewReader(``), wantCode: http.StatusBadRequest},
		{name: "trailing data", contentType: "application/json", body: strings.NewReader(`{"length":5}xyz`), wantCode: http.StatusBadRequest},
		{name: "two objects", contentType: "application/json", body: strings.NewReader(`{"length":5}{"length":6}`), wantCode: http.StatusBadRequest},
		{name: "wrong content type", contentType: "text/plain", body: strings.NewReader(`{"length":5}`), wantCode: http.StatusBadRequest},
		{name: "no content type", contentType: "", body: strings.NewReader(`{"length":5}`), wantCode: http.StatusBadRequest},
		{name: "body limit error passes through", contentType: "application/json", body: errReader{echo.ErrStatusRequestEntityTooLarge}, wantCode: http.StatusRequestEntityTooLarge},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", tc.body)
			req.Header.Set(echo.HeaderContentType, tc.contentType)
			c := quietEcho().NewContext(req, httptest.NewRecorder())

			var got body
			err := bindJSON(c, &got, "bad body")

			if tc.wantCode == 0 {
				require.NoError(t, err)
				require.Equal(t, 5, got.Length)
				return
			}
			var he *echo.HTTPError
			require.ErrorAs(t, err, &he)
			require.Equal(t, tc.wantCode, he.Code)
		})
	}
}

type errReader struct{ err error }

func (r errReader) Read([]byte) (int, error) { return 0, r.err }

func TestRequestTimeout(t *testing.T) {
	run := func(h echo.HandlerFunc) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		c := quietEcho().NewContext(httptest.NewRequest(http.MethodGet, "/", nil), rec)
		if err := RequestTimeout(20 * time.Millisecond)(h)(c); err != nil {
			ErrorHandler(err, c)
		}
		return rec
	}

	t.Run("slow request gets 503", func(t *testing.T) {
		rec := run(func(c echo.Context) error {
			<-c.Request().Context().Done()
			// Like lib/pq: a driver error, not context.DeadlineExceeded itself.
			return errors.New("pq: canceling statement due to user request")
		})
		requireResponse(t, rec, http.StatusServiceUnavailable, `{"message":"Service Unavailable"}`)
	})

	t.Run("fast error passes through", func(t *testing.T) {
		rec := run(func(c echo.Context) error { return errEstateNotFound })
		requireResponse(t, rec, http.StatusNotFound, `{"message":"estate not found"}`)
	})

	t.Run("fast success", func(t *testing.T) {
		rec := run(func(c echo.Context) error { return c.JSON(http.StatusOK, map[string]int{"distance": 54}) })
		requireResponse(t, rec, http.StatusOK, `{"distance":54}`)
	})
}
