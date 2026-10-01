// Package catalog knows where disk space usually hides: well-known cache,
// toolchain, VM and personal folders, plus detectors for per-project build
// artifacts (node_modules, target/, .venv…).
package catalog

import (
	"path/filepath"
	"strings"

	"github.com/ivaneidel/mr-plow/internal/i18n"
)

// Safety says how much thought a deletion needs.
type Safety int

const (
	Safe        Safety = iota // junk: nothing is lost
	Regenerable               // rebuilt/re-downloaded on demand
	Review                    // probably junk, but look first
	Personal                  // your stuff; only you can tell
)

// Group is a Suspects-screen section.
type Group string

const (
	GroupDev    Group = "dev"
	GroupCaches Group = "caches"
	GroupVMs    Group = "vms"
	GroupApps   Group = "apps"
	GroupBig    Group = "big"
	GroupTrash  Group = "trash"
	GroupMedia  Group = "media"
	GroupSystem Group = "system"
)

// Groups in display order.
var Groups = []Group{GroupDev, GroupCaches, GroupVMs, GroupApps, GroupBig, GroupTrash, GroupMedia, GroupSystem}

// Mode says what removing an entry means.
type Mode int

const (
	ModeRemove   Mode = iota // rm -rf the path
	ModeContents             // rm -rf everything inside the path, keep the folder
	ModeExpand               // not an item itself: each big child becomes an item
	ModeNative               // run the tool's own cleanup command instead of rm
	ModeInfo                 // shown for awareness only; no action offered
)

// Entry is one known location.
type Entry struct {
	ID      string
	Path    string // may start with ~ and contain * globs
	Group   Group
	Safety  Safety
	Mode    Mode
	Label   string
	Explain i18n.Text
	Native  []string // argv for ModeNative
	// ExpandDepth is how many levels down ModeExpand items are taken (default 1).
	ExpandDepth int
	// ExpandMin hides expanded children smaller than this (default 50 MB).
	ExpandMin int64
	// ChildSafety/ChildMode apply to expanded children (default Remove).
	ChildSafety Safety
	ChildMode   Mode
	System      bool // outside $HOME; deleting usually needs root
}

const mb = 1 << 20

// Resolve expands ~ and globs into absolute paths that exist.
func (e Entry) Resolve(home string) []string {
	p := e.Path
	if strings.HasPrefix(p, "~/") {
		p = filepath.Join(home, p[2:])
	}
	if !strings.ContainsAny(p, "*?[") {
		return []string{p}
	}
	m, _ := filepath.Glob(p)
	return m
}

// For returns the catalog for an OS ("darwin", "linux", …).
func For(goos string) []Entry {
	out := append([]Entry(nil), common...)
	switch goos {
	case "darwin":
		out = append(out, darwin...)
	case "linux":
		out = append(out, linux...)
	}
	return out
}

func t(en, es string) i18n.Text { return i18n.Text{EN: en, ES: es} }

