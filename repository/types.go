// This file contains types that are used in the repository layer.
package repository

import (
	"errors"

	"github.com/SawitProRecruitment/UserService/droneplan"
	"github.com/SawitProRecruitment/UserService/treestats"
	"github.com/google/uuid"
)

var (
	ErrNotFound     = errors.New("not found")
	ErrPlotOccupied = errors.New("plot already has a tree")
)

type CreateEstateInput struct {
	Length int
	Width  int
}

type CreateEstateOutput struct {
	Id uuid.UUID
}

type GetEstateInput struct {
	Id uuid.UUID
}

type GetEstateOutput struct {
	Id     uuid.UUID
	Length int
	Width  int
}

type CreateTreeInput struct {
	EstateId uuid.UUID
	X        int
	Y        int
	Height   int
}

type CreateTreeOutput struct {
	Id uuid.UUID
}

type ListTreesInput struct {
	EstateId uuid.UUID
}

type ListTreesOutput struct {
	Trees []droneplan.Tree
}

type GetHeightHistogramInput struct {
	EstateId uuid.UUID
}

type GetHeightHistogramOutput struct {
	Histogram []treestats.HeightCount
}
