package filesystem

import (
	"os"
)

type ProjectFS struct{}

func (p *ProjectFS) Validate(path string) bool {

	info, err := os.Stat(path)

	if err != nil {
		return false
	}

	return info.IsDir()
}
