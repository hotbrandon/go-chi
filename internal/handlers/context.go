package handlers

import (
	"context"

	"github.com/hotbrandon/go-chi/internal/repo"
)

type contextKey string

const (
	RepoContextKey contextKey = "repository"
	DBIDContextKey contextKey = "database_id"
)

func GetRepo(ctx context.Context) (*repo.Repository, bool) {
	repo, ok := ctx.Value(RepoContextKey).(*repo.Repository)
	return repo, ok
}

func GetDBID(ctx context.Context) (string, bool) {
	dbID, ok := ctx.Value(DBIDContextKey).(string)
	return dbID, ok
}

func MustGetRepo(ctx context.Context) *repo.Repository {
	repo, ok := GetRepo(ctx)
	if !ok {
		panic("repository not found in context")
	}
	return repo
}
