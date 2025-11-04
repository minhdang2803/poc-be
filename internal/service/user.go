package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"mmispoc/internal/repository"
)

var (
	usernamePattern = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
)

// ErrInvalidUsername indicates the supplied username failed validation.
var ErrInvalidUsername = errors.New("invalid username")

// ErrInvalidPassword indicates the supplied password failed validation.
var ErrInvalidPassword = errors.New("invalid password")

// ErrUsernameTaken is returned when trying to signup with an existing username.
var ErrUsernameTaken = errors.New("username already registered")

// ErrInvalidCredentials is returned when login fails.
var ErrInvalidCredentials = errors.New("invalid credentials")

// ErrInvalidRestaurantID indicates the supplied restaurant id is malformed.
var ErrInvalidRestaurantID = errors.New("invalid restaurant id")

// ErrRestaurantNotFound indicates the supplied restaurant cannot be located.
var ErrRestaurantNotFound = errors.New("restaurant not found")

// ErrInvalidToken indicates the supplied token could not be validated.
var ErrInvalidToken = errors.New("invalid token")

// ErrTokenExpired indicates the supplied token is expired.
var ErrTokenExpired = errors.New("token expired")

const defaultTokenTTL = 15 * time.Minute

// DefaultTokenTTL returns the default access token lifetime.
func DefaultTokenTTL() time.Duration {
	return defaultTokenTTL
}

// UserService orchestrates user related actions.
type UserService struct {
	repo           *repository.UserRepository
	restaurantRepo *repository.RestaurantRepository
	tokenSecret    []byte
	tokenTTL       time.Duration
}

// UserProfile describes the authenticated user response.
type UserProfile struct {
	ID             int64
	Username       string
	RestaurantID   int64
	RestaurantName string
	CreatedAt      time.Time
}

// NewUser constructs the service.
func NewUser(repo *repository.UserRepository, restaurantRepo *repository.RestaurantRepository, tokenSecret string, tokenTTL time.Duration) *UserService {
	if tokenTTL <= 0 {
		tokenTTL = defaultTokenTTL
	}
	if tokenSecret == "" {
		tokenSecret = "change-me"
	}
	return &UserService{
		repo:           repo,
		restaurantRepo: restaurantRepo,
		tokenSecret:    []byte(tokenSecret),
		tokenTTL:       tokenTTL,
	}
}

// SignUp validates input and persists a new user.
func (s *UserService) SignUp(ctx context.Context, username, password string, restaurantID int64) (*repository.User, error) {
	username = strings.TrimSpace(username)
	if !isValidUsername(username) {
		return nil, ErrInvalidUsername
	}

	if !isValidPassword(password) {
		return nil, ErrInvalidPassword
	}

	if restaurantID <= 0 {
		return nil, ErrInvalidRestaurantID
	}

	exists, err := s.restaurantRepo.Exists(ctx, restaurantID)
	if err != nil {
		return nil, fmt.Errorf("check restaurant: %w", err)
	}
	if !exists {
		return nil, ErrRestaurantNotFound
	}

	exists, err = s.repo.Exists(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("check username: %w", err)
	}
	if exists {
		return nil, ErrUsernameTaken
	}

	hashed := hashPassword(password)
	user, err := s.repo.Create(ctx, username, hashed, restaurantID)
	if err != nil {
		if errors.Is(err, repository.ErrConflict) {
			return nil, ErrUsernameTaken
		}
		return nil, fmt.Errorf("create user: %w", err)
	}

	return user, nil
}

// Authenticate validates credentials and issues a JWT access token.
func (s *UserService) Authenticate(ctx context.Context, username, password string) (string, error) {
	username = strings.TrimSpace(username)
	password = strings.TrimSpace(password)

	if !isValidUsername(username) || !isValidPassword(password) {
		return "", ErrInvalidCredentials
	}

	user, err := s.repo.GetByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return "", ErrInvalidCredentials
		}
		return "", fmt.Errorf("fetch user: %w", err)
	}

	if user.PasswordHash != hashPassword(password) {
		return "", ErrInvalidCredentials
	}

	token, err := s.generateToken(user)
	if err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}

	return token, nil
}

