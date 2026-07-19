package scripts

import (
	"os"
	"path/filepath"
)

func detectPython(dir string) []Found {
	var out []Found

	if data, ok := readFile(dir, "pyproject.toml"); ok {
		for name := range tomlSection(data, "tool.poetry.scripts") {
			if isLongRunning(name) {
				continue
			}
			out = append(out, Found{
				Name:    label(name),
				Command: "poetry run " + name,
				Source:  "pyproject.toml",
			})
		}
		for name := range tomlSection(data, "project.scripts") {
			if isLongRunning(name) {
				continue
			}
			out = append(out, Found{
				Name:    label(name),
				Command: name,
				Source:  "pyproject.toml",
			})
		}
	}

	// Django and Alembic expose their tasks through a CLI rather than a
	// manifest, so detect them from the well-known entrypoint files.
	if exists(dir, "manage.py") {
		out = append(out,
			Found{Name: "Migrate", Command: "python manage.py migrate", Source: "manage.py"},
			Found{Name: "Make Migrations", Command: "python manage.py makemigrations", Source: "manage.py"},
			Found{Name: "Collect Static", Command: "python manage.py collectstatic --noinput", Source: "manage.py"},
		)
	}
	if exists(dir, "alembic.ini") {
		out = append(out,
			Found{Name: "Alembic Upgrade", Command: "alembic upgrade head", Source: "alembic.ini"},
			Found{Name: "Alembic Downgrade", Command: "alembic downgrade -1", Source: "alembic.ini"},
		)
	}
	if hasRequirements(dir) {
		out = append(out, Found{
			Name:    "Install Requirements",
			Command: "pip install -r requirements.txt",
			Source:  "requirements.txt",
		})
	}
	return out
}

func hasRequirements(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "requirements.txt"))
	return err == nil
}
