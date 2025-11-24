package http

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

// ─── Org Management Endpoints ────────────────────────────────────────────────
// These endpoints require authentication. The gateway injects X-User-ID and X-Org-ID headers.

// resolveOrgID resolves the org ID from the request header, falling back to the user's first org.
func (h Handler) resolveOrgID(ctx context.Context, r *http.Request) string {
	orgID := r.Header.Get("X-Org-ID")
	if orgID != "" {
		return orgID
	}
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		return ""
	}
	orgs, err := h.repo.ListUserOrgs(ctx, userID)
	if err == nil && len(orgs) > 0 {
		return orgs[0].ID
	}
	return ""
}

type inviteMemberRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

type updateRoleRequest struct {
	UserID  string `json:"userId"`
	NewRole string `json:"role"`
}

type removeMemberRequest struct {
	UserID string `json:"userId"`
}

type updateOrgRequest struct {
	Name string `json:"name"`
}

type switchOrgRequest struct {
	OrgID string `json:"orgId"`
}

// GET /api/v1/org — get current user's org info
func (h Handler) getOrg(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondV1Error(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	orgID := h.resolveOrgID(ctx, r)
	if orgID == "" {
		respondV1Error(w, http.StatusBadRequest, "NO_ORG", "no organization context — user has no org memberships")
		return
	}

	org, err := h.repo.GetOrg(ctx, orgID)
	if err != nil {
		respondV1Error(w, http.StatusNotFound, "ORG_NOT_FOUND", "organization not found")
		return
	}

	respondV1Data(w, http.StatusOK, map[string]interface{}{
		"id":                   org.ID,
		"name":                 org.Name,
		"slug":                 org.Slug,
		"plan":                 org.Plan,
		"maxSeats":             org.MaxSeats,
		"maxProjects":          org.MaxProjects,
		"maxSimulationsPerDay": org.MaxSimulationsPerDay,
		"suspendedAt":          org.SuspendedAt,
		"createdAt":            org.CreatedAt,
	})
}

// PUT /api/v1/org — update org settings (name)
func (h Handler) updateOrg(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		respondV1Error(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	userID := r.Header.Get("X-User-ID")
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	orgID := h.resolveOrgID(ctx, r)
	if orgID == "" || userID == "" {
		respondV1Error(w, http.StatusBadRequest, "NO_ORG", "no organization context")
		return
	}

	// Only owner/admin can update org
	role, err := h.repo.GetMemberRole(ctx, orgID, userID)
	if err != nil || (role != "owner" && role != "admin") {
		respondV1Error(w, http.StatusForbidden, "INSUFFICIENT_ROLE", "only org owner or admin can update settings")
		return
	}

	var req updateOrgRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondV1Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "invalid payload")
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		respondV1Error(w, http.StatusBadRequest, "INVALID_NAME", "name is required")
		return
	}

	if err := h.repo.UpdateOrg(ctx, orgID, name); err != nil {
		respondV1Error(w, http.StatusInternalServerError, "UPDATE_FAILED", "failed to update organization")
		return
	}

	respondV1Data(w, http.StatusOK, map[string]string{"status": "updated"})
}

// GET /api/v1/org/members — list org members
func (h Handler) listMembers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondV1Error(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	userID := r.Header.Get("X-User-ID")
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	orgID := h.resolveOrgID(ctx, r)
	if orgID == "" || userID == "" {
		respondV1Error(w, http.StatusBadRequest, "NO_ORG", "no organization context")
		return
	}

	// Verify caller is a member of this org
	callerRole, err := h.repo.GetMemberRole(ctx, orgID, userID)
	if err != nil || callerRole == "" {
		respondV1Error(w, http.StatusForbidden, "NOT_A_MEMBER", "you are not a member of this organization")
		return
	}

	members, err := h.repo.ListOrgMembers(ctx, orgID)
	if err != nil {
		respondV1Error(w, http.StatusInternalServerError, "QUERY_FAILED", "failed to list members")
		return
	}

	type memberResp struct {
		UserID   string `json:"userId"`
		Email    string `json:"email"`
		Role     string `json:"role"`
		JoinedAt string `json:"joinedAt"`
	}

	result := make([]memberResp, 0, len(members))
	for _, m := range members {
		result = append(result, memberResp{
			UserID:   m.UserID,
			Email:    m.Email,
			Role:     m.Role,
			JoinedAt: m.JoinedAt.Format(time.RFC3339),
		})
	}

	respondV1Data(w, http.StatusOK, map[string]interface{}{
		"members":    result,
		"callerRole": callerRole,
	})
}

