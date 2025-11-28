package model

import "time"

type Project struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	OrgID       string    `json:"orgId"`
	OwnerID     string    `json:"ownerId"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type ProjectListResponse struct {
	Projects   []Project `json:"projects"`
	NextCursor *string   `json:"nextCursor"`
}

type UpsertProjectRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	OrgID       string `json:"orgId"`
	OwnerID     string `json:"ownerId"`
}

type TeamMember struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

type TeamInviteRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}
