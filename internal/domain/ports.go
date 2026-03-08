package domain

type ProjectRepository interface {
	Validate(path string) bool
}
