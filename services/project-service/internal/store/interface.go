package store

import "moktopus/project-service/internal/model"

// Store is the project persistence interface
type Store interface {
	List(orgID, cursor string, limit int) model.ProjectListResponse
	Get(id, orgID string) (model.Project, bool)
	Upsert(id string, req model.UpsertProjectRequest) model.Project
	Delete(id, orgID string)
	CountByOrg(orgID string) int
	GetOrgMaxProjects(orgID string) int
	ListTeam(projectID string) []model.TeamMember
	AddTeamMember(projectID string, member model.TeamMember) model.TeamMember
}
