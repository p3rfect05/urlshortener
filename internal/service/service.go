package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v4"
	"golang.org/x/crypto/bcrypt"
	"math/rand"
	"sync"
	"time"
	"urleater/internal/repository/postgresDB"
)

type Storage interface {
	CreateUser(ctx context.Context, email string, password string) error
	ChangePassword(ctx context.Context, email string, password string) error
	GetUser(ctx context.Context, email string) (*postgresDB.User, error)
	CreateShortLink(ctx context.Context, shortLink string, longLink string, userID string) (*postgresDB.Link, error)
	GetShortLink(ctx context.Context, shortLink string) (*postgresDB.Link, error)
	DeleteShortLink(ctx context.Context, shortLink string) error
	ExtendShortLink(ctx context.Context, shortLink string, expiresAt time.Time) (*postgresDB.Link, error)
	GetAllUserShortLinks(ctx context.Context, email string) ([]postgresDB.Link, error)
	UpdateUserLinks(ctx context.Context, email string, urlsDelta int) (*postgresDB.User, error)
	GetSubscriptions(ctx context.Context) ([]postgresDB.Subscription, error)
}

var mutex = &sync.Mutex{}

type Service struct {
	storage Storage
}

var reservedNames = []string{
	"register",
	"login",
	"logout",
	"create_link",
	"buy",
	"subscriptions",
}

func New(storage Storage) *Service {
	return &Service{
		storage: storage,
	}
}

func (s *Service) LoginUser(ctx context.Context, email string, password string) error {
	user, err := s.storage.GetUser(ctx, email)
	if err != nil {
		return fmt.Errorf("LoginUser: could not get user %w", err)
	}

	if err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return fmt.Errorf("LoginUser: invalid password")
	}
	return nil
}

func (s *Service) RegisterUser(ctx context.Context, email string, password string) error {
	_, err := s.storage.GetUser(ctx, email)

	switch {
	case errors.Is(err, pgx.ErrNoRows):

	case err != nil:
		return fmt.Errorf("RegisterUser: could not get user %w", err)
	default:
		return fmt.Errorf("RegisterUser: user already exists")
	}

	err = s.storage.CreateUser(ctx, email, password)

	if err != nil {
		return fmt.Errorf("RegisterUser: could not create user %w", err)
	}

	return nil
}
func (s *Service) CreateShortLink(ctx context.Context, alias string, longLink string, userEmail string) (*postgresDB.Link, error) {

	var shortLink string

	if alias != "" {
		shortLink = alias
	} else {
		var err error

	forLoop:
		for i := 0; i < 10; i++ { // генерируем ссылки, пока такие существуют
			shortLink = GenerateShortLink()

			_, err = s.storage.GetShortLink(ctx, shortLink)

			switch {
			case errors.Is(err, pgx.ErrNoRows):
				break forLoop

			case err != nil:
				return nil, fmt.Errorf("failed to check if short link exists: %w", err)

			case i == 9:
				return nil, fmt.Errorf("could not generate short link in 10 tries")

			}
		}
	}

	for _, val := range reservedNames {
		if val == shortLink {
			return nil, fmt.Errorf("short link %s is not available", shortLink)
		}
	}

	_, err := s.storage.GetShortLink(ctx, shortLink)

	switch {
	case errors.Is(err, pgx.ErrNoRows):

	case err != nil:
		return nil, fmt.Errorf("CreateShortLink: error while getting shortlink: %#v", err)
	default:
		return nil, fmt.Errorf("CreateShortLink: shortlink already exists")
	}

	link, err := s.storage.CreateShortLink(ctx, shortLink, longLink, userEmail)

	if err != nil {
		return nil, fmt.Errorf("CreateShortLink: error while creating a short link %s", shortLink)
	}
	return link, nil
}

func (s *Service) GetSubscriptions(ctx context.Context) ([]postgresDB.Subscription, error) {
	subs, err := s.storage.GetSubscriptions(ctx)

	if err != nil {
		return nil, fmt.Errorf("GetSubscriptions: could not get subscriptions %w", err)
	}

	return subs, nil
}

func (s *Service) GetUser(ctx context.Context, email string) (*postgresDB.User, error) {
	user, err := s.storage.GetUser(ctx, email)

	if err != nil {
		return nil, fmt.Errorf("GetUser: could not get user %w", err)
	}

	return user, nil
}

func (s *Service) GetAllUserShortLinks(ctx context.Context, email string) ([]postgresDB.Link, *postgresDB.User, error) {
	user, err := s.storage.GetUser(ctx, email)

	if err != nil {
		return nil, nil, fmt.Errorf("GetAllUserShortLinks: error while getting user %s: %w", email, err)
	}

	links, err := s.storage.GetAllUserShortLinks(ctx, email)

	switch {
	case errors.Is(err, pgx.ErrNoRows):

	default:
		return nil, nil, fmt.Errorf("GetAllUsersShortLinks: error while getting all user's %s shortlinks: %w", email, err)
	}

	return links, user, nil

}

func (s *Service) UpdateUserShortLinks(ctx context.Context, email string, deltaLinks int) (*postgresDB.User, error) {
	user, err := s.storage.UpdateUserLinks(ctx, email, deltaLinks)

	if err != nil {
		return nil, fmt.Errorf("UpdateUserShortLinks: error while updating user's %s shortlinks: %w by %d", email, err, deltaLinks)
	}

	return user, nil
}

func (s *Service) GetShortLink(ctx context.Context, shortLink string) (*postgresDB.Link, error) {
	link, err := s.storage.GetShortLink(ctx, shortLink)
	if err != nil {
		return nil, fmt.Errorf("GetShortLink: error while getting short link %s: %w", shortLink, err)
	}

	return link, nil

}

const letterBytes = "1234567890abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func GenerateShortLink() string {
	mutex.Lock()
	defer mutex.Unlock()
	source := rand.NewSource(time.Now().UnixNano())

	res := make([]byte, 8)

	for i := range res {
		res[i] = letterBytes[source.Int63()%int64(len(letterBytes))]
	}

	return string(res)

}
