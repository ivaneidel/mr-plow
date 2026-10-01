# 04 · Catalog

Generated from `internal/catalog` by `go run scripts/catalogdoc.go > docs/04-CATALOG.md`. Don't edit by hand.

**Modes**: *rm -rf* removes the path · *empty folder* removes what's inside and keeps the folder · *expand children* offers each child above a size threshold as its own item · *native command* runs the tool's own cleanup · *info only* is shown but never acted on.

## darwin

| location | group | mode | safety | label | note |
|---|---|---|---|---|---|
| `~/.npm/_cacache` | caches | rm -rf | Regenerable | npm cache | Downloaded npm package tarballs. npm fetches them again when needed. |
| `~/.cargo/registry` | caches | rm -rf | Regenerable | Cargo registry | Rust crate sources and index. cargo re-downloads them on the next build. |
| `~/.cargo/git` | caches | rm -rf | Regenerable | Cargo git checkouts | Git dependencies checked out by cargo. Re-fetched on demand. |
| `~/go/pkg/mod` | caches | `go clean -modcache` | Regenerable | Go module cache | Go module downloads. Files are read-only, so Go's own command removes them. |
| `~/.gradle/caches` | caches | rm -rf | Regenerable | Gradle caches | Gradle dependency and build caches. Rebuilt on the next Gradle/Android build. |
| `~/.gradle/wrapper/dists` | caches | rm -rf | Regenerable | Gradle wrapper distributions | Every Gradle version a project ever asked for. Re-downloaded per project. |
| `~/.m2/repository` | caches | rm -rf | Regenerable | Maven repository | Maven/Java dependencies. Re-downloaded on the next build. |
| `~/.pub-cache` | caches | rm -rf | Regenerable | Dart/Flutter pub cache | Dart and Flutter packages. `flutter pub get` downloads them again. |
| `~/.bun/install/cache` | caches | rm -rf | Regenerable | Bun install cache | Packages downloaded by bun. Re-fetched on install. |
| `~/.cache/huggingface` | caches | rm -rf | Review | Hugging Face models | Downloaded ML models and datasets. Big, re-downloadable, but slow to fetch again. |
| `~/.cache` | caches | expand children | children: Regenerable | ~/.cache | Per-tool caches (XDG). Tools rebuild them when needed. |
| `~/.ollama/models` | dev | rm -rf | Review | Ollama models | Local LLM weights. `ollama pull` gets them back; `ollama rm <model>` removes one. |
| `~/.lmstudio/models` | dev | expand children | children: Review | LM Studio models | Local LLM weights downloaded by LM Studio. |
| `~/.rustup/toolchains` | dev | expand children | children: Review | Rust toolchains | Installed Rust toolchains. Old nightlies pile up; `rustup toolchain list` shows them. |
| `~/.nvm/versions/node` | dev | expand children | children: Review | nvm Node versions | Node.js versions installed by nvm. Keep the ones your projects use. |
| `~/.asdf/installs` | dev | expand children | children: Review | asdf runtime versions | Runtime versions installed by asdf. Old ones are dead weight unless a .tool-versions pins them. |
| `~/.android/avd` | vms | expand children | children: Review | Android emulators | Android Virtual Devices. Each one keeps its own disk image. |
| `~/.colima` | vms | rm -rf | Review | Colima VM | Colima's Docker VM, including all its containers and images. |
| `~/.lima` | vms | expand children | children: Review | Lima VMs | Lima Linux VMs. `limactl delete <name>` removes one cleanly. |
| `~/.minikube` | vms | rm -rf | Review | minikube | minikube clusters and cached images. `minikube delete --all` is the clean way. |
| `~/Downloads` | trash | expand children | children: Review | Downloads | Things you downloaded once. Installers and archives are usually safe to drop. |
| `~/Documents` | media | expand children | children: Personal | Documents | Your documents. Only you know what matters here. |
| `~/Desktop` | media | expand children | children: Personal | Desktop | Things on your Desktop. |
| `~/Music` | media | expand children | children: Personal | Music | Music libraries. Personal: maybe your mom's. |
| `~/Pictures` | media | expand children | children: Personal | Pictures | Photos and images. A Photos library must be managed from the Photos app. |
| `~/Library/Caches` | caches | expand children | children: Regenerable | App caches (~/Library/Caches) | Per-app caches. Apps rebuild them; quit an app before removing its cache. |
| `~/Library/Caches/Homebrew` | caches | `brew cleanup -s --prune=all` | Regenerable | Homebrew downloads | Downloaded bottles and old versions. `brew cleanup` also drops outdated installed versions. |
| `~/Library/pnpm/store` | caches | `pnpm store prune` | Regenerable | pnpm store | pnpm's shared package store. Projects hard-link into it, so prune instead of rm. |
| `~/Library/Logs` | caches | empty folder | Regenerable | App logs | Log files written by apps. Only useful when debugging a crash. |
| `~/Library/Developer/Xcode/DerivedData` | dev | rm -rf | Safe | Xcode DerivedData | Xcode build products and indexes. Rebuilt on the next build. |
| `~/Library/Developer/Xcode/Archives` | dev | expand children | children: Review | Xcode archives | App archives you built for distribution. Keep the ones you may need to symbolicate. |
| `~/Library/Developer/Xcode/iOS DeviceSupport` | dev | expand children | children: Regenerable | iOS DeviceSupport | Debug symbols per iOS version you plugged in. Recreated when a device connects. |
| `~/Library/Developer/CoreSimulator/Devices` | vms | `xcrun simctl delete unavailable` | Review | iOS Simulators | Simulator devices and their data. The command only removes simulators for runtimes you no longer have. |
| `~/Library/Developer/CoreSimulator/Caches` | dev | rm -rf | Safe | Simulator caches | Simulator dyld caches. Rebuilt on the next simulator boot. |
| `~/Library/Android/sdk/system-images` | vms | expand children | children: Review | Android system images | Emulator OS images. Re-downloadable from the SDK Manager. |
| `~/Library/Containers/com.docker.docker/Data/vms/0/data` | vms | `docker system prune -a -f` | Review | Docker Desktop disk | Docker's VM disk (Docker.raw). Prune removes stopped containers, unused images and build cache; volumes are kept. Docker must be running. |
| `~/Library/Group Containers/HUGAJBCSOP.dev.orbstack` | vms | `docker system prune -a -f` | Review | OrbStack data | OrbStack's containers, images and Linux machines. Prune unused images; delete machines from OrbStack. |
| `~/Library/Containers/com.utmapp.UTM/Data/Documents` | vms | expand children | children: Personal | UTM virtual machines | UTM virtual machines. Each is a whole OS disk. |
| `~/Parallels` | vms | expand children | children: Personal | Parallels VMs | Parallels virtual machines. |
| `~/VirtualBox VMs` | vms | expand children | children: Personal | VirtualBox VMs | VirtualBox virtual machines. |
| `~/Library/Application Support/MobileSync/Backup` | apps | expand children | children: Personal | iPhone/iPad backups | Local device backups. Old phones' backups are often forgotten here. |
| `~/Library/Application Support/Code/Cache*` | apps | rm -rf | Regenerable | VS Code caches | VS Code's Chromium caches and downloaded extension packages. Quit VS Code first. |
| `~/Library/Application Support/Slack/Cache*` | apps | rm -rf | Regenerable | Slack caches | Slack's web cache. Quit Slack first. |
| `~/Library/Application Support/Google/Chrome/*/Service Worker/CacheStorage` | apps | rm -rf | Regenerable | Chrome service-worker caches | Offline caches websites stored in Chrome. Sites refill them. Quit Chrome first. |
| `~/Library/Application Support` | apps | expand children | children: Review | App data (Application Support) | Data apps keep for themselves. Leftovers of uninstalled apps live here. |
| `~/Library/Containers` | apps | expand children | children: Review | Sandboxed app data (Containers) | Data of sandboxed (App Store) apps. Leftovers of uninstalled apps live here. |
| `~/Library/Messages/Attachments` | apps | info only | Personal | Messages attachments | Photos and files from iMessage chats. Remove them from Messages (Settings › Storage), not here. |
| `~/Library/Mail` | apps | info only | Personal | Mail | Your mail store. Manage it from the Mail app. |
| `~/.Trash` | trash | empty folder | Safe | Trash | Things you already deleted. Emptying the Trash frees the space. |
| `~/Movies` | media | expand children | children: Personal | Movies | Videos and editing projects (iMovie/Final Cut libraries live here). |
| `/Library/Caches` | system | expand children | children: Regenerable | System-wide caches | Caches shared by all users. Removing them needs sudo. |
| `(Time Machine local snapshots)` | system | `tmutil thinlocalsnapshots / 999999999999 4` | Regenerable | Time Machine local snapshots | APFS snapshots Time Machine keeps locally; they can hold deleted files' space. macOS thins them itself when space runs low. |

