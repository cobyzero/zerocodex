package domain

type ProjectStore interface {
	SaveProject(path string) error
	ListProjects() ([]string, error)
}
