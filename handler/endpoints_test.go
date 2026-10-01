package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/SawitProRecruitment/UserService/droneplan"
	"github.com/SawitProRecruitment/UserService/generated"
	"github.com/SawitProRecruitment/UserService/repository"
	"github.com/SawitProRecruitment/UserService/treestats"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

var (
	estateID  = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	treeID    = uuid.MustParse("22222222-2222-2222-2222-222222222222")
	errDB     = errors.New("connection reset")
	estate5x1 = repository.GetEstateOutput{Id: estateID, Length: 5, Width: 1}
)

func newServer(t *testing.T) (*Server, *repository.MockRepositoryInterface) {
	repo := repository.NewMockRepositoryInterface(gomock.NewController(t))
	return NewServer(NewServerOptions{Repository: repo}), repo
}

// call runs a handler the way Echo would, including the error handler.
func call(t *testing.T, method, body string, h func(echo.Context) error) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, "/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := echo.New().NewContext(req, rec)
	if err := h(c); err != nil {
		ErrorHandler(err, c)
	}
	return rec
}

func requireResponse(t *testing.T, rec *httptest.ResponseRecorder, code int, body string) {
	t.Helper()
	require.Equal(t, code, rec.Code)
	require.JSONEq(t, body, rec.Body.String())
}

func TestCreateEstate(t *testing.T) {
	t.Run("201 with id", func(t *testing.T) {
		s, repo := newServer(t)
		repo.EXPECT().CreateEstate(gomock.Any(), repository.CreateEstateInput{Length: 5, Width: 1}).
			Return(repository.CreateEstateOutput{Id: estateID}, nil)

		rec := call(t, http.MethodPost, `{"length":5,"width":1}`, s.CreateEstate)
		requireResponse(t, rec, http.StatusCreated, `{"id":"`+estateID.String()+`"}`)
	})

	badBodies := []struct{ name, body string }{
		{name: "empty body", body: ``},
		{name: "malformed json", body: `{"length":`},
		{name: "float", body: `{"length":10.5,"width":1}`},
		{name: "string number", body: `{"length":"10","width":1}`},
		{name: "zero length", body: `{"length":0,"width":1}`},
		{name: "negative", body: `{"length":-1,"width":-5}`},
		{name: "too wide", body: `{"length":1,"width":50001}`},
		{name: "missing width", body: `{"length":5}`},
		{name: "trailing data", body: `{"length":5,"width":1}xyz`},
		{name: "two objects", body: `{"length":5,"width":1}{"length":9,"width":9}`},
	}
	for _, tc := range badBodies {
		t.Run("400 "+tc.name, func(t *testing.T) {
			s, _ := newServer(t) // no repository call expected
			rec := call(t, http.MethodPost, tc.body, s.CreateEstate)
			require.Equal(t, http.StatusBadRequest, rec.Code)
			require.Contains(t, rec.Body.String(), `"message"`)
		})
	}

	t.Run("upper bound is allowed", func(t *testing.T) {
		s, repo := newServer(t)
		repo.EXPECT().CreateEstate(gomock.Any(), repository.CreateEstateInput{Length: 50000, Width: 50000}).
			Return(repository.CreateEstateOutput{Id: estateID}, nil)

		rec := call(t, http.MethodPost, `{"length":50000,"width":50000}`, s.CreateEstate)
		require.Equal(t, http.StatusCreated, rec.Code)
	})

	t.Run("unknown fields are ignored", func(t *testing.T) {
		s, repo := newServer(t)
		repo.EXPECT().CreateEstate(gomock.Any(), gomock.Any()).Return(repository.CreateEstateOutput{Id: estateID}, nil)

		rec := call(t, http.MethodPost, `{"length":5,"width":1,"name":"north"}`, s.CreateEstate)
		require.Equal(t, http.StatusCreated, rec.Code)
	})

	t.Run("500 hides db error", func(t *testing.T) {
		s, repo := newServer(t)
		repo.EXPECT().CreateEstate(gomock.Any(), gomock.Any()).Return(repository.CreateEstateOutput{}, errDB)

		rec := call(t, http.MethodPost, `{"length":5,"width":1}`, s.CreateEstate)
		requireResponse(t, rec, http.StatusInternalServerError, `{"message":"Internal Server Error"}`)
	})
}