## linux

| location | group | mode | safety | label | note |
|---|---|---|---|---|---|
| `~/.npm/_cacache` | caches | rm -rf | Regenerable | npm cache | Downloaded npm package tarballs. npm fetches them again when needed. |
| `~/.cargo/registry` | caches | rm -rf | Regenerable | Cargo registry | Rust crate sources and index. cargo re-downloads them on the next build. |
| `~/.cargo/git` | caches | rm -rf | Regenerable | Cargo git checkouts | Git dependencies checked out by cargo. Re-fetched on demand. |
| `~/go/pkg/mod` | caches | `go clean -modcache` | Regenerable | Go module cache | Go module downloads. Files are read-only, so Go's own command removes them. |
| `~/.gradle/caches` | caches | rm -rf | Regenerable | Gradle caches | Gradle dependency and build caches. Rebuilt on the next Gradle/Android build. |
| `~/.gradle/wrapper/dists` | caches | rm -rf | Regenerable | Gradle wrapper distributions | Every Gradle version a project ever asked for. Re-downloaded per project. |
| `~/.m2/repository` | caches | rm -rf | Regenerable | Maven repository | Maven/Java dependencies. Re-downloaded on the next build. |
| `~/.pub-cache` | caches | rm -rf | Regenerable | Dart/Flutter pub cache | Dart and Flutter packages. `flutter pub get` downloads them again. |
| `~/.bun/install/cache` | caches | rm -rf | Regenerable | Bun install cache | Packages downloaded by bun. Re-fetched on install. |
| `~/.cache/huggingface` | caches | rm -rf | Review | Hugging Face models | Downloaded ML models and datasets. Big, re-downloadable, but slow to fetch again. |
| `~/.cache` | caches | expand children | children: Regenerable | ~/.cache | Per-tool caches (XDG). Tools rebuild them when needed. |
| `~/.ollama/models` | dev | rm -rf | Review | Ollama models | Local LLM weights. `ollama pull` gets them back; `ollama rm <model>` removes one. |
| `~/.lmstudio/models` | dev | expand children | children: Review | LM Studio models | Local LLM weights downloaded by LM Studio. |
| `~/.rustup/toolchains` | dev | expand children | children: Review | Rust toolchains | Installed Rust toolchains. Old nightlies pile up; `rustup toolchain list` shows them. |
| `~/.nvm/versions/node` | dev | expand children | children: Review | nvm Node versions | Node.js versions installed by nvm. Keep the ones your projects use. |
| `~/.asdf/installs` | dev | expand children | children: Review | asdf runtime versions | Runtime versions installed by asdf. Old ones are dead weight unless a .tool-versions pins them. |
| `~/.android/avd` | vms | expand children | children: Review | Android emulators | Android Virtual Devices. Each one keeps its own disk image. |
| `~/.colima` | vms | rm -rf | Review | Colima VM | Colima's Docker VM, including all its containers and images. |
| `~/.lima` | vms | expand children | children: Review | Lima VMs | Lima Linux VMs. `limactl delete <name>` removes one cleanly. |
| `~/.minikube` | vms | rm -rf | Review | minikube | minikube clusters and cached images. `minikube delete --all` is the clean way. |
| `~/Downloads` | trash | expand children | children: Review | Downloads | Things you downloaded once. Installers and archives are usually safe to drop. |
| `~/Documents` | media | expand children | children: Personal | Documents | Your documents. Only you know what matters here. |
| `~/Desktop` | media | expand children | children: Personal | Desktop | Things on your Desktop. |
| `~/Music` | media | expand children | children: Personal | Music | Music libraries. Personal: maybe your mom's. |
| `~/Pictures` | media | expand children | children: Personal | Pictures | Photos and images. A Photos library must be managed from the Photos app. |
| `~/.local/share/pnpm/store` | caches | `pnpm store prune` | Regenerable | pnpm store | pnpm's shared package store. Projects hard-link into it, so prune instead of rm. |
| `~/.local/share/Trash` | trash | empty folder | Safe | Trash | Things you already deleted. Emptying the Trash frees the space. |
| `~/.local/share/docker` | vms | `docker system prune -a -f` | Review | Docker (rootless) | Docker images, containers and build cache. Prune keeps volumes. |
| `~/.var/app` | apps | expand children | children: Review | Flatpak app data | Per-app data of Flatpak apps. |
| `~/snap` | apps | expand children | children: Review | Snap app data | Per-app data of Snap apps. |
| `~/Videos` | media | expand children | children: Personal | Videos | Your videos. |
| `/var/lib/docker` | system | `docker system prune -a -f` | Review | Docker | Docker images, containers and build cache. Prune keeps volumes. |
| `/var/cache/apt/archives` | system | `apt-get clean` | Safe | apt package cache | Downloaded .deb packages. `apt-get clean` removes them. |
| `/var/log/journal` | system | `journalctl --vacuum-time=2weeks` | Regenerable | systemd journal | System logs. Vacuuming keeps the last two weeks. |
| `/var/lib/snapd/cache` | system | empty folder | Safe | snapd cache | Cached snap downloads. |

