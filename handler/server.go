package handler

import (
	"github.com/SawitProRecruitment/UserService/generated"
	"github.com/SawitProRecruitment/UserService/repository"
)

// Compile-time check that Server implements the generated contract. It lives here,
// not only in cmd/, so a signature mismatch also fails `go test ./handler/...`.
var _ generated.ServerInterface = (*Server)(nil)

type Server struct {
	Repository repository.RepositoryInterface
}

type NewServerOptions struct {
	Repository repository.RepositoryInterface
}

func NewServer(opts NewServerOptions) *Server {
	return &Server{
		Repository: opts.Repository,
	}
}
