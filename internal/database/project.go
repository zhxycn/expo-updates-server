package database

import (
	"context"
	"time"

	"github.com/google/uuid"

	"expo-updates-server/internal/model"
)

func (d *Database) CreateProject(ctx context.Context, name, slug, userId string) (*model.Project, error) {
	now := time.Now().UTC().Format(time.RFC3339)

	project := &model.Project{
		ID:        uuid.Must(uuid.NewV7()).String(),
		Name:      name,
		Slug:      slug,
		CreatedAt: now,
	}

	_, err := d.client.NewInsert().Model(project).Exec(ctx)
	if err != nil {
		return nil, err
	}

	user := &model.ProjectUser{
		ID:        uuid.Must(uuid.NewV7()).String(),
		ProjectID: project.ID,
		UserID:    userId,
		Role:      model.RoleOwner,
		CreatedAt: now,
	}

	_, err = d.client.NewInsert().Model(user).Exec(ctx)
	if err != nil {
		return nil, err
	}

	return project, nil
}

func (d *Database) ListProjectsByUser(ctx context.Context, userId string) ([]model.Project, error) {
	projects := make([]model.Project, 0)

	err := d.client.NewSelect().Model(&projects).
		Join("JOIN project_user ON project_user.project_id = project.id").
		Where("project_user.user_id = ?", userId).
		OrderExpr("project.created_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	return projects, nil
}

func (d *Database) GetProjectUser(ctx context.Context, projectId, userId string) (*model.ProjectUser, error) {
	user := new(model.ProjectUser)

	err := d.client.NewSelect().Model(user).Where("project_id = ? AND user_id = ?", projectId, userId).Scan(ctx)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (d *Database) AddProjectUser(ctx context.Context, projectId, userId string, role model.Role) (*model.ProjectUser, error) {
	user := &model.ProjectUser{
		ID:        uuid.Must(uuid.NewV7()).String(),
		ProjectID: projectId,
		UserID:    userId,
		Role:      role,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	_, err := d.client.NewInsert().Model(user).Exec(ctx)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (d *Database) DeleteProjectUser(ctx context.Context, projectId, userId string) error {
	_, err := d.client.NewDelete().Model((*model.ProjectUser)(nil)).
		Where("project_id = ? AND user_id = ?", projectId, userId).
		Exec(ctx)
	return err
}
