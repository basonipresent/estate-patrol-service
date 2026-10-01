package handler

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/SawitProRecruitment/UserService/droneplan"
	"github.com/SawitProRecruitment/UserService/generated"
	"github.com/SawitProRecruitment/UserService/repository"
	"github.com/SawitProRecruitment/UserService/treestats"
	"github.com/labstack/echo/v4"
)

// Limits from the assignment; api.yml documents the same values.
const (
	maxEstateSide = 50000
	minTreeHeight = 1
	maxTreeHeight = 30
)

// (POST /estate)
func (s *Server) CreateEstate(ctx echo.Context) error {
	var req generated.CreateEstateRequest
	if err := bindJSON(ctx, &req, "request body must be JSON with integer length and width"); err != nil {
		return err
	}
	if !inRange(req.Length, 1, maxEstateSide) || !inRange(req.Width, 1, maxEstateSide) {
		return badRequest(fmt.Sprintf("length and width must be between 1 and %d", maxEstateSide))
	}

	out, err := s.Repository.CreateEstate(ctx.Request().Context(), repository.CreateEstateInput{
		Length: req.Length,
		Width:  req.Width,
	})
	if err != nil {
		return err
	}
	return ctx.JSON(http.StatusCreated, generated.IdResponse{Id: out.Id})
}

// (POST /estate/{id}/tree)
func (s *Server) CreateTree(ctx echo.Context, id generated.EstateId) error {
	var req generated.CreateTreeRequest
	if err := bindJSON(ctx, &req, "request body must be JSON with integer x, y and height"); err != nil {
		return err
	}
	if !inRange(req.Height, minTreeHeight, maxTreeHeight) {
		return badRequest(fmt.Sprintf("height must be between %d and %d", minTreeHeight, maxTreeHeight))
	}

	estate, err := s.Repository.GetEstate(ctx.Request().Context(), repository.GetEstateInput{Id: id})
	if err != nil {
		return mapRepoError(err)
	}
	// Estates are immutable, so this bounds check cannot go stale before the insert.
	if !inRange(req.X, 1, estate.Length) || !inRange(req.Y, 1, estate.Width) {
		return badRequest(fmt.Sprintf("plot (%d, %d) is outside the %dx%d estate", req.X, req.Y, estate.Length, estate.Width))
	}

	out, err := s.Repository.CreateTree(ctx.Request().Context(), repository.CreateTreeInput{
		EstateId: id,
		X:        req.X,
		Y:        req.Y,
		Height:   req.Height,
	})
	if err != nil {
		return mapRepoError(err)
	}
	return ctx.JSON(http.StatusCreated, generated.IdResponse{Id: out.Id})
}

// (GET /estate/{id}/stats)
func (s *Server) GetEstateStats(ctx echo.Context, id generated.EstateId) error {
	if _, err := s.Repository.GetEstate(ctx.Request().Context(), repository.GetEstateInput{Id: id}); err != nil {
		return mapRepoError(err)
	}
	out, err := s.Repository.GetHeightHistogram(ctx.Request().Context(), repository.GetHeightHistogramInput{EstateId: id})
	if err != nil {
		return err
	}

	stats := treestats.FromHistogram(out.Histogram)
	return ctx.JSON(http.StatusOK, generated.EstateStatsResponse{
		Count:  stats.Count,
		Min:    stats.Min,
		Max:    stats.Max,
		Median: stats.Median,
	})
}

// (GET /estate/{id}/drone-plan)
func (s *Server) GetDronePlan(ctx echo.Context, id generated.EstateId, params generated.GetDronePlanParams) error {
	if params.MaxDistance != nil && *params.MaxDistance < 1 {
		return badRequest("max_distance must be a positive integer")
	}

	estate, err := s.Repository.GetEstate(ctx.Request().Context(), repository.GetEstateInput{Id: id})
	if err != nil {
		return mapRepoError(err)
	}
	out, err := s.Repository.ListTrees(ctx.Request().Context(), repository.ListTreesInput{EstateId: id})
	if err != nil {
		return err
	}

	resp := generated.DronePlanResponse{
		Distance: droneplan.Distance(estate.Length, estate.Width, out.Trees),
	}
	if params.MaxDistance != nil {
		// Distance flown on the main battery; the forced-landing descent uses the backup.
		resp.Distance = min(resp.Distance, *params.MaxDistance)
		x, y := droneplan.Rest(estate.Length, estate.Width, out.Trees, *params.MaxDistance)
		resp.Rest = &generated.Plot{X: x, Y: y}
	}
	return ctx.JSON(http.StatusOK, resp)
}

func mapRepoError(err error) error {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		return errEstateNotFound
	case errors.Is(err, repository.ErrPlotOccupied):
		return badRequest("plot already has a tree")
	default:
		return err
	}
}

func inRange(v, lo, hi int) bool {
	return v >= lo && v <= hi
}
