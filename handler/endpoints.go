package handler

import (
	"net/http"

	"github.com/SawitProRecruitment/UserService/generated"
	"github.com/labstack/echo/v4"
)

// TODO(phase 5): the handlers below are stubs. They return 501 so an
// unimplemented endpoint can never be mistaken for a working one.

// (POST /estate)
func (s *Server) CreateEstate(ctx echo.Context) error {
	return notImplemented(ctx)
}

// (POST /estate/{id}/tree)
func (s *Server) CreateTree(ctx echo.Context, id generated.EstateId) error {
	return notImplemented(ctx)
}

// (GET /estate/{id}/stats)
func (s *Server) GetEstateStats(ctx echo.Context, id generated.EstateId) error {
	return notImplemented(ctx)
}

// (GET /estate/{id}/drone-plan)
func (s *Server) GetDronePlan(ctx echo.Context, id generated.EstateId, params generated.GetDronePlanParams) error {
	return notImplemented(ctx)
}

func notImplemented(ctx echo.Context) error {
	return ctx.JSON(http.StatusNotImplemented, generated.ErrorResponse{Message: "not implemented"})
}
