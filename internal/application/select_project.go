package application

import "github.com/cobyzero/zerocodex/internal/domain"

type SelectProjectUseCase struct {
	Repo domain.ProjectRepository
}

func (u *SelectProjectUseCase) Execute(path string) (*domain.Project, error) {

	if !u.Repo.Validate(path) {
		return nil, nil
	}

	project := &domain.Project{
		Path: path,
	}

	return project, nil
}
