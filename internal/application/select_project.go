package application

import (
	"fmt"
	"path/filepath"

	"github.com/cobyzero/zerocodex/internal/domain"
)

type SelectProject struct {
	Repo  domain.ProjectRepository
	Store domain.ProjectStore
}

func (s *SelectProject) Execute(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	if s.Repo.Validate(absPath) {
		if s.Store != nil {
			if err := s.Store.SaveProject(absPath); err != nil {
				return "", err
			}
		}
		return absPath, nil
	}
	return "", fmt.Errorf("invalid project path")
}

func (s *SelectProject) ListSaved() ([]string, error) {
	if s.Store == nil {
		return []string{}, nil
	}
	return s.Store.ListProjects()
}
