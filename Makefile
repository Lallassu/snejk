SHELL := /usr/bin/env bash

LOADER_APK_PATH := android/app/build/outputs/apk/debug/app-debug.apk
GAME_APK_PATH := build/qake-snake-android-game.apk
GAME_PKG_INFO := build/qake-snake-android-game.package
ADB := android-sdk/platform-tools/adb
LOADER_APP_ID := com.qake.snake

.PHONY: help test run android-setup \
	android-apk-loader android-apk-game android-apk android-apk-path android-clean \
	android-device-check \
	android-install-loader android-uninstall-loader android-launch-loader android-load-usb-loader \
	android-install-game android-uninstall-game android-launch-game android-load-usb-game \
	android-install android-uninstall android-launch android-load-usb

help:
	@echo "Targets:"
	@echo "  test                 Run Go tests"
	@echo "  run                  Run desktop game"
	@echo "  android-setup        Download/install local Android toolchain in repo"
	@echo "  android-apk-loader   Build Java loader APK"
	@echo "  android-apk-game     Build actual Go game APK (gomobile)"
	@echo "  android-apk          Alias for android-apk-game"
	@echo "  android-apk-path     Print actual game APK path"
	@echo "  android-clean        Remove Android build outputs in repo"
	@echo "  android-load-usb     Build+install+launch actual game on USB device"
	@echo "  android-load-usb-game Same as android-load-usb"
	@echo "  android-load-usb-loader Build+install+launch loader app on USB device"

test:
	GOCACHE=/tmp/go-build go test ./...

run:
	go run ./cmd/snake

android-setup:
	./scripts/android_setup.sh

android-apk-loader:
	./scripts/android_build_apk.sh

android-apk-game:
	./scripts/android_build_game_apk.sh

android-apk: android-apk-game

android-apk-path:
	@echo "$(GAME_APK_PATH)"

android-clean:
	rm -rf android/app/build build

android-device-check:
	@if [[ ! -x "$(ADB)" ]]; then \
		echo "adb not found at $(ADB). Run: make android-setup"; \
		exit 1; \
	fi
	@$(ADB) start-server >/dev/null
	@if [[ "$$($(ADB) devices | awk 'NR>1 && $$2=="device" {count++} END {print count+0}')" -lt 1 ]]; then \
		echo "No connected Android device detected."; \
		echo "Connect phone over USB, enable USB debugging, and accept the RSA prompt."; \
		$(ADB) devices; \
		exit 1; \
	fi

android-install-loader: android-apk-loader android-device-check
	$(ADB) install -r "$(LOADER_APK_PATH)"

android-uninstall-loader: android-device-check
	$(ADB) uninstall "$(LOADER_APP_ID)" || true

android-launch-loader: android-device-check
	$(ADB) shell monkey -p "$(LOADER_APP_ID)" -c android.intent.category.LAUNCHER 1

android-load-usb-loader: android-install-loader android-launch-loader

android-install-game: android-apk-game android-device-check
	$(ADB) install -r "$(GAME_APK_PATH)"

android-uninstall-game: android-device-check
	@if [[ -f "$(GAME_PKG_INFO)" ]]; then \
		pkg="$$(cat "$(GAME_PKG_INFO)")"; \
		echo "Uninstalling $$pkg"; \
		$(ADB) uninstall "$$pkg" || true; \
	else \
		echo "Package info not found at $(GAME_PKG_INFO). Build game APK first."; \
	fi

android-launch-game: android-device-check
	@if [[ -f "$(GAME_PKG_INFO)" ]]; then \
		pkg="$$(cat "$(GAME_PKG_INFO)")"; \
		echo "Launching $$pkg"; \
		$(ADB) shell monkey -p "$$pkg" -c android.intent.category.LAUNCHER 1; \
	else \
		echo "Package info not found at $(GAME_PKG_INFO). Build game APK first."; \
		exit 1; \
	fi

android-load-usb-game: android-install-game android-launch-game

android-install: android-install-game
android-uninstall: android-uninstall-game
android-launch: android-launch-game
android-load-usb: android-load-usb-game