// POST /api/v1/org/invite — invite a user to the org
func (h Handler) inviteMember(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondV1Error(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	userID := r.Header.Get("X-User-ID")
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	orgID := h.resolveOrgID(ctx, r)
	if orgID == "" || userID == "" {
		respondV1Error(w, http.StatusBadRequest, "NO_ORG", "no organization context")
		return
	}

	// Only owner/admin can invite
	callerRole, err := h.repo.GetMemberRole(ctx, orgID, userID)
	if err != nil || (callerRole != "owner" && callerRole != "admin") {
		respondV1Error(w, http.StatusForbidden, "INSUFFICIENT_ROLE", "only org owner or admin can invite members")
		return
	}

	var req inviteMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondV1Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "invalid payload")
		return
	}

	email := strings.TrimSpace(strings.ToLower(req.Email))
	role := normalizeOrgRole(req.Role)

	if email == "" {
		respondV1Error(w, http.StatusBadRequest, "INVALID_EMAIL", "email is required")
		return
	}

	// Check seat limit
	org, err := h.repo.GetOrg(ctx, orgID)
	if err != nil {
		respondV1Error(w, http.StatusInternalServerError, "ORG_NOT_FOUND", "failed to get org")
		return
	}
	memberCount, err := h.repo.CountOrgMembers(ctx, orgID)
	if err != nil {
		respondV1Error(w, http.StatusInternalServerError, "COUNT_FAILED", "failed to count members")
		return
	}
	if memberCount >= org.MaxSeats {
		respondV1Error(w, http.StatusForbidden, "SEAT_LIMIT", "organization seat limit reached, upgrade your plan")
		return
	}

	// Generate invite token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		respondV1Error(w, http.StatusInternalServerError, "TOKEN_GEN_FAILED", "failed to generate token")
		return
	}
	token := hex.EncodeToString(tokenBytes)

	expiresAt := time.Now().Add(7 * 24 * time.Hour) // 7 days
	invite, err := h.repo.CreateOrgInvite(ctx, orgID, email, role, token, expiresAt)
	if err != nil {
		respondV1Error(w, http.StatusInternalServerError, "INVITE_FAILED", "failed to create invite")
		return
	}

	respondV1Data(w, http.StatusCreated, map[string]interface{}{
		"inviteId":  invite.ID,
		"email":     invite.Email,
		"role":      invite.Role,
		"token":     token,
		"expiresAt": invite.ExpiresAt.Format(time.RFC3339),
	})
}

