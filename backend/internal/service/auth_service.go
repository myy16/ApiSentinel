package service

import (
	"context"
	"errors"
	"time"

	"github.com/apisentinel/apisentinel/internal/database"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	queries   *database.Queries
	jwtSecret string
}

func NewAuthService(queries *database.Queries, jwtSecret string) *AuthService {
	return &AuthService{
		queries:   queries,
		jwtSecret: jwtSecret,
	}
}

type AuthResponse struct {
	AccessToken  string             `json:"accessToken"`
	RefreshToken string             `json:"refreshToken"`
	User         UserResponse       `json:"user"`
	Organization OrgResponse        `json:"organization"`
}

type UserResponse struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	CreatedAt string `json:"createdAt"`
}

type OrgResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Role      string `json:"role"`
	CreatedAt string `json:"createdAt"`
}

func (s *AuthService) Register(ctx context.Context, email, password, orgName string) (*AuthResponse, error) {
	// Check if user exists
	_, err := s.queries.GetUserByEmail(ctx, email)
	if err == nil {
		return nil, errors.New("bu e-posta adresi zaten kullanımda")
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	if err != nil {
		return nil, err
	}

	user, err := s.queries.CreateUser(ctx, database.CreateUserParams{
		Email:        email,
		PasswordHash: string(hashed),
	})
	if err != nil {
		return nil, err
	}

	org, err := s.queries.CreateOrganization(ctx, orgName)
	if err != nil {
		return nil, err
	}

	membership, err := s.queries.CreateMembership(ctx, database.CreateMembershipParams{
		OrganizationID: org.ID,
		UserID:         user.ID,
		Role:           "OWNER",
	})
	if err != nil {
		return nil, err
	}

	userIdStr := uuid.UUID(user.ID.Bytes).String()
	orgIdStr := uuid.UUID(org.ID.Bytes).String()

	accessToken, err := s.generateToken(userIdStr, orgIdStr, "OWNER", 24*time.Hour)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.generateToken(userIdStr, orgIdStr, "OWNER", 7*24*time.Hour)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User: UserResponse{
			ID:        userIdStr,
			Email:     user.Email,
			CreatedAt: user.CreatedAt.Time.Format(time.RFC3339),
		},
		Organization: OrgResponse{
			ID:        orgIdStr,
			Name:      org.Name,
			Role:      membership.Role,
			CreatedAt: org.CreatedAt.Time.Format(time.RFC3339),
		},
	}, nil
}

func (s *AuthService) Login(ctx context.Context, email, password string) (*AuthResponse, error) {
	user, err := s.queries.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, errors.New("geçersiz e-posta veya şifre")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, errors.New("geçersiz e-posta veya şifre")
	}

	memberships, err := s.queries.ListUserMemberships(ctx, user.ID)
	if err != nil || len(memberships) == 0 {
		return nil, errors.New("kullanıcıya ait organizasyon bulunamadı")
	}

	firstOrg := memberships[0]
	userIdStr := uuid.UUID(user.ID.Bytes).String()
	orgIdStr := uuid.UUID(firstOrg.OrganizationID.Bytes).String()

	accessToken, err := s.generateToken(userIdStr, orgIdStr, firstOrg.Role, 24*time.Hour)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.generateToken(userIdStr, orgIdStr, firstOrg.Role, 7*24*time.Hour)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User: UserResponse{
			ID:        userIdStr,
			Email:     user.Email,
			CreatedAt: user.CreatedAt.Time.Format(time.RFC3339),
		},
		Organization: OrgResponse{
			ID:        orgIdStr,
			Name:      firstOrg.OrganizationName,
			Role:      firstOrg.Role,
			CreatedAt: firstOrg.CreatedAt.Time.Format(time.RFC3339),
		},
	}, nil
}

func (s *AuthService) GetMe(ctx context.Context, userId string) (*UserResponse, []OrgResponse, error) {
	uid, err := uuid.Parse(userId)
	if err != nil {
		return nil, nil, err
	}

	var pgUid pgtype.UUID
	copy(pgUid.Bytes[:], uid[:])
	pgUid.Valid = true

	user, err := s.queries.GetUserByID(ctx, pgUid)
	if err != nil {
		return nil, nil, err
	}

	memberships, err := s.queries.ListUserMemberships(ctx, pgUid)
	if err != nil {
		return nil, nil, err
	}

	var orgs []OrgResponse
	for _, m := range memberships {
		orgs = append(orgs, OrgResponse{
			ID:        uuid.UUID(m.OrganizationID.Bytes).String(),
			Name:      m.OrganizationName,
			Role:      m.Role,
			CreatedAt: m.CreatedAt.Time.Format(time.RFC3339),
		})
	}

	return &UserResponse{
		ID:        userId,
		Email:     user.Email,
		CreatedAt: user.CreatedAt.Time.Format(time.RFC3339),
	}, orgs, nil
}

type RefreshResponse struct {
	AccessToken string `json:"accessToken"`
}

// RefreshAccessToken validates a refresh token and issues a new short-lived access token (#17).
func (s *AuthService) RefreshAccessToken(ctx context.Context, refreshToken string) (*RefreshResponse, error) {
	token, err := jwt.Parse(refreshToken, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("beklenmeyen imza yöntemi")
		}
		return []byte(s.jwtSecret), nil
	})
	if err != nil || !token.Valid {
		return nil, errors.New("geçersiz veya süresi dolmuş refresh token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("token claims okunamadı")
	}

	// Verify this is actually a refresh token (not an access token being reused)
	if tokenType, exists := claims["tokenType"]; exists {
		if tokenType != "refresh" {
			return nil, errors.New("bu bir refresh token değil")
		}
	}

	userId, _ := claims["userId"].(string)
	orgId, _ := claims["organizationId"].(string)
	role, _ := claims["role"].(string)

	if userId == "" || orgId == "" {
		return nil, errors.New("eksik token bilgileri")
	}

	accessToken, err := s.generateToken(userId, orgId, role, 1*time.Hour)
	if err != nil {
		return nil, err
	}

	return &RefreshResponse{AccessToken: accessToken}, nil
}

func (s *AuthService) generateToken(userId, orgId, role string, ttl time.Duration) (string, error) {
	claims := jwt.MapClaims{
		"userId":         userId,
		"organizationId": orgId,
		"role":           role,
		"exp":            time.Now().Add(ttl).Unix(),
		"iat":            time.Now().Unix(),
	}

	// Tag refresh tokens so they can be distinguished from access tokens
	if ttl > 24*time.Hour {
		claims["tokenType"] = "refresh"
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}
