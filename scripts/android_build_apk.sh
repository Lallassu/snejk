#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ANDROID_DIR="$ROOT_DIR/android"
JDK_DIR="$ROOT_DIR/android-jdk"
GRADLE_VERSION="${GRADLE_VERSION:-8.7}"
GRADLE_BIN="$ROOT_DIR/android-gradle/gradle-${GRADLE_VERSION}/bin/gradle"
APK_PATH="$ANDROID_DIR/app/build/outputs/apk/debug/app-debug.apk"

"$ROOT_DIR/scripts/android_setup.sh"

export JAVA_HOME="$JDK_DIR"
export ANDROID_SDK_ROOT="$ROOT_DIR/android-sdk"
export PATH="$JAVA_HOME/bin:$PATH"

if [[ ! -x "$GRADLE_BIN" ]]; then
  echo "Gradle binary not found at $GRADLE_BIN"
  exit 1
fi

echo "Building Android debug APK"
"$GRADLE_BIN" -p "$ANDROID_DIR" --no-daemon assembleDebug

if [[ ! -f "$APK_PATH" ]]; then
  echo "APK was not generated at expected path: $APK_PATH"
  exit 1
fi

echo
echo "APK built:"
echo "$APK_PATH"
