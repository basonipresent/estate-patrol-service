// This file contains the interfaces for the repository layer.
// The repository layer is responsible for interacting with the database.
// For testing purpose we will generate mock implementations of these
// interfaces using mockgen. See the Makefile for more information.
package repository

import "context"

type RepositoryInterface interface {
	CreateEstate(ctx context.Context, input CreateEstateInput) (output CreateEstateOutput, err error)
	GetEstate(ctx context.Context, input GetEstateInput) (output GetEstateOutput, err error)
	CreateTree(ctx context.Context, input CreateTreeInput) (output CreateTreeOutput, err error)
	ListTrees(ctx context.Context, input ListTreesInput) (output ListTreesOutput, err error)
	GetHeightHistogram(ctx context.Context, input GetHeightHistogramInput) (output GetHeightHistogramOutput, err error)
}