func TestCreateTree(t *testing.T) {
	createTree := func(s *Server) func(echo.Context) error {
		return func(c echo.Context) error { return s.CreateTree(c, estateID) }
	}

	t.Run("201 with id", func(t *testing.T) {
		s, repo := newServer(t)
		repo.EXPECT().GetEstate(gomock.Any(), repository.GetEstateInput{Id: estateID}).Return(estate5x1, nil)
		repo.EXPECT().CreateTree(gomock.Any(), repository.CreateTreeInput{EstateId: estateID, X: 5, Y: 1, Height: 30}).
			Return(repository.CreateTreeOutput{Id: treeID}, nil)

		rec := call(t, http.MethodPost, `{"x":5,"y":1,"height":30}`, createTree(s))
		requireResponse(t, rec, http.StatusCreated, `{"id":"`+treeID.String()+`"}`)
	})

	// Rejected before touching the database.
	invalid := []struct{ name, body string }{
		{name: "empty body", body: ``},
		{name: "malformed json", body: `{"x":1,`},
		{name: "height 0", body: `{"x":1,"y":1,"height":0}`},
		{name: "height 31", body: `{"x":1,"y":1,"height":31}`},
	}
	for _, tc := range invalid {
		t.Run("400 "+tc.name, func(t *testing.T) {
			s, _ := newServer(t)
			rec := call(t, http.MethodPost, tc.body, createTree(s))
			require.Equal(t, http.StatusBadRequest, rec.Code)
		})
	}

	// Rejected by the bounds check against the 5x1 estate.
	outside := []struct{ name, body string }{
		{name: "x 0", body: `{"x":0,"y":1,"height":5}`},
		{name: "y 0", body: `{"x":1,"y":0,"height":5}`},
		{name: "x past length", body: `{"x":6,"y":1,"height":5}`},
		{name: "y past width", body: `{"x":1,"y":2,"height":5}`},
	}
	for _, tc := range outside {
		t.Run("400 outside estate "+tc.name, func(t *testing.T) {
			s, repo := newServer(t)
			repo.EXPECT().GetEstate(gomock.Any(), gomock.Any()).Return(estate5x1, nil)

			rec := call(t, http.MethodPost, tc.body, createTree(s))
			require.Equal(t, http.StatusBadRequest, rec.Code)
			require.Contains(t, rec.Body.String(), "outside")
		})
	}

	t.Run("404 unknown estate", func(t *testing.T) {
		s, repo := newServer(t)
		repo.EXPECT().GetEstate(gomock.Any(), gomock.Any()).Return(repository.GetEstateOutput{}, repository.ErrNotFound)

		rec := call(t, http.MethodPost, `{"x":1,"y":1,"height":5}`, createTree(s))
		requireResponse(t, rec, http.StatusNotFound, `{"message":"estate not found"}`)
	})

	insertErrors := []struct {
		name     string
		err      error
		wantCode int
		wantBody string
	}{
		{name: "occupied plot", err: repository.ErrPlotOccupied, wantCode: http.StatusBadRequest, wantBody: `{"message":"plot already has a tree"}`},
		{name: "estate gone", err: repository.ErrNotFound, wantCode: http.StatusNotFound, wantBody: `{"message":"estate not found"}`},
		{name: "db error", err: errDB, wantCode: http.StatusInternalServerError, wantBody: `{"message":"Internal Server Error"}`},
	}
	for _, tc := range insertErrors {
		t.Run(tc.name, func(t *testing.T) {
			s, repo := newServer(t)
			repo.EXPECT().GetEstate(gomock.Any(), gomock.Any()).Return(estate5x1, nil)
			repo.EXPECT().CreateTree(gomock.Any(), gomock.Any()).Return(repository.CreateTreeOutput{}, tc.err)

			rec := call(t, http.MethodPost, `{"x":1,"y":1,"height":5}`, createTree(s))
			requireResponse(t, rec, tc.wantCode, tc.wantBody)
		})
	}

	t.Run("500 when estate lookup fails", func(t *testing.T) {
		s, repo := newServer(t)
		repo.EXPECT().GetEstate(gomock.Any(), gomock.Any()).Return(repository.GetEstateOutput{}, errDB)

		rec := call(t, http.MethodPost, `{"x":1,"y":1,"height":5}`, createTree(s))
		require.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}

func TestGetEstateStats(t *testing.T) {
	getStats := func(s *Server) func(echo.Context) error {
		return func(c echo.Context) error { return s.GetEstateStats(c, estateID) }
	}

	t.Run("200 with stats", func(t *testing.T) {
		s, repo := newServer(t)
		repo.EXPECT().GetEstate(gomock.Any(), gomock.Any()).Return(estate5x1, nil)
		repo.EXPECT().GetHeightHistogram(gomock.Any(), repository.GetHeightHistogramInput{EstateId: estateID}).
			Return(repository.GetHeightHistogramOutput{Histogram: []treestats.HeightCount{
				{Height: 5, Count: 1}, {Height: 3, Count: 1}, {Height: 4, Count: 1},
			}}, nil)

		rec := call(t, http.MethodGet, ``, getStats(s))
		requireResponse(t, rec, http.StatusOK, `{"count":3,"min":3,"max":5,"median":4}`)
	})

	t.Run("200 zeros for an estate without trees", func(t *testing.T) {
		s, repo := newServer(t)
		repo.EXPECT().GetEstate(gomock.Any(), gomock.Any()).Return(estate5x1, nil)
		repo.EXPECT().GetHeightHistogram(gomock.Any(), gomock.Any()).Return(repository.GetHeightHistogramOutput{}, nil)

		rec := call(t, http.MethodGet, ``, getStats(s))
		requireResponse(t, rec, http.StatusOK, `{"count":0,"min":0,"max":0,"median":0}`)
	})

	t.Run("404 unknown estate", func(t *testing.T) {
		s, repo := newServer(t)
		repo.EXPECT().GetEstate(gomock.Any(), gomock.Any()).Return(repository.GetEstateOutput{}, repository.ErrNotFound)

		rec := call(t, http.MethodGet, ``, getStats(s))
		requireResponse(t, rec, http.StatusNotFound, `{"message":"estate not found"}`)
	})

	t.Run("500 when histogram fails", func(t *testing.T) {
		s, repo := newServer(t)
		repo.EXPECT().GetEstate(gomock.Any(), gomock.Any()).Return(estate5x1, nil)
		repo.EXPECT().GetHeightHistogram(gomock.Any(), gomock.Any()).Return(repository.GetHeightHistogramOutput{}, errDB)

		rec := call(t, http.MethodGet, ``, getStats(s))
		require.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}

func TestGetDronePlan(t *testing.T) {
	getPlan := func(s *Server, maxDistance *int64) func(echo.Context) error {
		return func(c echo.Context) error {
			return s.GetDronePlan(c, estateID, generated.GetDronePlanParams{MaxDistance: maxDistance})
		}
	}
	specTrees := repository.ListTreesOutput{Trees: []droneplan.Tree{
		{X: 2, Y: 1, Height: 5}, {X: 3, Y: 1, Height: 3}, {X: 4, Y: 1, Height: 4},
	}}

	t.Run("200 with distance", func(t *testing.T) {
		s, repo := newServer(t)
		repo.EXPECT().GetEstate(gomock.Any(), gomock.Any()).Return(estate5x1, nil)
		repo.EXPECT().ListTrees(gomock.Any(), repository.ListTreesInput{EstateId: estateID}).Return(specTrees, nil)

		rec := call(t, http.MethodGet, ``, getPlan(s, nil))
		requireResponse(t, rec, http.StatusOK, `{"distance":54}`)
	})

	bonus := []struct {
		name        string
		maxDistance int64
		wantBody    string
	}{
		{name: "battery runs out mid-move", maxDistance: 20, wantBody: `{"distance":20,"rest":{"x":2,"y":1}}`},
		{name: "battery covers the plan", maxDistance: 100, wantBody: `{"distance":54,"rest":{"x":5,"y":1}}`},
	}
	for _, tc := range bonus {
		t.Run("200 with rest: "+tc.name, func(t *testing.T) {
			s, repo := newServer(t)
			repo.EXPECT().GetEstate(gomock.Any(), gomock.Any()).Return(estate5x1, nil)
			repo.EXPECT().ListTrees(gomock.Any(), gomock.Any()).Return(specTrees, nil)

			rec := call(t, http.MethodGet, ``, getPlan(s, &tc.maxDistance))
			requireResponse(t, rec, http.StatusOK, tc.wantBody)
		})
	}

	for _, v := range []int64{0, -1} {
		v := v
		t.Run("400 non-positive max_distance", func(t *testing.T) {
			s, _ := newServer(t)
			rec := call(t, http.MethodGet, ``, getPlan(s, &v))
			requireResponse(t, rec, http.StatusBadRequest, `{"message":"max_distance must be a positive integer"}`)
		})
	}

	t.Run("404 unknown estate", func(t *testing.T) {
		s, repo := newServer(t)
		repo.EXPECT().GetEstate(gomock.Any(), gomock.Any()).Return(repository.GetEstateOutput{}, repository.ErrNotFound)

		rec := call(t, http.MethodGet, ``, getPlan(s, nil))
		requireResponse(t, rec, http.StatusNotFound, `{"message":"estate not found"}`)
	})

	t.Run("500 when listing trees fails", func(t *testing.T) {
		s, repo := newServer(t)
		repo.EXPECT().GetEstate(gomock.Any(), gomock.Any()).Return(estate5x1, nil)
		repo.EXPECT().ListTrees(gomock.Any(), gomock.Any()).Return(repository.ListTreesOutput{}, errDB)

		rec := call(t, http.MethodGet, ``, getPlan(s, nil))
		require.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}
