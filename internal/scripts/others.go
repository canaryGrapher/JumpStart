package scripts

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func detectRust(dir string) []Found {
	data, ok := readFile(dir, "Cargo.toml")
	if !ok {
		return nil
	}
	out := []Found{
		{Name: "Build", Command: "cargo build", Source: "Cargo.toml"},
		{Name: "Build Release", Command: "cargo build --release", Source: "Cargo.toml"},
		{Name: "Check", Command: "cargo check", Source: "Cargo.toml"},
		{Name: "Clippy", Command: "cargo clippy", Source: "Cargo.toml"},
	}
	for _, bin := range tomlTableNames(data, "bin") {
		out = append(out, Found{
			Name:    label(bin),
			Command: "cargo run --bin " + bin,
			Source:  "Cargo.toml",
		})
	}
	for name := range tomlSection(data, "alias") {
		out = append(out, Found{
			Name:    label(name),
			Command: "cargo " + name,
			Source:  "Cargo.toml",
		})
	}
	return out
}

// rakeTask matches `task :migrate do` / `desc "..."` style Rakefile tasks.
var rakeTask = regexp.MustCompile(`^\s*task\s+:([A-Za-z0-9_]+)`)

func detectRuby(dir string) []Found {
	var out []Found
	if data, ok := readFile(dir, "Rakefile"); ok {
		for _, m := range rakeTask.FindAllStringSubmatch(string(data), -1) {
			if isLongRunning(m[1]) {
				continue
			}
			out = append(out, Found{
				Name:    label(m[1]),
				Command: "rake " + m[1],
				Source:  "Rakefile",
			})
		}
	}
	if exists(dir, "Gemfile") {
		out = append(out, Found{Name: "Bundle Install", Command: "bundle install", Source: "Gemfile"})
	}
	if exists(dir, "config/application.rb") {
		out = append(out,
			Found{Name: "Db Migrate", Command: "bin/rails db:migrate", Source: "rails"},
			Found{Name: "Db Seed", Command: "bin/rails db:seed", Source: "rails"},
		)
	}
	return out
}

type composerJSON struct {
	Scripts map[string]json.RawMessage `json:"scripts"`
}

func detectPHP(dir string) []Found {
	var out []Found
	if data, ok := readFile(dir, "composer.json"); ok {
		var c composerJSON
		if json.Unmarshal(data, &c) == nil {
			for name := range c.Scripts {
				if isLongRunning(name) || strings.HasPrefix(name, "post-") || strings.HasPrefix(name, "pre-") {
					continue
				}
				out = append(out, Found{
					Name:    label(name),
					Command: "composer run-script " + name,
					Source:  "composer.json",
				})
			}
		}
	}
	if exists(dir, "artisan") {
		out = append(out,
			Found{Name: "Migrate", Command: "php artisan migrate", Source: "artisan"},
			Found{Name: "Seed", Command: "php artisan db:seed", Source: "artisan"},
			Found{Name: "Cache Clear", Command: "php artisan cache:clear", Source: "artisan"},
		)
	}
	return out
}

func detectJVM(dir string) []Found {
	switch {
	case exists(dir, "gradlew"):
		return []Found{
			{Name: "Build", Command: "./gradlew build", Source: "gradle"},
			{Name: "Clean", Command: "./gradlew clean", Source: "gradle"},
			{Name: "Tasks", Command: "./gradlew tasks", Source: "gradle"},
		}
	case exists(dir, "build.gradle"), exists(dir, "build.gradle.kts"):
		return []Found{
			{Name: "Build", Command: "gradle build", Source: "gradle"},
			{Name: "Clean", Command: "gradle clean", Source: "gradle"},
		}
	case exists(dir, "pom.xml"):
		return []Found{
			{Name: "Package", Command: "mvn package", Source: "pom.xml"},
			{Name: "Clean Install", Command: "mvn clean install", Source: "pom.xml"},
			{Name: "Compile", Command: "mvn compile", Source: "pom.xml"},
		}
	}
	return nil
}

func detectDotNet(dir string) []Found {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".csproj") || strings.HasSuffix(e.Name(), ".sln") {
			return []Found{
				{Name: "Build", Command: "dotnet build", Source: "dotnet"},
				{Name: "Publish", Command: "dotnet publish -c Release", Source: "dotnet"},
				{Name: "Restore", Command: "dotnet restore", Source: "dotnet"},
			}
		}
	}
	return nil
}

// scriptDirs are conventional places projects keep runnable shell scripts.
var scriptDirs = []string{"scripts", "bin", "tools"}

func detectShellScripts(dir string) []Found {
	var out []Found
	for _, sub := range scriptDirs {
		entries, err := os.ReadDir(filepath.Join(dir, sub))
		if err != nil {
			continue
		}
		for _, e := range entries {
			name := e.Name()
			if e.IsDir() || !strings.HasSuffix(name, ".sh") {
				continue
			}
			base := strings.TrimSuffix(name, ".sh")
			if isLongRunning(base) {
				continue
			}
			out = append(out, Found{
				Name:    label(base),
				Command: "./" + sub + "/" + name,
				Source:  sub + "/",
			})
		}
	}
	return out
}
