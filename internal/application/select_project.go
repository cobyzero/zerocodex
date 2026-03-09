package application

import (
	"fmt"

	"github.com/cobyzero/zerocodex/internal/domain"
)

type SelectProject struct {
	Repo domain.ProjectRepository
}

func (s *SelectProject) Execute(path string) (string, error) {
	if s.Repo.Validate(path) {
		return path, nil
	}
	return "", fmt.Errorf("invalid project path")
}