// GET /api/v1/org/invites — list pending invites
func (h Handler) listInvites(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondV1Error(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	userID := r.Header.Get("X-User-ID")
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	orgID := h.resolveOrgID(ctx, r)
	if orgID == "" || userID == "" {
		respondV1Error(w, http.StatusBadRequest, "NO_ORG", "no organization context")
		return
	}

	// Only owner/admin can see pending invites
	callerRole, err := h.repo.GetMemberRole(ctx, orgID, userID)
	if err != nil || (callerRole != "owner" && callerRole != "admin") {
		respondV1Error(w, http.StatusForbidden, "INSUFFICIENT_ROLE", "only org owner or admin can view invites")
		return
	}

	invites, err := h.repo.ListOrgInvites(ctx, orgID)
	if err != nil {
		respondV1Error(w, http.StatusInternalServerError, "QUERY_FAILED", "failed to list invites")
		return
	}

	type inviteResp struct {
		ID        string `json:"id"`
		Email     string `json:"email"`
		Role      string `json:"role"`
		ExpiresAt string `json:"expiresAt"`
	}

	result := make([]inviteResp, 0, len(invites))
	for _, inv := range invites {
		result = append(result, inviteResp{
			ID:        inv.ID,
			Email:     inv.Email,
			Role:      inv.Role,
			ExpiresAt: inv.ExpiresAt.Format(time.RFC3339),
		})
	}

	respondV1Data(w, http.StatusOK, result)
}

// POST /api/v1/org/invite/accept — accept an invite (called by the invited user)
func (h Handler) acceptInvite(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondV1Error(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		respondV1Error(w, http.StatusUnauthorized, "AUTH_REQUIRED", "authentication required")
		return
	}

	var req struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondV1Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "invalid payload")
		return
	}

	token := strings.TrimSpace(req.Token)
	if token == "" {
		respondV1Error(w, http.StatusBadRequest, "INVALID_TOKEN", "invite token required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	invite, err := h.repo.FindInviteByToken(ctx, token)
	if err != nil {
		respondV1Error(w, http.StatusNotFound, "INVITE_NOT_FOUND", "invite not found or expired")
		return
	}

	if err := h.repo.AcceptInvite(ctx, invite.ID, userID); err != nil {
		respondV1Error(w, http.StatusInternalServerError, "ACCEPT_FAILED", "failed to accept invite")
		return
	}

	respondV1Data(w, http.StatusOK, map[string]interface{}{
		"orgId": invite.OrgID,
		"role":  invite.Role,
	})
}

// PUT /api/v1/org/members/role — change a member's role
func (h Handler) updateMemberRole(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		respondV1Error(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	userID := r.Header.Get("X-User-ID")
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	orgID := h.resolveOrgID(ctx, r)
	if orgID == "" || userID == "" {
		respondV1Error(w, http.StatusBadRequest, "NO_ORG", "no organization context")
		return
	}

	// Only owner can change roles
	callerRole, err := h.repo.GetMemberRole(ctx, orgID, userID)
	if err != nil || callerRole != "owner" {
		respondV1Error(w, http.StatusForbidden, "INSUFFICIENT_ROLE", "only org owner can change member roles")
		return
	}

	var req updateRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondV1Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "invalid payload")
		return
	}

	targetUserID := strings.TrimSpace(req.UserID)
	newRole := normalizeOrgRole(req.NewRole)

	if targetUserID == "" {
		respondV1Error(w, http.StatusBadRequest, "INVALID_USER", "userId is required")
		return
	}

	// Cannot change own role
	if targetUserID == userID {
		respondV1Error(w, http.StatusBadRequest, "CANNOT_CHANGE_OWN_ROLE", "cannot change your own role")
		return
	}

	if err := h.repo.UpdateMemberRole(ctx, orgID, targetUserID, newRole); err != nil {
		respondV1Error(w, http.StatusInternalServerError, "UPDATE_FAILED", "failed to update role")
		return
	}

	respondV1Data(w, http.StatusOK, map[string]string{"userId": targetUserID, "role": newRole})
}