// common entries exist on every Unix-like system.
var common = []Entry{
	{ID: "npm", Path: "~/.npm/_cacache", Group: GroupCaches, Safety: Regenerable, Label: "npm cache",
		Explain: t("Downloaded npm package tarballs. npm fetches them again when needed.",
			"Paquetes de npm descargados. npm los vuelve a bajar cuando hagan falta.")},
	{ID: "cargo-registry", Path: "~/.cargo/registry", Group: GroupCaches, Safety: Regenerable, Label: "Cargo registry",
		Explain: t("Rust crate sources and index. cargo re-downloads them on the next build.",
			"Fuentes e índice de crates de Rust. cargo los vuelve a bajar en el próximo build.")},
	{ID: "cargo-git", Path: "~/.cargo/git", Group: GroupCaches, Safety: Regenerable, Label: "Cargo git checkouts",
		Explain: t("Git dependencies checked out by cargo. Re-fetched on demand.",
			"Dependencias git de cargo. Se vuelven a bajar cuando hagan falta.")},
	{ID: "go-modcache", Path: "~/go/pkg/mod", Group: GroupCaches, Safety: Regenerable, Mode: ModeNative,
		Native: []string{"go", "clean", "-modcache"}, Label: "Go module cache",
		Explain: t("Go module downloads. Files are read-only, so Go's own command removes them.",
			"Módulos de Go descargados. Son de solo lectura, así que se borran con el comando de Go.")},
	{ID: "gradle-caches", Path: "~/.gradle/caches", Group: GroupCaches, Safety: Regenerable, Label: "Gradle caches",
		Explain: t("Gradle dependency and build caches. Rebuilt on the next Gradle/Android build.",
			"Caché de dependencias y builds de Gradle. Se regenera en el próximo build.")},
	{ID: "gradle-wrapper", Path: "~/.gradle/wrapper/dists", Group: GroupCaches, Safety: Regenerable, Label: "Gradle wrapper distributions",
		Explain: t("Every Gradle version a project ever asked for. Re-downloaded per project.",
			"Cada versión de Gradle que algún proyecto pidió. Se vuelven a bajar por proyecto.")},
	{ID: "m2", Path: "~/.m2/repository", Group: GroupCaches, Safety: Regenerable, Label: "Maven repository",
		Explain: t("Maven/Java dependencies. Re-downloaded on the next build.",
			"Dependencias de Maven/Java. Se vuelven a bajar en el próximo build.")},
	{ID: "pub-cache", Path: "~/.pub-cache", Group: GroupCaches, Safety: Regenerable, Label: "Dart/Flutter pub cache",
		Explain: t("Dart and Flutter packages. `flutter pub get` downloads them again.",
			"Paquetes de Dart y Flutter. `flutter pub get` los vuelve a bajar.")},
	{ID: "bun-cache", Path: "~/.bun/install/cache", Group: GroupCaches, Safety: Regenerable, Label: "Bun install cache",
		Explain: t("Packages downloaded by bun. Re-fetched on install.",
			"Paquetes descargados por bun. Se vuelven a bajar al instalar.")},
	{ID: "huggingface", Path: "~/.cache/huggingface", Group: GroupCaches, Safety: Review, Label: "Hugging Face models",
		Explain: t("Downloaded ML models and datasets. Big, re-downloadable, but slow to fetch again.",
			"Modelos y datasets de ML descargados. Grandes y re-descargables, pero lentos de bajar.")},
	{ID: "dot-cache", Path: "~/.cache", Group: GroupCaches, Mode: ModeExpand, ChildSafety: Regenerable, Label: "~/.cache",
		Explain: t("Per-tool caches (XDG). Tools rebuild them when needed.",
			"Cachés de herramientas (XDG). Se regeneran cuando hacen falta.")},
	{ID: "ollama", Path: "~/.ollama/models", Group: GroupDev, Safety: Review, Label: "Ollama models",
		Explain: t("Local LLM weights. `ollama pull` gets them back; `ollama rm <model>` removes one.",
			"Pesos de LLMs locales. `ollama pull` los recupera; `ollama rm <modelo>` borra uno.")},
	{ID: "lmstudio", Path: "~/.lmstudio/models", Group: GroupDev, Mode: ModeExpand, ExpandDepth: 2, ChildSafety: Review, Label: "LM Studio models",
		Explain: t("Local LLM weights downloaded by LM Studio.", "Pesos de LLMs locales bajados por LM Studio.")},
	{ID: "rustup", Path: "~/.rustup/toolchains", Group: GroupDev, Mode: ModeExpand, ChildSafety: Review, Label: "Rust toolchains",
		Explain: t("Installed Rust toolchains. Old nightlies pile up; `rustup toolchain list` shows them.",
			"Toolchains de Rust instalados. Los nightlies viejos se acumulan.")},
	{ID: "nvm", Path: "~/.nvm/versions/node", Group: GroupDev, Mode: ModeExpand, ChildSafety: Review, ExpandMin: 20 * mb, Label: "nvm Node versions",
		Explain: t("Node.js versions installed by nvm. Keep the ones your projects use.",
			"Versiones de Node.js instaladas con nvm. Quedate con las que usan tus proyectos.")},
	{ID: "asdf", Path: "~/.asdf/installs", Group: GroupDev, Mode: ModeExpand, ExpandDepth: 2, ChildSafety: Review, ExpandMin: 20 * mb, Label: "asdf runtime versions",
		Explain: t("Runtime versions installed by asdf. Old ones are dead weight unless a .tool-versions pins them.",
			"Versiones instaladas con asdf. Las viejas sobran salvo que un .tool-versions las fije.")},
	{ID: "android-avd", Path: "~/.android/avd", Group: GroupVMs, Mode: ModeExpand, ChildSafety: Review, Label: "Android emulators",
		Explain: t("Android Virtual Devices. Each one keeps its own disk image.",
			"Emuladores de Android. Cada uno guarda su propia imagen de disco.")},
	{ID: "colima", Path: "~/.colima", Group: GroupVMs, Safety: Review, Label: "Colima VM",
		Explain: t("Colima's Docker VM, including all its containers and images.",
			"La VM de Docker de Colima, con todos sus contenedores e imágenes.")},
	{ID: "lima", Path: "~/.lima", Group: GroupVMs, Mode: ModeExpand, ChildSafety: Review, Label: "Lima VMs",
		Explain: t("Lima Linux VMs. `limactl delete <name>` removes one cleanly.",
			"VMs Linux de Lima. `limactl delete <nombre>` borra una limpiamente.")},
	{ID: "minikube", Path: "~/.minikube", Group: GroupVMs, Safety: Review, Label: "minikube",
		Explain: t("minikube clusters and cached images. `minikube delete --all` is the clean way.",
			"Clusters e imágenes de minikube. `minikube delete --all` es la forma limpia.")},
	{ID: "downloads", Path: "~/Downloads", Group: GroupTrash, Mode: ModeExpand, ChildSafety: Review, Label: "Downloads",
		Explain: t("Things you downloaded once. Installers and archives are usually safe to drop.",
			"Cosas que bajaste alguna vez. Instaladores y comprimidos suelen poder borrarse.")},
	{ID: "documents", Path: "~/Documents", Group: GroupMedia, Mode: ModeExpand, ChildSafety: Personal, ExpandMin: 200 * mb, Label: "Documents",
		Explain: t("Your documents. Only you know what matters here.", "Tus documentos. Solo vos sabés qué importa acá.")},
	{ID: "desktop", Path: "~/Desktop", Group: GroupMedia, Mode: ModeExpand, ChildSafety: Personal, ExpandMin: 100 * mb, Label: "Desktop",
		Explain: t("Things on your Desktop.", "Cosas en tu Escritorio.")},
	{ID: "music", Path: "~/Music", Group: GroupMedia, Mode: ModeExpand, ExpandDepth: 2, ChildSafety: Personal, ExpandMin: 100 * mb, Label: "Music",
		Explain: t("Music libraries. Personal: maybe your mom's.", "Bibliotecas de música. Personal: quizás de tu mamá.")},
	{ID: "pictures", Path: "~/Pictures", Group: GroupMedia, Mode: ModeExpand, ChildSafety: Personal, ExpandMin: 100 * mb, Label: "Pictures",
		Explain: t("Photos and images. A Photos library must be managed from the Photos app.",
			"Fotos e imágenes. Una fototeca de Fotos se gestiona desde la app Fotos.")},
}

