package handlers

import "path/filepath"

func templatePath(name string) string {
	return filepath.Join("templates", name)
}
