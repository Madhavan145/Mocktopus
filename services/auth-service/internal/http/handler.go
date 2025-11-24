package http

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"moktopus/auth-service/internal/auth"
	"moktopus/auth-service/internal/config"
	"moktopus/auth-service/internal/db"
)

type Handler struct {
	repo   db.Repository
	tokens auth.TokenManager
	cfg    config.Config
}

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type refreshRequest struct {
	UserID       string `json:"userId"`
	RefreshToken string `json:"refreshToken"`
}

type forgotPasswordRequest struct {
	Email string `json:"email"`
}

type resetPasswordRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"newPassword"`
}

type forgotPasswordResponse struct {
	Status     string  `json:"status"`
	ResetToken *string `json:"resetToken,omitempty"`
}

type authResponse struct {
	UserID       string `json:"userId"`
	Email        string `json:"email"`
	OrgID        string `json:"orgId"`
	Role         string `json:"role"`
	IsAdmin      bool   `json:"isAdmin"`
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    int    `json:"expiresIn"`
}

func NewHandler(repo db.Repository, tokens auth.TokenManager, cfg config.Config) http.Handler {
	h := Handler{repo: repo, tokens: tokens, cfg: cfg}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", h.health)
	mux.HandleFunc("/api/auth/register", h.register)
	mux.HandleFunc("/api/v1/auth/register", h.registerV1)
	mux.HandleFunc("/api/auth/login", h.login)
	mux.HandleFunc("/api/v1/auth/login", h.loginV1)
	mux.HandleFunc("/api/auth/refresh", h.refresh)
	mux.HandleFunc("/api/v1/auth/refresh", h.refreshV1)
	mux.HandleFunc("/api/v1/auth/logout", h.logoutV1)
	mux.HandleFunc("/api/auth/password/forgot", h.forgotPassword)
	mux.HandleFunc("/api/v1/auth/password/forgot", h.forgotPasswordV1)
	mux.HandleFunc("/api/auth/password/reset", h.resetPassword)
	mux.HandleFunc("/api/v1/auth/password/reset", h.resetPasswordV1)
	mux.HandleFunc("/api/v1/auth/me", h.me)
	mux.HandleFunc("/api/v1/auth/password/change", h.changePasswordV1)

	// OAuth endpoints
	mux.HandleFunc("/api/v1/auth/oauth/google", h.oauthGoogleRedirect)
	mux.HandleFunc("/api/v1/auth/oauth/google/callback", h.oauthGoogleCallback)
	mux.HandleFunc("/api/v1/auth/oauth/github", h.oauthGithubRedirect)
	mux.HandleFunc("/api/v1/auth/oauth/github/callback", h.oauthGithubCallback)

	// Org management endpoints (authenticated, gateway injects X-User-ID/X-Org-ID)
	mux.HandleFunc("/api/v1/org", h.orgRouter)
	mux.HandleFunc("/api/v1/org/members", h.orgMembersRouter)
	mux.HandleFunc("/api/v1/org/members/role", h.updateMemberRole)
	mux.HandleFunc("/api/v1/org/invite", h.inviteMember)
	mux.HandleFunc("/api/v1/org/invite/accept", h.acceptInvite)
	mux.HandleFunc("/api/v1/org/invites", h.listInvites)
	mux.HandleFunc("/api/v1/org/list", h.listUserOrgs)
	mux.HandleFunc("/api/v1/org/switch", h.switchOrg)

	return withJSON(mux)
}

func (h Handler) health(w http.ResponseWriter, _ *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{"service": "auth-service", "status": "ok"})
}

func (h Handler) register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if req.Email == "" || !isStrongPassword(req.Password) {
		http.Error(w, "email and password must be at least 12 chars with upper/lower/number/symbol", http.StatusBadRequest)
		return
	}

	role := normalizeRole(req.Role)
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		http.Error(w, "unable to process password", http.StatusInternalServerError)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	user, err := h.repo.CreateUser(ctx, req.Email, hash, role)
	if err != nil {
		http.Error(w, "user may already exist", http.StatusConflict)
		return
	}

	accessToken, refreshToken, refreshExpiry, err := h.tokens.CreateTokenPair(user.ID, user.OrgID, user.Role, user.IsPlatformAdmin)
	if err != nil {
		http.Error(w, "unable to create token pair", http.StatusInternalServerError)
		return
	}

	_, err = h.repo.CreateSession(ctx, user.ID, auth.HashRefreshToken(refreshToken), refreshExpiry)
	if err != nil {
		http.Error(w, "unable to create session", http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusCreated, authResponse{
		UserID:       user.ID,
		Email:        user.Email,
		OrgID:        user.OrgID,
		Role:         user.Role,
		IsAdmin:      user.IsPlatformAdmin,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int(h.cfg.AccessTokenTTL.Seconds()),
	})
}

func (h Handler) registerV1(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondV1Error(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondV1Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "invalid payload")
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if req.Email == "" || !isStrongPassword(req.Password) {
		respondV1Error(w, http.StatusBadRequest, "INVALID_REGISTER_REQUEST", "email and password must be at least 12 chars with upper/lower/number/symbol")
		return
	}

	role := normalizeRole(req.Role)
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		respondV1Error(w, http.StatusInternalServerError, "PASSWORD_HASH_FAILED", "unable to process password")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	user, err := h.repo.CreateUser(ctx, req.Email, hash, role)
	if err != nil {
		respondV1Error(w, http.StatusConflict, "USER_EXISTS", "user may already exist")
		return
	}

	accessToken, refreshToken, refreshExpiry, err := h.tokens.CreateTokenPair(user.ID, user.OrgID, user.Role, user.IsPlatformAdmin)
	if err != nil {
		respondV1Error(w, http.StatusInternalServerError, "TOKEN_PAIR_FAILED", "unable to create token pair")
		return
	}

	_, err = h.repo.CreateSession(ctx, user.ID, auth.HashRefreshToken(refreshToken), refreshExpiry)
	if err != nil {
		respondV1Error(w, http.StatusInternalServerError, "SESSION_CREATE_FAILED", "unable to create session")
		return
	}

	respondV1Data(w, http.StatusCreated, authResponse{
		UserID:       user.ID,
		Email:        user.Email,
		OrgID:        user.OrgID,
		Role:         user.Role,
		IsAdmin:      user.IsPlatformAdmin,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int(h.cfg.AccessTokenTTL.Seconds()),
	})
}

func (h Handler) login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	user, err := h.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "invalid credentials", http.StatusUnauthorized)
			return
		}
		http.Error(w, "unable to read user", http.StatusInternalServerError)
		return
	}

	if !auth.VerifyPassword(req.Password, user.PasswordHash) {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	accessToken, refreshToken, refreshExpiry, err := h.tokens.CreateTokenPair(user.ID, user.OrgID, user.Role, user.IsPlatformAdmin)
	if err != nil {
		http.Error(w, "unable to create token pair", http.StatusInternalServerError)
		return
	}

	_, err = h.repo.CreateSession(ctx, user.ID, auth.HashRefreshToken(refreshToken), refreshExpiry)
	if err != nil {
		http.Error(w, "unable to create session", http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, authResponse{
		UserID:       user.ID,
		Email:        user.Email,
		OrgID:        user.OrgID,
		Role:         user.Role,
		IsAdmin:      user.IsPlatformAdmin,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int(h.cfg.AccessTokenTTL.Seconds()),
	})
}

func (h Handler) loginV1(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondV1Error(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondV1Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "invalid payload")
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	user, err := h.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			respondV1Error(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "invalid credentials")
			return
		}
		respondV1Error(w, http.StatusInternalServerError, "USER_READ_FAILED", "unable to read user")
		return
	}

	if !auth.VerifyPassword(req.Password, user.PasswordHash) {
		respondV1Error(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "invalid credentials")
		return
	}

	accessToken, refreshToken, refreshExpiry, err := h.tokens.CreateTokenPair(user.ID, user.OrgID, user.Role, user.IsPlatformAdmin)
	if err != nil {
		respondV1Error(w, http.StatusInternalServerError, "TOKEN_PAIR_FAILED", "unable to create token pair")
		return
	}

	_, err = h.repo.CreateSession(ctx, user.ID, auth.HashRefreshToken(refreshToken), refreshExpiry)
	if err != nil {
		respondV1Error(w, http.StatusInternalServerError, "SESSION_CREATE_FAILED", "unable to create session")
		return
	}

	respondV1Data(w, http.StatusOK, authResponse{
		UserID:       user.ID,
		Email:        user.Email,
		OrgID:        user.OrgID,
		Role:         user.Role,
		IsAdmin:      user.IsPlatformAdmin,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int(h.cfg.AccessTokenTTL.Seconds()),
	})
}

func (h Handler) refresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	ok, err := h.repo.ValidateSession(ctx, req.UserID, auth.HashRefreshToken(req.RefreshToken))
	if err != nil {
		http.Error(w, "unable to validate session", http.StatusInternalServerError)
		return
	}
	if !ok {
		http.Error(w, "invalid refresh token", http.StatusUnauthorized)
		return
	}

	user, err := h.repo.GetUserByID(ctx, req.UserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "user not found", http.StatusUnauthorized)
			return
		}
		http.Error(w, "unable to read user", http.StatusInternalServerError)
		return
	}
	role := user.Role

	accessToken, newRefreshToken, refreshExpiry, err := h.tokens.CreateTokenPair(req.UserID, user.OrgID, role, user.IsPlatformAdmin)
	if err != nil {
		http.Error(w, "unable to create token pair", http.StatusInternalServerError)
		return
	}

	_, _ = h.repo.CreateSession(ctx, req.UserID, auth.HashRefreshToken(newRefreshToken), refreshExpiry)

	respondJSON(w, http.StatusOK, authResponse{
		UserID:       req.UserID,
		Email:        user.Email,
		OrgID:        user.OrgID,
		Role:         role,
		IsAdmin:      user.IsPlatformAdmin,
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    int(h.cfg.AccessTokenTTL.Seconds()),
	})
}

func (h Handler) refreshV1(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondV1Error(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondV1Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "invalid payload")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	ok, err := h.repo.ValidateSession(ctx, req.UserID, auth.HashRefreshToken(req.RefreshToken))
	if err != nil {
		respondV1Error(w, http.StatusInternalServerError, "SESSION_VALIDATE_FAILED", "unable to validate session")
		return
	}
	if !ok {
		respondV1Error(w, http.StatusUnauthorized, "INVALID_REFRESH_TOKEN", "invalid refresh token")
		return
	}

	user, err := h.repo.GetUserByID(ctx, req.UserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			respondV1Error(w, http.StatusUnauthorized, "USER_NOT_FOUND", "user not found")
			return
		}
		respondV1Error(w, http.StatusInternalServerError, "USER_READ_FAILED", "unable to read user")
		return
	}
	role := user.Role

	accessToken, newRefreshToken, refreshExpiry, err := h.tokens.CreateTokenPair(req.UserID, user.OrgID, role, user.IsPlatformAdmin)
	if err != nil {
		respondV1Error(w, http.StatusInternalServerError, "TOKEN_PAIR_FAILED", "unable to create token pair")
		return
	}

	_, _ = h.repo.CreateSession(ctx, req.UserID, auth.HashRefreshToken(newRefreshToken), refreshExpiry)

	respondV1Data(w, http.StatusOK, authResponse{
		UserID:       req.UserID,
		Email:        user.Email,
		OrgID:        user.OrgID,
		Role:         role,
		IsAdmin:      user.IsPlatformAdmin,
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    int(h.cfg.AccessTokenTTL.Seconds()),
	})
}

func (h Handler) forgotPassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req forgotPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	response := h.createPasswordResetToken(r.Context(), req.Email)
	respondJSON(w, http.StatusAccepted, response)
}

func (h Handler) forgotPasswordV1(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondV1Error(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	var req forgotPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondV1Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "invalid payload")
		return
	}

	response := h.createPasswordResetToken(r.Context(), req.Email)
	respondV1Data(w, http.StatusAccepted, response)
}

func (h Handler) resetPassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := h.executePasswordReset(r); err != nil {
		http.Error(w, err.message, err.status)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "password reset complete"})
}

func (h Handler) resetPasswordV1(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondV1Error(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	if err := h.executePasswordReset(r); err != nil {
		respondV1Error(w, err.status, err.code, err.message)
		return
	}

	respondV1Data(w, http.StatusOK, map[string]string{"status": "password reset complete"})
}

type changePasswordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

func (h Handler) changePasswordV1(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondV1Error(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		respondV1Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}

	var req changePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondV1Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "invalid payload")
		return
	}

	if strings.TrimSpace(req.CurrentPassword) == "" {
		respondV1Error(w, http.StatusBadRequest, "CURRENT_PASSWORD_REQUIRED", "current password is required")
		return
	}
	if !isStrongPassword(req.NewPassword) {
		respondV1Error(w, http.StatusBadRequest, "WEAK_PASSWORD", "password must be at least 12 chars with upper/lower/number/symbol")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Fetch current password hash and verify
	user, err := h.repo.GetUserByID(ctx, userID)
	if err != nil {
		respondV1Error(w, http.StatusInternalServerError, "USER_LOOKUP_FAILED", "unable to verify user")
		return
	}

	if !auth.VerifyPassword(req.CurrentPassword, user.PasswordHash) {
		respondV1Error(w, http.StatusUnauthorized, "WRONG_PASSWORD", "current password is incorrect")
		return
	}

	// Hash new password and update
	newHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		respondV1Error(w, http.StatusInternalServerError, "PASSWORD_HASH_FAILED", "unable to process password")
		return
	}

	if err := h.repo.UpdateUserPassword(ctx, userID, newHash); err != nil {
		respondV1Error(w, http.StatusInternalServerError, "PASSWORD_UPDATE_FAILED", "unable to update password")
		return
	}

	// Invalidate all other sessions for security
	_ = h.repo.DeleteAllUserSessions(ctx, userID)

	respondV1Data(w, http.StatusOK, map[string]string{"status": "password changed successfully"})
}

type resetError struct {
	status  int
	code    string
	message string
}

func (h Handler) executePasswordReset(r *http.Request) *resetError {
	var req resetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return &resetError{status: http.StatusBadRequest, code: "INVALID_PAYLOAD", message: "invalid payload"}
	}

	if strings.TrimSpace(req.Token) == "" {
		return &resetError{status: http.StatusBadRequest, code: "TOKEN_REQUIRED", message: "token is required"}
	}
	if !isStrongPassword(req.NewPassword) {
		return &resetError{status: http.StatusBadRequest, code: "WEAK_PASSWORD", message: "password must be at least 12 chars with upper/lower/number/symbol"}
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	tokenHash := auth.HashRefreshToken(strings.TrimSpace(req.Token))
	userID, err := h.repo.GetUserIDByPasswordResetToken(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &resetError{status: http.StatusUnauthorized, code: "INVALID_TOKEN", message: "invalid or expired token"}
		}
		return &resetError{status: http.StatusInternalServerError, code: "TOKEN_LOOKUP_FAILED", message: "unable to validate token"}
	}

	passwordHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		return &resetError{status: http.StatusInternalServerError, code: "PASSWORD_HASH_FAILED", message: "unable to process password"}
	}

	if err := h.repo.UpdateUserPassword(ctx, userID, passwordHash); err != nil {
		return &resetError{status: http.StatusInternalServerError, code: "PASSWORD_UPDATE_FAILED", message: "unable to update password"}
	}

	if err := h.repo.MarkPasswordResetTokenUsed(ctx, tokenHash); err != nil {
		return &resetError{status: http.StatusInternalServerError, code: "TOKEN_UPDATE_FAILED", message: "unable to consume token"}
	}

	return nil
}

func (h Handler) createPasswordResetToken(ctx context.Context, email string) forgotPasswordResponse {
	normalized := strings.TrimSpace(strings.ToLower(email))
	response := forgotPasswordResponse{Status: "if account exists, reset instructions were generated"}
	if normalized == "" {
		return response
	}

	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	user, err := h.repo.GetUserByEmail(dbCtx, normalized)
	if err != nil {
		return response
	}

	token, err := generateResetToken()
	if err != nil {
		return response
	}

	if err := h.repo.CreatePasswordResetToken(dbCtx, user.ID, auth.HashRefreshToken(token), time.Now().UTC().Add(15*time.Minute)); err != nil {
		return response
	}

	// Local-first development mode: return token directly until Mailhog flow is fully wired.
	response.ResetToken = &token
	return response
}

func generateResetToken() (string, error) {
	buffer := make([]byte, 32)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return hex.EncodeToString(buffer), nil
}

func withJSON(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}

func respondJSON(w http.ResponseWriter, status int, data any) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

type apiEnvelope struct {
	Success bool      `json:"success"`
	Data    any       `json:"data"`
	Error   *apiError `json:"error"`
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func respondV1Data(w http.ResponseWriter, status int, payload any) {
	respondJSON(w, status, apiEnvelope{Success: true, Data: payload, Error: nil})
}

func respondV1Error(w http.ResponseWriter, status int, code, message string) {
	respondJSON(w, status, apiEnvelope{Success: false, Data: nil, Error: &apiError{Code: code, Message: message}})
}

func normalizeRole(role string) string {
	r := strings.TrimSpace(strings.ToLower(role))
	switch r {
	case "owner", "engineer", "viewer":
		return r
	default:
		return "engineer"
	}
}

func isStrongPassword(password string) bool {
	if len(password) < 12 {
		return false
	}

	hasUpper := false
	hasLower := false
	hasNumber := false
	hasSymbol := false

	for _, ch := range password {
		switch {
		case ch >= 'A' && ch <= 'Z':
			hasUpper = true
		case ch >= 'a' && ch <= 'z':
			hasLower = true
		case ch >= '0' && ch <= '9':
			hasNumber = true
		default:
			hasSymbol = true
		}
	}

	return hasUpper && hasLower && hasNumber && hasSymbol
}
