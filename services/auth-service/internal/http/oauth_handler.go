package http

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"moktopus/auth-service/internal/auth"
)

// ─── Logout ─────────────────────────────────────────────────────────────────

type logoutRequest struct {
	RefreshToken string `json:"refreshToken"`
}

func (h Handler) logoutV1(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondV1Error(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	userID := r.Header.Get("X-User-ID")

	var req logoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RefreshToken == "" {
		// Even without a refresh token body, invalidate all user sessions
		if userID != "" {
			ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
			defer cancel()
			_ = h.repo.DeleteAllUserSessions(ctx, userID)
		}
		respondV1Data(w, http.StatusOK, map[string]string{"status": "logged_out"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	refreshHash := auth.HashRefreshToken(req.RefreshToken)
	if userID != "" {
		_ = h.repo.DeleteSessionByRefreshHash(ctx, userID, refreshHash)
	}

	respondV1Data(w, http.StatusOK, map[string]string{"status": "logged_out"})
}

// ─── Get current user ────────────────────────────────────────────────────────

func (h Handler) me(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondV1Error(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		respondV1Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	user, err := h.repo.GetUserByID(ctx, userID)
	if err != nil {
		respondV1Error(w, http.StatusNotFound, "USER_NOT_FOUND", "user not found")
		return
	}

	respondV1Data(w, http.StatusOK, map[string]interface{}{
		"id":      user.ID,
		"email":   user.Email,
		"role":    user.Role,
		"orgId":   user.OrgID,
		"isAdmin": user.IsPlatformAdmin,
	})
}

// ─── OAuth: Google ──────────────────────────────────────────────────────────

func (h Handler) oauthGoogleRedirect(w http.ResponseWriter, r *http.Request) {
	if h.cfg.GoogleClientID == "" {
		respondV1Error(w, http.StatusNotImplemented, "OAUTH_NOT_CONFIGURED", "Google OAuth not configured")
		return
	}

	state := generateOAuthState()
	redirectURI := h.cfg.OAuthRedirectBase + "/api/v1/auth/oauth/google/callback"

	authURL := fmt.Sprintf(
		"https://accounts.google.com/o/oauth2/v2/auth?client_id=%s&redirect_uri=%s&response_type=code&scope=%s&state=%s&access_type=offline&prompt=consent",
		url.QueryEscape(h.cfg.GoogleClientID),
		url.QueryEscape(redirectURI),
		url.QueryEscape("openid email profile"),
		url.QueryEscape(state),
	)

	respondV1Data(w, http.StatusOK, map[string]string{"url": authURL, "state": state})
}

func (h Handler) oauthGoogleCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Redirect(w, r, h.cfg.FrontendURL+"/login?error=oauth_failed", http.StatusTemporaryRedirect)
		return
	}

	redirectURI := h.cfg.OAuthRedirectBase + "/api/v1/auth/oauth/google/callback"

	// Exchange code for tokens
	tokenResp, err := exchangeGoogleCode(code, h.cfg.GoogleClientID, h.cfg.GoogleClientSecret, redirectURI)
	if err != nil {
		http.Redirect(w, r, h.cfg.FrontendURL+"/login?error=oauth_exchange_failed", http.StatusTemporaryRedirect)
		return
	}

	// Get user info from Google
	userInfo, err := getGoogleUserInfo(tokenResp.AccessToken)
	if err != nil {
		http.Redirect(w, r, h.cfg.FrontendURL+"/login?error=oauth_userinfo_failed", http.StatusTemporaryRedirect)
		return
	}

	// Find or create user
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	user, _, err := h.repo.GetOrCreateOAuthUser(ctx, userInfo.Email, userInfo.Name, "google")
	if err != nil {
		http.Redirect(w, r, h.cfg.FrontendURL+"/login?error=oauth_user_create_failed", http.StatusTemporaryRedirect)
		return
	}

	// Create token pair
	accessToken, refreshToken, refreshExpiry, err := h.tokens.CreateTokenPair(user.ID, user.OrgID, user.Role, user.IsPlatformAdmin)
	if err != nil {
		http.Redirect(w, r, h.cfg.FrontendURL+"/login?error=token_failed", http.StatusTemporaryRedirect)
		return
	}

	_, _ = h.repo.CreateSession(ctx, user.ID, auth.HashRefreshToken(refreshToken), refreshExpiry)

	// Redirect to frontend with tokens
	redirectURL := fmt.Sprintf(
		"%s/oauth/callback?accessToken=%s&refreshToken=%s&userId=%s&orgId=%s&role=%s&email=%s&isAdmin=%t",
		h.cfg.FrontendURL,
		url.QueryEscape(accessToken),
		url.QueryEscape(refreshToken),
		url.QueryEscape(user.ID),
		url.QueryEscape(user.OrgID),
		url.QueryEscape(user.Role),
		url.QueryEscape(user.Email),
		user.IsPlatformAdmin,
	)
	http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
}

// ─── OAuth: GitHub ──────────────────────────────────────────────────────────

func (h Handler) oauthGithubRedirect(w http.ResponseWriter, r *http.Request) {
	if h.cfg.GitHubClientID == "" {
		respondV1Error(w, http.StatusNotImplemented, "OAUTH_NOT_CONFIGURED", "GitHub OAuth not configured")
		return
	}

	state := generateOAuthState()
	redirectURI := h.cfg.OAuthRedirectBase + "/api/v1/auth/oauth/github/callback"

	authURL := fmt.Sprintf(
		"https://github.com/login/oauth/authorize?client_id=%s&redirect_uri=%s&scope=%s&state=%s",
		url.QueryEscape(h.cfg.GitHubClientID),
		url.QueryEscape(redirectURI),
		url.QueryEscape("read:user user:email"),
		url.QueryEscape(state),
	)

	respondV1Data(w, http.StatusOK, map[string]string{"url": authURL, "state": state})
}

func (h Handler) oauthGithubCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Redirect(w, r, h.cfg.FrontendURL+"/login?error=oauth_failed", http.StatusTemporaryRedirect)
		return
	}

	redirectURI := h.cfg.OAuthRedirectBase + "/api/v1/auth/oauth/github/callback"

	// Exchange code for token
	accessToken, err := exchangeGitHubCode(code, h.cfg.GitHubClientID, h.cfg.GitHubClientSecret, redirectURI)
	if err != nil {
		http.Redirect(w, r, h.cfg.FrontendURL+"/login?error=oauth_exchange_failed", http.StatusTemporaryRedirect)
		return
	}

	// Get user info from GitHub
	userInfo, err := getGitHubUserInfo(accessToken)
	if err != nil {
		http.Redirect(w, r, h.cfg.FrontendURL+"/login?error=oauth_userinfo_failed", http.StatusTemporaryRedirect)
		return
	}

	// Find or create user
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	user, _, err := h.repo.GetOrCreateOAuthUser(ctx, userInfo.Email, userInfo.Name, "github")
	if err != nil {
		http.Redirect(w, r, h.cfg.FrontendURL+"/login?error=oauth_user_create_failed", http.StatusTemporaryRedirect)
		return
	}

	// Create token pair
	jwtAccessToken, refreshToken, refreshExpiry, err := h.tokens.CreateTokenPair(user.ID, user.OrgID, user.Role, user.IsPlatformAdmin)
	if err != nil {
		http.Redirect(w, r, h.cfg.FrontendURL+"/login?error=token_failed", http.StatusTemporaryRedirect)
		return
	}

	_, _ = h.repo.CreateSession(ctx, user.ID, auth.HashRefreshToken(refreshToken), refreshExpiry)

	// Redirect to frontend with tokens
	redirectURL := fmt.Sprintf(
		"%s/oauth/callback?accessToken=%s&refreshToken=%s&userId=%s&orgId=%s&role=%s&email=%s&isAdmin=%t",
		h.cfg.FrontendURL,
		url.QueryEscape(jwtAccessToken),
		url.QueryEscape(refreshToken),
		url.QueryEscape(user.ID),
		url.QueryEscape(user.OrgID),
		url.QueryEscape(user.Role),
		url.QueryEscape(user.Email),
		user.IsPlatformAdmin,
	)
	http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
}

// ─── OAuth Helpers ──────────────────────────────────────────────────────────

func generateOAuthState() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

type googleTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
}

