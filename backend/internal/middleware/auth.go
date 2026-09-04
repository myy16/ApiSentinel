package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/apisentinel/apisentinel/internal/database"
	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type contextKey string

const (
	UserIDKey   contextKey = "userId"
	UserRoleKey contextKey = "userRole"
	OrgIDKey    contextKey = "orgId"
)

func Auth(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var tokenString string
			authHeader := r.Header.Get("Authorization")
			if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
				tokenString = strings.TrimPrefix(authHeader, "Bearer ")
			} else if cookie, err := r.Cookie("apisentinel_access_token"); err == nil && cookie.Value != "" {
				tokenString = cookie.Value
			} else if cookie, err := r.Cookie("auth_token"); err == nil && cookie.Value != "" {
				tokenString = cookie.Value
			} else if cookie, err := r.Cookie("access_token"); err == nil && cookie.Value != "" {
				tokenString = cookie.Value
			} else if queryToken := r.URL.Query().Get("token"); queryToken != "" {
				// EventSource / SSE fallback where browser cannot send custom headers
				tokenString = queryToken
			}

			if tokenString == "" {
				http.Error(w, `{"error":{"code":"UNAUTHORIZED","message":"Missing or invalid authorization token"}}`, http.StatusUnauthorized)
				return
			}

			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return []byte(jwtSecret), nil
			})

			if err != nil || !token.Valid {
				http.Error(w, `{"error":{"code":"UNAUTHORIZED","message":"Invalid or expired access token"}}`, http.StatusUnauthorized)
				return
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				http.Error(w, `{"error":{"code":"UNAUTHORIZED","message":"Invalid token claims"}}`, http.StatusUnauthorized)
				return
			}

			userId, _ := claims["userId"].(string)
			ctx := context.WithValue(r.Context(), UserIDKey, userId)

			// Extract organization ID from header, query param, or cookie
			orgId := r.Header.Get("x-organization-id")
			if orgId == "" {
				orgId = r.URL.Query().Get("orgId")
				if orgId == "" {
					orgId = r.URL.Query().Get("organizationId")
					if orgId == "" {
						if orgCookie, err := r.Cookie("x-organization-id"); err == nil && orgCookie.Value != "" {
							orgId = orgCookie.Value
						} else if orgCookie, err := r.Cookie("organizationId"); err == nil && orgCookie.Value != "" {
							orgId = orgCookie.Value
						}
					}
				}
			}
			if orgId != "" {
				ctx = context.WithValue(ctx, OrgIDKey, orgId)
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireTenant enforces that x-organization-id header is present AND
// the authenticated user actually has a membership in that organization.
func RequireTenant(queries *database.Queries) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			orgIdRaw := r.Context().Value(OrgIDKey)
			userIdRaw := r.Context().Value(UserIDKey)

			if orgIdRaw == nil || orgIdRaw == "" {
				http.Error(w, `{"error":{"code":"TENANT_REQUIRED","message":"Header 'x-organization-id' is required"}}`, http.StatusBadRequest)
				return
			}

			orgIdStr, _ := orgIdRaw.(string)
			userIdStr, _ := userIdRaw.(string)

			if orgIdStr == "" || userIdStr == "" {
				http.Error(w, `{"error":{"code":"TENANT_REQUIRED","message":"Organization and User context required"}}`, http.StatusBadRequest)
				return
			}

			// Parse UUIDs
			parsedOrgId, err := uuid.Parse(orgIdStr)
			if err != nil {
				http.Error(w, `{"error":{"code":"INVALID_ORG_ID","message":"Invalid organization ID format"}}`, http.StatusBadRequest)
				return
			}
			parsedUserId, err := uuid.Parse(userIdStr)
			if err != nil {
				http.Error(w, `{"error":{"code":"INVALID_USER_ID","message":"Invalid user ID format"}}`, http.StatusBadRequest)
				return
			}

			var pgOrgId pgtype.UUID
			copy(pgOrgId.Bytes[:], parsedOrgId[:])
			pgOrgId.Valid = true

			var pgUserId pgtype.UUID
			copy(pgUserId.Bytes[:], parsedUserId[:])
			pgUserId.Valid = true

			// Verify membership exists in database
			membership, err := queries.GetMembership(r.Context(), database.GetMembershipParams{
				OrganizationID: pgOrgId,
				UserID:         pgUserId,
			})
			if err != nil {
				http.Error(w, `{"error":{"code":"FORBIDDEN","message":"You do not have access to this organization"}}`, http.StatusForbidden)
				return
			}

			// Inject verified role into context
			ctx := context.WithValue(r.Context(), UserRoleKey, membership.Role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireProjectOwnership verifies that the route project belongs to the active tenant.
func RequireProjectOwnership(queries *database.Queries, param string) func(http.Handler) http.Handler {
	return requireResourceOwnership(param, func(ctx context.Context, id, orgID pgtype.UUID) error {
		_, err := queries.VerifyProjectOwnership(ctx, database.VerifyProjectOwnershipParams{ID: id, OrganizationID: orgID})
		return err
	})
}

// RequireEndpointOwnership verifies endpoint -> project -> organization ownership.
func RequireEndpointOwnership(queries *database.Queries, param string) func(http.Handler) http.Handler {
	return requireResourceOwnership(param, func(ctx context.Context, id, orgID pgtype.UUID) error {
		_, err := queries.GetEndpointWithOwnership(ctx, database.GetEndpointWithOwnershipParams{ID: id, OrganizationID: orgID})
		return err
	})
}

// RequireRequestOwnership verifies captured request -> endpoint -> project -> organization ownership.
func RequireRequestOwnership(queries *database.Queries, param string) func(http.Handler) http.Handler {
	return requireResourceOwnership(param, func(ctx context.Context, id, orgID pgtype.UUID) error {
		_, err := queries.VerifyRequestOwnership(ctx, database.VerifyRequestOwnershipParams{ID: id, OrganizationID: orgID})
		return err
	})
}

// RequireAlertChannelOwnership verifies alert channel -> project -> organization ownership.
func RequireAlertChannelOwnership(queries *database.Queries, param string) func(http.Handler) http.Handler {
	return requireResourceOwnership(param, func(ctx context.Context, id, orgID pgtype.UUID) error {
		_, err := queries.VerifyAlertChannelOwnership(ctx, database.VerifyAlertChannelOwnershipParams{ID: id, OrganizationID: orgID})
		return err
	})
}

// RequireDLQRecordOwnership verifies DLQ record -> endpoint -> project -> organization ownership.
func RequireDLQRecordOwnership(queries *database.Queries, param string) func(http.Handler) http.Handler {
	return requireResourceOwnership(param, func(ctx context.Context, id, orgID pgtype.UUID) error {
		_, err := queries.VerifyDLQRecordOwnership(ctx, database.VerifyDLQRecordOwnershipParams{ID: id, OrganizationID: orgID})
		return err
	})
}

// RequireDeliveryJobOwnership verifies delivery job -> endpoint -> project -> organization ownership.
func RequireDeliveryJobOwnership(queries *database.Queries, param string) func(http.Handler) http.Handler {
	return requireResourceOwnership(param, func(ctx context.Context, id, orgID pgtype.UUID) error {
		_, err := queries.VerifyDeliveryJobOwnership(ctx, database.VerifyDeliveryJobOwnershipParams{ID: id, OrganizationID: orgID})
		return err
	})
}

// RequireReplayJobOwnership verifies replay job -> captured_request -> endpoint -> project -> organization ownership.
func RequireReplayJobOwnership(queries *database.Queries, param string) func(http.Handler) http.Handler {
	return requireResourceOwnership(param, func(ctx context.Context, id, orgID pgtype.UUID) error {
		_, err := queries.VerifyReplayJobOwnership(ctx, database.VerifyReplayJobOwnershipParams{ID: id, OrganizationID: orgID})
		return err
	})
}

// RequireTestSuiteOwnership verifies test suite -> project -> organization ownership.
func RequireTestSuiteOwnership(queries *database.Queries, param string) func(http.Handler) http.Handler {
	return requireResourceOwnership(param, func(ctx context.Context, id, orgID pgtype.UUID) error {
		_, err := queries.VerifyTestSuiteOwnership(ctx, database.VerifyTestSuiteOwnershipParams{ID: id, OrganizationID: orgID})
		return err
	})
}

type ownershipCheck func(context.Context, pgtype.UUID, pgtype.UUID) error

func requireResourceOwnership(param string, check ownershipCheck) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resourceID, err := uuid.Parse(chi.URLParam(r, param))
			if err != nil {
				http.Error(w, `{"error":{"code":"INVALID_RESOURCE_ID","message":"Invalid resource ID format"}}`, http.StatusBadRequest)
				return
			}

			orgIDRaw, ok := r.Context().Value(OrgIDKey).(string)
			if !ok {
				http.Error(w, `{"error":{"code":"TENANT_REQUIRED","message":"Organization context required"}}`, http.StatusBadRequest)
				return
			}
			orgID, err := uuid.Parse(orgIDRaw)
			if err != nil {
				http.Error(w, `{"error":{"code":"INVALID_ORG_ID","message":"Invalid organization ID format"}}`, http.StatusBadRequest)
				return
			}

			if err := check(r.Context(), uuidToPG(resourceID), uuidToPG(orgID)); err != nil {
				http.Error(w, `{"error":{"code":"FORBIDDEN","message":"You do not have access to this resource"}}`, http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func uuidToPG(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

func roleRank(role string) int {
	switch strings.ToUpper(role) {
	case "OWNER", "ADMIN":
		return 3
	case "DEVELOPER", "MEMBER":
		return 2
	case "VIEWER":
		return 1
	default:
		return 0
	}
}

// RequireRole verifies that the tenant-authenticated user has at least minRole rank (#4.1)
func RequireRole(minRole string) func(http.Handler) http.Handler {
	requiredRank := roleRank(minRole)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userRoleRaw := r.Context().Value(UserRoleKey)
			if userRoleRaw == nil {
				http.Error(w, `{"error":{"code":"FORBIDDEN","message":"User role not resolved in tenant context"}}`, http.StatusForbidden)
				return
			}

			userRole, ok := userRoleRaw.(string)
			if !ok || roleRank(userRole) < requiredRank {
				http.Error(w, `{"error":{"code":"FORBIDDEN","message":"Insufficient permissions: requires role `+strings.ToUpper(minRole)+` or higher"}}`, http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// GetUserID retrieves the authenticated user's UUID from request context.
func GetUserID(ctx context.Context) pgtype.UUID {
	if raw, ok := ctx.Value(UserIDKey).(string); ok {
		if u, err := uuid.Parse(raw); err == nil {
			return pgtype.UUID{Bytes: u, Valid: true}
		}
	}
	return pgtype.UUID{Valid: false}
}

// GetOrganizationID retrieves the resolved organization UUID from request context.
func GetOrganizationID(ctx context.Context) pgtype.UUID {
	if raw, ok := ctx.Value(OrgIDKey).(string); ok {
		if u, err := uuid.Parse(raw); err == nil {
			return pgtype.UUID{Bytes: u, Valid: true}
		}
	}
	return pgtype.UUID{Valid: false}
}

// GetRole retrieves the tenant user's role (OWNER, DEVELOPER, VIEWER) from context.
func GetRole(ctx context.Context) string {
	if raw, ok := ctx.Value(UserRoleKey).(string); ok {
		return strings.ToUpper(raw)
	}
	return ""
}

