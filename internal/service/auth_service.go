package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/octguy/auth-sqlc/internal/model"
	"github.com/octguy/auth-sqlc/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrEmailTaken         = errors.New("email already taken")
)

type Claims struct {
	UserID uuid.UUID `json:"user_id"`
	jwt.RegisteredClaims
}

// AuthService is the contract for authentication business logic.
type AuthService interface {
	Register(ctx context.Context, req *model.RegisterRequest) (*model.AuthResponse, error)
	Login(ctx context.Context, req *model.LoginRequest) (*model.AuthResponse, error)
	ValidateToken(tokenStr string) (*Claims, error)
	GetProfile(ctx context.Context, userId uuid.UUID) (*model.User, error)
}

type authService struct {
	repo      repository.UserRepository
	jwtSecret []byte
	tokenTTL  time.Duration
}

// NewAuthService wires the service with its dependencies.
func NewAuthService(repo repository.UserRepository, jwtSecret string, tokenTTL time.Duration) AuthService {
	return &authService{
		repo:      repo,
		jwtSecret: []byte(jwtSecret),
		tokenTTL:  tokenTTL,
	}
}

func (s *authService) Register(ctx context.Context, req *model.RegisterRequest) (*model.AuthResponse, error) {
	log.Printf("[INFO] register attempt: email=%s username=%s", req.Email, req.Username)

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("[ERROR] register failed - hashing password: email=%s error=%v", req.Email, err)
		return nil, fmt.Errorf("hashing password: %w", err)
	}

	user := &model.User{
		Username: req.Username,
		Email:    req.Email,
		Password: string(hashed),
	}

	if err := s.repo.Create(ctx, user); err != nil {
		if errors.Is(err, repository.ErrEmailDuplicate) {
			log.Printf("[WARN] register failed - email already taken: email=%s", req.Email)
			return nil, ErrEmailTaken
		}
		log.Printf("[ERROR] register failed - creating user: email=%s error=%v", req.Email, err)
		return nil, fmt.Errorf("creating user: %w", err)
	}

	token, err := s.generateToken(user.ID)
	if err != nil {
		return nil, err
	}

	log.Printf("[INFO] register successful: user_id=%s email=%s", user.ID, user.Email)
	return &model.AuthResponse{Token: token, User: user}, nil
}

func (s *authService) Login(ctx context.Context, req *model.LoginRequest) (*model.AuthResponse, error) {
	log.Printf("[INFO] login attempt: email=%s", req.Email)

	user, err := s.repo.FindByEmail(ctx, req.Email)
	if err != nil {
		log.Printf("[WARN] login failed - user not found: email=%s", req.Email)
		// always return the same error to prevent user enumeration
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		log.Printf("[WARN] login failed - invalid password: email=%s", req.Email)
		return nil, ErrInvalidCredentials
	}

	// Login successful, generate JWT
	token, err := s.generateToken(user.ID)
	if err != nil {
		return nil, err
	}

	log.Printf("[INFO] login successful: user_id=%s email=%s", user.ID, user.Email)
	return &model.AuthResponse{Token: token, User: user}, nil
}

func (s *authService) ValidateToken(tokenStr string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.jwtSecret, nil
	})

	if err != nil || !token.Valid {
		return nil, errors.New("invalid or expired token")
	}

	return claims, nil
}

func (s *authService) GetProfile(ctx context.Context, userID uuid.UUID) (*model.User, error) {
	return s.repo.FindByID(ctx, userID)
}

func (s *authService) generateToken(userID uuid.UUID) (string, error) {
	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.tokenTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.jwtSecret)

	if err != nil {
		return "", fmt.Errorf("signing token: %w", err)
	}

	return signed, nil
}