type googleUserInfo struct {
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
}

func exchangeGoogleCode(code, clientID, clientSecret, redirectURI string) (*googleTokenResponse, error) {
	data := url.Values{
		"code":          {code},
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"redirect_uri":  {redirectURI},
		"grant_type":    {"authorization_code"},
	}

	resp, err := http.Post("https://oauth2.googleapis.com/token", "application/x-www-form-urlencoded", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("google token exchange failed: %s", string(body))
	}

	var tokenResp googleTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, err
	}
	return &tokenResp, nil
}

func getGoogleUserInfo(accessToken string) (*googleUserInfo, error) {
	req, _ := http.NewRequest("GET", "https://www.googleapis.com/oauth2/v3/userinfo", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("google userinfo failed: status %d", resp.StatusCode)
	}

	var info googleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}
	return &info, nil
}

type gitHubTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
}

type gitHubUserInfo struct {
	Login string `json:"login"`
	Name  string `json:"name"`
	Email string `json:"email"`
	ID    int64  `json:"id"`
}

type gitHubEmail struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}

func exchangeGitHubCode(code, clientID, clientSecret, redirectURI string) (string, error) {
	data := url.Values{
		"code":          {code},
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"redirect_uri":  {redirectURI},
	}

	req, _ := http.NewRequest("POST", "https://github.com/login/oauth/access_token", strings.NewReader(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var tokenResp gitHubTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", err
	}
	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("github token exchange returned empty token")
	}
	return tokenResp.AccessToken, nil
}

func getGitHubUserInfo(accessToken string) (*gitHubUserInfo, error) {
	req, _ := http.NewRequest("GET", "https://api.github.com/user", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var info gitHubUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}

	// If email is empty, fetch from /user/emails
	if info.Email == "" {
		info.Email = fetchGitHubPrimaryEmail(accessToken)
	}

	if info.Email == "" {
		return nil, fmt.Errorf("unable to get email from GitHub")
	}

	return &info, nil
}

func fetchGitHubPrimaryEmail(accessToken string) string {
	req, _ := http.NewRequest("GET", "https://api.github.com/user/emails", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	var emails []gitHubEmail
	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return ""
	}

	for _, e := range emails {
		if e.Primary && e.Verified {
			return e.Email
		}
	}
	for _, e := range emails {
		if e.Verified {
			return e.Email
		}
	}
	return ""
}