// Detector tags a project build artifact found anywhere during the walk.
type Detector struct {
	ID      string
	Names   []string // directory names that trigger the check
	Label   string
	Explain i18n.Text
	Safety  Safety
	// Siblings: at least one must exist next to the directory ("" = no requirement).
	Siblings []string
}

// Detectors are checked for every directory during the walk.
var Detectors = []Detector{
	{ID: "node_modules", Names: []string{"node_modules"}, Label: "node_modules", Safety: Regenerable,
		Explain: t("JavaScript dependencies of a project. `npm install` (or pnpm/yarn/bun) brings them back.",
			"Dependencias JavaScript de un proyecto. `npm install` (o pnpm/yarn/bun) las recupera.")},
	{ID: "rust-target", Names: []string{"target"}, Siblings: []string{"Cargo.toml"}, Label: "Rust target/", Safety: Regenerable,
		Explain: t("Rust build output. `cargo build` recreates it.", "Salida de compilación de Rust. `cargo build` la recrea.")},
	{ID: "python-venv", Names: []string{".venv", "venv", ".tox", ".nox"}, Siblings: []string{"pyproject.toml", "requirements.txt", "setup.py", "Pipfile", "poetry.lock", "uv.lock", "tox.ini"}, Label: "Python virtualenv", Safety: Regenerable,
		Explain: t("A project's Python virtual environment. Recreate with `uv sync` / `pip install -r requirements.txt`.",
			"Entorno virtual de Python de un proyecto. Se recrea con `uv sync` / `pip install -r requirements.txt`.")},
	{ID: "pods", Names: []string{"Pods"}, Siblings: []string{"Podfile"}, Label: "CocoaPods Pods/", Safety: Regenerable,
		Explain: t("iOS dependencies installed by CocoaPods. `pod install` brings them back.",
			"Dependencias iOS instaladas por CocoaPods. `pod install` las recupera.")},
	{ID: "gradle-build", Names: []string{"build", ".gradle", ".cxx"}, Siblings: []string{"build.gradle", "build.gradle.kts", "settings.gradle", "settings.gradle.kts"}, Label: "Gradle build output", Safety: Regenerable,
		Explain: t("Gradle/Android build output. The next build recreates it.", "Salida de build de Gradle/Android. El próximo build la recrea.")},
	{ID: "flutter-build", Names: []string{"build", ".dart_tool"}, Siblings: []string{"pubspec.yaml"}, Label: "Flutter/Dart build output", Safety: Regenerable,
		Explain: t("Flutter build output and tool state. `flutter pub get` + build recreates it.",
			"Salida de build de Flutter. `flutter pub get` + build la recrea.")},
	{ID: "js-build-cache", Names: []string{".next", ".nuxt", ".svelte-kit", ".turbo", ".parcel-cache", ".angular", ".expo", ".vite"}, Label: "JS framework build cache", Safety: Regenerable,
		Explain: t("Framework build cache (Next, Nuxt, SvelteKit, Turbo, Expo…). Recreated on the next dev/build.",
			"Caché de build de frameworks (Next, Nuxt, SvelteKit, Turbo, Expo…). Se recrea en el próximo dev/build.")},
	{ID: "swift-build", Names: []string{".build"}, Siblings: []string{"Package.swift"}, Label: "SwiftPM .build", Safety: Regenerable,
		Explain: t("Swift Package Manager build output. `swift build` recreates it.", "Salida de build de SwiftPM. `swift build` la recrea.")},
	{ID: "zig-cache", Names: []string{".zig-cache", "zig-cache", "zig-out"}, Label: "Zig build cache", Safety: Regenerable,
		Explain: t("Zig build cache/output. Recreated on the next build.", "Caché/salida de build de Zig. Se recrea en el próximo build.")},
	{ID: "terraform", Names: []string{".terraform"}, Label: "Terraform providers", Safety: Regenerable,
		Explain: t("Downloaded Terraform providers and modules. `terraform init` restores them.",
			"Providers y módulos de Terraform descargados. `terraform init` los restaura.")},
	{ID: "pycache", Names: []string{"__pycache__", ".pytest_cache", ".mypy_cache", ".ruff_cache"}, Label: "Python caches", Safety: Safe,
		Explain: t("Bytecode and tool caches. Python recreates them automatically.",
			"Bytecode y cachés de herramientas. Python los recrea solo.")},
}

// DetectorByID returns the detector with that ID, or nil.
func DetectorByID(id string) *Detector {
	for i := range Detectors {
		if Detectors[i].ID == id {
			return &Detectors[i]
		}
	}
	return nil
}
