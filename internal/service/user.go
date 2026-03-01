package service

import (
	"context"

	"github.com/ianzepp/gofolio-api/internal/model"
	"github.com/ianzepp/gofolio-api/internal/repository"
	"github.com/jmoiron/sqlx"
)

// UserService manages user accounts, authentication tokens, and per-user
// settings. It is deliberately slim: heavy portfolio logic lives in higher-level
// services (order, account) that join on userID; UserService only handles
// identity and preferences.
type UserService struct {
	db *sqlx.DB
}

// NewUserService constructs a UserService backed by the given pool.
func NewUserService(db *sqlx.DB) *UserService {
	return &UserService{db: db}
}

// CreateUser persists a new user record and returns it with any DB-generated
// fields (e.g. id, createdAt) populated.
func (s *UserService) CreateUser(ctx context.Context, user model.User) (model.User, error) {
	repo := repository.NewUserRepository(s.db)
	return repo.Create(ctx, user)
}

func (s *UserService) GetUser(ctx context.Context, id string) (model.User, error) {
	repo := repository.NewUserRepository(s.db)
	return repo.FindByID(ctx, id)
}

func (s *UserService) GetUsers(ctx context.Context) ([]model.User, error) {
	repo := repository.NewUserRepository(s.db)
	return repo.FindAll(ctx)
}

func (s *UserService) DeleteUser(ctx context.Context, id string) error {
	repo := repository.NewUserRepository(s.db)
	return repo.Delete(ctx, id)
}

// GetUserByAccessToken looks up a user by their hashed API access token, which
// is the credential used by the bearer-token auth flow. The token is stored
// hashed in the DB so that a compromised DB does not expose raw tokens.
func (s *UserService) GetUserByAccessToken(ctx context.Context, accessToken string) (model.User, error) {
	repo := repository.NewUserRepository(s.db)
	return repo.FindByAccessToken(ctx, accessToken)
}

// UpdateAccessToken replaces a user's stored hashed access token. Callers are
// responsible for hashing before calling this method — the service layer does
// not perform hashing so that the hashing algorithm can be chosen at the
// handler/auth layer without coupling it to the DB schema.
func (s *UserService) UpdateAccessToken(ctx context.Context, id string, hashedAccessToken string) error {
	repo := repository.NewUserRepository(s.db)
	return repo.UpdateAccessToken(ctx, id, hashedAccessToken)
}

func (s *UserService) GetSettings(ctx context.Context, userID string) (model.Settings, error) {
	repo := repository.NewUserRepository(s.db)
	return repo.FindSettings(ctx, userID)
}

// UpdateSettings upserts a user's settings record. Settings are stored as a
// separate row (not embedded in the User table) so they can be updated
// independently without touching the core user identity fields.
func (s *UserService) UpdateSettings(ctx context.Context, s2 model.Settings) (model.Settings, error) {
	repo := repository.NewUserRepository(s.db)
	return repo.UpsertSettings(ctx, s2)
}
