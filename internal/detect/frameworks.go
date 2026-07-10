package detect

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

func exists(dir, name string) bool {
	_, err := os.Stat(filepath.Join(dir, name))
	return err == nil
}

// detectFramework returns language, framework and a suggested start
// command for dir, or nil if no known project marker is found.
func detectFramework(dir string) *Detected {
	switch {
	case exists(dir, "package.json"):
		return detectNode(dir)
	case exists(dir, "go.mod"):
		return &Detected{Language: "Go", Command: "go run ."}
	case exists(dir, "Cargo.toml"):
		return &Detected{Language: "Rust", Command: "cargo run"}
	case exists(dir, "pyproject.toml"), exists(dir, "requirements.txt"), exists(dir, "manage.py"):
		return detectPython(dir)
	case exists(dir, "Gemfile"):
		return detectRuby(dir)
	case exists(dir, "composer.json"):
		return detectPHP(dir)
	case exists(dir, "pom.xml"):
		return &Detected{Language: "Java", Framework: "Maven", Command: "mvn spring-boot:run"}
	case exists(dir, "build.gradle"), exists(dir, "build.gradle.kts"):
		return &Detected{Language: "Java", Framework: "Gradle", Command: "./gradlew bootRun"}
	case exists(dir, "docker-compose.yml"), exists(dir, "docker-compose.yaml"), exists(dir, "compose.yaml"):
		return &Detected{Language: "Docker", Framework: "Docker Compose", Command: "docker compose up"}
	}
	return nil
}

// --- Node ---

type packageJSON struct {
	Scripts         map[string]string `json:"scripts"`
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
}

var nodeFrameworks = []struct{ dep, name string }{
	{"next", "Next.js"},
	{"nuxt", "Nuxt"},
	{"@remix-run/react", "Remix"},
	{"@sveltejs/kit", "SvelteKit"},
	{"astro", "Astro"},
	{"@angular/core", "Angular"},
	{"@nestjs/core", "NestJS"},
	{"react-scripts", "Create React App"},
	{"vite", "Vite"},
	{"express", "Express"},
	{"fastify", "Fastify"},
	{"koa", "Koa"},
	{"react", "React"},
	{"vue", "Vue"},
	{"electron", "Electron"},
}

func detectNode(dir string) *Detected {
	d := &Detected{Language: "JavaScript/TypeScript"}
	pm := nodePackageManager(dir)

	data, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		d.Command = pm.run("dev")
		return d
	}
	var pkg packageJSON
	if json.Unmarshal(data, &pkg) != nil {
		d.Command = pm.run("dev")
		return d
	}

	deps := map[string]bool{}
	for k := range pkg.Dependencies {
		deps[k] = true
	}
	for k := range pkg.DevDependencies {
		deps[k] = true
	}
	for _, f := range nodeFrameworks {
		if deps[f.dep] {
			d.Framework = f.name
			break
		}
	}

	for _, script := range []string{"dev", "start", "serve"} {
		if _, ok := pkg.Scripts[script]; ok {
			d.Command = pm.run(script)
			return d
		}
	}
	d.Command = pm.run("dev")
	return d
}

type pkgManager string

func (p pkgManager) run(script string) string {
	if p == "npm" && script == "start" {
		return "npm start"
	}
	if p == "npm" {
		return "npm run " + script
	}
	return string(p) + " " + script
}

func nodePackageManager(dir string) pkgManager {
	switch {
	case exists(dir, "pnpm-lock.yaml"):
		return "pnpm"
	case exists(dir, "yarn.lock"):
		return "yarn"
	case exists(dir, "bun.lockb"), exists(dir, "bun.lock"):
		return "bun run"
	default:
		return "npm"
	}
}

// --- Python ---

func detectPython(dir string) *Detected {
	d := &Detected{Language: "Python"}
	if exists(dir, "manage.py") {
		d.Framework = "Django"
		d.Command = "python manage.py runserver"
		return d
	}
	deps := readPythonDeps(dir)
	switch {
	case strings.Contains(deps, "fastapi"):
		d.Framework = "FastAPI"
		d.Command = "uvicorn main:app --reload"
	case strings.Contains(deps, "flask"):
		d.Framework = "Flask"
		d.Command = "flask run"
	default:
		d.Command = "python main.py"
	}
	return d
}

func readPythonDeps(dir string) string {
	var out strings.Builder
	for _, f := range []string{"requirements.txt", "pyproject.toml"} {
		if data, err := os.ReadFile(filepath.Join(dir, f)); err == nil {
			out.WriteString(strings.ToLower(string(data)))
		}
	}
	return out.String()
}

// --- Ruby / PHP ---

func detectRuby(dir string) *Detected {
	d := &Detected{Language: "Ruby"}
	if data, err := os.ReadFile(filepath.Join(dir, "Gemfile")); err == nil &&
		strings.Contains(strings.ToLower(string(data)), "rails") {
		d.Framework = "Rails"
		d.Command = "bin/rails server"
		return d
	}
	d.Command = "bundle exec ruby app.rb"
	return d
}

func detectPHP(dir string) *Detected {
	d := &Detected{Language: "PHP"}
	if exists(dir, "artisan") {
		d.Framework = "Laravel"
		d.Command = "php artisan serve"
		return d
	}
	d.Command = "php -S localhost:8000"
	return d
}