func isValidUsername(username string) bool {
	if len(username) < 3 || len(username) > 32 {
		return false
	}
	return usernamePattern.MatchString(username)
}

func isValidPassword(password string) bool {
	return len(strings.TrimSpace(password)) >= 8
}

func hashPassword(password string) string {
	sum := sha256.Sum256([]byte(password))
	return hex.EncodeToString(sum[:])
}

// GetProfile returns the profile for the supplied user id.
func (s *UserService) GetProfile(ctx context.Context, userID int64) (*UserProfile, error) {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrInvalidToken
		}
		return nil, fmt.Errorf("fetch user: %w", err)
	}

	profile := &UserProfile{
		ID:           user.ID,
		Username:     user.Username,
		RestaurantID: user.RestaurantID,
		CreatedAt:    user.CreatedAt,
	}

	if user.RestaurantID > 0 {
		name, err := s.restaurantRepo.GetName(ctx, user.RestaurantID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, ErrRestaurantNotFound
			}
			return nil, fmt.Errorf("fetch restaurant name: %w", err)
		}
		profile.RestaurantName = name
	}

	return profile, nil
}

// ValidateAccessToken verifies the supplied JWT access token and returns the authenticated user.
func (s *UserService) ValidateAccessToken(ctx context.Context, token string) (*repository.User, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, ErrInvalidToken
	}

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, ErrInvalidToken
	}

	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, ErrInvalidToken
	}

	unsigned := parts[0] + "." + parts[1]
	expectedMAC := hmac.New(sha256.New, s.tokenSecret)
	if _, err := expectedMAC.Write([]byte(unsigned)); err != nil {
		return nil, ErrInvalidToken
	}
	if !hmac.Equal(expectedMAC.Sum(nil), sig) {
		return nil, ErrInvalidToken
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, ErrInvalidToken
	}

	var claims struct {
		UserID int64  `json:"user_id"`
		Sub    string `json:"sub"`
		Issued int64  `json:"iat"`
		Exp    int64  `json:"exp"`
	}
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return nil, ErrInvalidToken
	}

	if claims.UserID == 0 && claims.Sub != "" {
		if parsed, parseErr := strconv.ParseInt(claims.Sub, 10, 64); parseErr == nil {
			claims.UserID = parsed
		}
	}
	if claims.UserID == 0 {
		return nil, ErrInvalidToken
	}

	if claims.Exp != 0 && time.Now().UTC().Unix() > claims.Exp {
		return nil, ErrTokenExpired
	}

	user, err := s.repo.GetByID(ctx, claims.UserID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrInvalidToken
		}
		return nil, fmt.Errorf("fetch user: %w", err)
	}

	return user, nil
}

func (s *UserService) generateToken(user *repository.User) (string, error) {
	if len(s.tokenSecret) == 0 {
		return "", errors.New("token secret not configured")
	}

	now := time.Now().UTC()
	exp := now.Add(s.tokenTTL)

	header := map[string]string{
		"alg": "HS256",
		"typ": "JWT",
	}
	claims := map[string]interface{}{
		"user_id":       user.ID,
		"restaurant_id": user.RestaurantID,
		"sub":           strconv.FormatInt(user.ID, 10),
		"iat":           now.Unix(),
		"exp":           exp.Unix(),
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", fmt.Errorf("marshal jwt header: %w", err)
	}

	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("marshal jwt claims: %w", err)
	}

	encode := func(data []byte) string {
		return base64.RawURLEncoding.EncodeToString(data)
	}

	unsigned := encode(headerJSON) + "." + encode(claimsJSON)

	mac := hmac.New(sha256.New, s.tokenSecret)
	if _, err := mac.Write([]byte(unsigned)); err != nil {
		return "", fmt.Errorf("sign jwt: %w", err)
	}
	signature := encode(mac.Sum(nil))

	return unsigned + "." + signature, nil
}