## Project-artifact detectors

Checked for every directory during the walk. Nested matches (a `node_modules` inside a `node_modules`) are not tagged, and neither are matches inside tool-managed areas (any hidden folder under home, `~/Library`, `~/Applications`).

| id | directory names | requires a sibling | safety |
|---|---|---|---|
| node_modules | `node_modules` | — | Regenerable |
| rust-target | `target` | `Cargo.toml` | Regenerable |
| python-venv | `.venv`, `venv`, `.tox`, `.nox` | `pyproject.toml`, `requirements.txt`, `setup.py`, `Pipfile`, `poetry.lock`, `uv.lock`, `tox.ini` | Regenerable |
| pods | `Pods` | `Podfile` | Regenerable |
| gradle-build | `build`, `.gradle`, `.cxx` | `build.gradle`, `build.gradle.kts`, `settings.gradle`, `settings.gradle.kts` | Regenerable |
| flutter-build | `build`, `.dart_tool` | `pubspec.yaml` | Regenerable |
| js-build-cache | `.next`, `.nuxt`, `.svelte-kit`, `.turbo`, `.parcel-cache`, `.angular`, `.expo`, `.vite` | — | Regenerable |
| swift-build | `.build` | `Package.swift` | Regenerable |
| zig-cache | `.zig-cache`, `zig-cache`, `zig-out` | — | Regenerable |
| terraform | `.terraform` | — | Regenerable |
| pycache | `__pycache__`, `.pytest_cache`, `.mypy_cache`, `.ruff_cache` | — | Safe |

## Old big files

Files from the top-500 heap (≥ 50 MB) not modified for a year, outside tool-managed areas, that are disk images, archives/installers or videos, or are over 1 GB.