// DELETE /api/v1/org/members — remove a member from the org
func (h Handler) removeMember(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		respondV1Error(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	userID := r.Header.Get("X-User-ID")
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	orgID := h.resolveOrgID(ctx, r)
	if orgID == "" || userID == "" {
		respondV1Error(w, http.StatusBadRequest, "NO_ORG", "no organization context")
		return
	}

	// Only owner/admin can remove members
	callerRole, err := h.repo.GetMemberRole(ctx, orgID, userID)
	if err != nil || (callerRole != "owner" && callerRole != "admin") {
		respondV1Error(w, http.StatusForbidden, "INSUFFICIENT_ROLE", "only org owner or admin can remove members")
		return
	}

	targetUserID := strings.TrimSpace(r.URL.Query().Get("userId"))
	if targetUserID == "" {
		respondV1Error(w, http.StatusBadRequest, "INVALID_USER", "userId query param required")
		return
	}

	// Cannot remove self
	if targetUserID == userID {
		respondV1Error(w, http.StatusBadRequest, "CANNOT_REMOVE_SELF", "cannot remove yourself; transfer ownership first")
		return
	}

	// Cannot remove owner
	targetRole, err := h.repo.GetMemberRole(ctx, orgID, targetUserID)
	if err == nil && targetRole == "owner" {
		respondV1Error(w, http.StatusForbidden, "CANNOT_REMOVE_OWNER", "cannot remove org owner")
		return
	}

	if err := h.repo.RemoveMember(ctx, orgID, targetUserID); err != nil {
		respondV1Error(w, http.StatusInternalServerError, "REMOVE_FAILED", "failed to remove member")
		return
	}

	respondV1Data(w, http.StatusOK, map[string]string{"status": "removed", "userId": targetUserID})
}

// GET /api/v1/org/list — list all orgs the user belongs to
func (h Handler) listUserOrgs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondV1Error(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		respondV1Error(w, http.StatusUnauthorized, "AUTH_REQUIRED", "authentication required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	orgs, err := h.repo.ListUserOrgs(ctx, userID)
	if err != nil {
		respondV1Error(w, http.StatusInternalServerError, "QUERY_FAILED", "failed to list organizations")
		return
	}

	type orgResp struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Slug string `json:"slug"`
		Plan string `json:"plan"`
	}

	result := make([]orgResp, 0, len(orgs))
	for _, o := range orgs {
		result = append(result, orgResp{ID: o.ID, Name: o.Name, Slug: o.Slug, Plan: o.Plan})
	}

	respondV1Data(w, http.StatusOK, result)
}

// POST /api/v1/org/switch — switch active org context (returns new token pair)
func (h Handler) switchOrg(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondV1Error(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		respondV1Error(w, http.StatusUnauthorized, "AUTH_REQUIRED", "authentication required")
		return
	}

	var req switchOrgRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondV1Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "invalid payload")
		return
	}

	targetOrgID := strings.TrimSpace(req.OrgID)
	if targetOrgID == "" {
		respondV1Error(w, http.StatusBadRequest, "INVALID_ORG", "orgId is required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Verify membership
	if err := h.repo.SwitchUserOrg(ctx, userID, targetOrgID); err != nil {
		respondV1Error(w, http.StatusForbidden, "NOT_A_MEMBER", "you are not a member of this organization")
		return
	}

	// Get user info for new token
	user, err := h.repo.GetUserByID(ctx, userID)
	if err != nil {
		respondV1Error(w, http.StatusInternalServerError, "USER_NOT_FOUND", "failed to get user")
		return
	}

	// Get role in target org
	role, err := h.repo.GetMemberRole(ctx, targetOrgID, userID)
	if err != nil {
		role = "viewer"
	}

	// Issue new token pair scoped to the new org
	accessToken, refreshToken, refreshExpiry, err := h.tokens.CreateTokenPair(user.ID, targetOrgID, role, user.IsPlatformAdmin)
	if err != nil {
		respondV1Error(w, http.StatusInternalServerError, "TOKEN_FAILED", "failed to issue new tokens")
		return
	}

	_, _ = h.repo.CreateSession(ctx, user.ID, hashToken(refreshToken), refreshExpiry)

	respondV1Data(w, http.StatusOK, map[string]interface{}{
		"accessToken":  accessToken,
		"refreshToken": refreshToken,
		"orgId":        targetOrgID,
		"role":         role,
	})
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func normalizeOrgRole(role string) string {
	r := strings.TrimSpace(strings.ToLower(role))
	switch r {
	case "owner", "admin", "engineer", "viewer":
		return r
	default:
		return "engineer"
	}
}

// orgRouter dispatches GET/PUT on /api/v1/org
func (h Handler) orgRouter(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.getOrg(w, r)
	case http.MethodPut:
		h.updateOrg(w, r)
	default:
		respondV1Error(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
	}
}

// orgMembersRouter dispatches GET/DELETE on /api/v1/org/members
func (h Handler) orgMembersRouter(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listMembers(w, r)
	case http.MethodDelete:
		h.removeMember(w, r)
	default:
		respondV1Error(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
	}
}
