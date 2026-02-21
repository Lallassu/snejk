# Qake The Snake

Qake The Snake is a fast top-down action game where a growing worm tears through a generated city, hunts pedestrians, and triggers extreme bonus effects.
The game supports both desktop and Android.

Website: https://www.qake.se

<img src="website/screenshots/2026-02-21_12-40-07.png" alt="Qake The Snake preview" width="360" />

## Project Structure

- `cmd/snake/`: platform entrypoints (desktop + Android).
- `internal/game/`: core game logic and systems.
- `internal/game/main.go`: desktop game loop orchestration.
- `internal/game/main_android.go`: Android app loop and touch input path.
- `internal/game/snake.go`: player logic, movement, combat, and bonus abilities.
- `internal/game/bonus.go`: bonus definitions, spawning, and activation behavior.
- `internal/game/world.go`, `internal/game/worldgen.go`, `internal/game/chunk.go`: world and generation.
- `internal/game/pedestrians.go`, `internal/game/traffic.go`, `internal/game/cops.go`, `internal/game/military.go`: NPC systems.
- `internal/game/destruction.go`, `internal/game/particle*.go`: destruction and particles.
- `internal/game/renderer.go`, `internal/game/render_*.go`, `internal/game/shaders.go`: rendering paths.
- `internal/game/ui.go`, `internal/game/gamestate.go`, `internal/game/levels.go`: HUD and progression.
- `website/`: marketing/download site, screenshots, and static web assets.
- `.github/workflows/release-tag.yml`: tag-triggered release build + website link PR automation.

## Build Locally

Requirements:

- Go (version from `go.mod`)

Run:

```bash
go test ./...
go run ./cmd/snake
```

## Android Build

Requirements:

- USB debugging enabled on your Android phone (for install/launch via USB)

Quick start:

```bash
make android-setup
make android-apk-game
```

Game APK output:

```bash
build/qake-snake-android-game.apk
```

Install and launch on a connected phone:

```bash
make android-load-usb
```

Useful Android targets:

- `make android-apk-game`: build the Android game APK (gomobile path).
- `make android-load-usb`: build + install + launch on USB device.
- `make android-install-game`: install already-built APK.
- `make android-launch-game`: launch installed package.
- `make android-device-check`: verify adb + connected device.
- `make android-apk-loader`: build Java bootstrap loader APK (dev/fallback path).

## Release Automation

Tag pushes on the default branch trigger GitHub Actions to:

1. Build static binaries for Linux, Windows, and macOS (amd64 + arm64 package).
2. Build the Android APK.
3. Publish assets to the GitHub release for that tag and mark it as `latest`.

Website download buttons use stable latest-release URLs, so no website link updates are needed:

- `https://github.com/Lallassu/snejk/releases/latest/download/qake-the-snake-windows-amd64.zip`
- `https://github.com/Lallassu/snejk/releases/latest/download/qake-the-snake-linux-amd64.tar.gz`
- `https://github.com/Lallassu/snejk/releases/latest/download/qake-the-snake-macos-universal.tar.gz`
- `https://github.com/Lallassu/snejk/releases/latest/download/qake-the-snake-android.apk`

## Generated Project

This repository is a generated project.

For non-generated reference work and long-developed source projects with pixel destruction, see:

- https://github.com/Lallassu/gizmo
- https://github.com/Lallassu/badsanta
- https://github.com/Lallassu/bintris
- https://github.com/Lallassu/moonshot
- ...and many more at https://github.com/Lallassu/

## Screenshots

![Qake screenshot 1](website/screenshots/2026-02-21_12-40-07.png)
![Qake screenshot 2](website/screenshots/2026-02-21_12-40-34.png)
![Qake screenshot 3](website/screenshots/2026-02-21_12-40-43.png)
![Qake screenshot 4](website/screenshots/2026-02-21_12-40-54.png)
![Qake screenshot 5](website/screenshots/2026-02-21_12-41-07.png)
![Qake screenshot 6](website/screenshots/2026-02-21_12-41-39.png)
![Qake screenshot 7](website/screenshots/2026-02-21_12-41-59.png)
![Qake screenshot 8](website/screenshots/2026-02-21_12-42-18.png)
![Qake screenshot 9](website/screenshots/2026-02-21_12-42-35.png)
![Qake screenshot 10](website/screenshots/2026-02-21_12-42-46.png)
![Qake screenshot 11](website/screenshots/Screenshot_20260221-124359.jpg)
![Qake screenshot 12](website/screenshots/Screenshot_20260221-124402.jpg)
![Qake screenshot 13](website/screenshots/Screenshot_20260221-124429.jpg)
![Qake screenshot 14](website/screenshots/Screenshot_20260221-124440.jpg)
![Qake screenshot 15](website/screenshots/Screenshot_20260221-124508.jpg)
