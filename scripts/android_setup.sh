#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TMP_DIR="$ROOT_DIR/.android-tmp"
JDK_DIR="$ROOT_DIR/android-jdk"
GRADLE_DIR="$ROOT_DIR/android-gradle"
SDK_DIR="$ROOT_DIR/android-sdk"
ANDROID_DIR="$ROOT_DIR/android"

JDK_URL="${JDK_URL:-https://api.adoptium.net/v3/binary/latest/17/ga/linux/x64/jdk/hotspot/normal/eclipse}"
GRADLE_VERSION="${GRADLE_VERSION:-8.7}"
GRADLE_URL="${GRADLE_URL:-https://services.gradle.org/distributions/gradle-${GRADLE_VERSION}-bin.zip}"
CMDLINE_TOOLS_URL="${CMDLINE_TOOLS_URL:-https://dl.google.com/android/repository/commandlinetools-linux-11076708_latest.zip}"
NDK_PACKAGE="${NDK_PACKAGE:-ndk;26.3.11579264}"

mkdir -p "$TMP_DIR"

install_jdk() {
  if [[ -x "$JDK_DIR/bin/java" ]]; then
    echo "JDK already installed at $JDK_DIR"
    return
  fi
  echo "Installing JDK into $JDK_DIR"
  rm -rf "$JDK_DIR"
  local archive="$TMP_DIR/jdk.tar.gz"
  curl -fL "$JDK_URL" -o "$archive"
  local unpack="$TMP_DIR/jdk-unpack"
  rm -rf "$unpack"
  mkdir -p "$unpack"
  tar -xzf "$archive" -C "$unpack"
  local extracted
  extracted="$(find "$unpack" -mindepth 1 -maxdepth 1 -type d | head -n1)"
  if [[ -z "$extracted" ]]; then
    echo "Failed to unpack JDK archive"
    exit 1
  fi
  mv "$extracted" "$JDK_DIR"
}

install_gradle() {
  local target="$GRADLE_DIR/gradle-${GRADLE_VERSION}"
  if [[ -x "$target/bin/gradle" ]]; then
    echo "Gradle already installed at $target"
    return
  fi
  echo "Installing Gradle ${GRADLE_VERSION} into $GRADLE_DIR"
  rm -rf "$GRADLE_DIR"
  mkdir -p "$GRADLE_DIR"
  local archive="$TMP_DIR/gradle.zip"
  curl -fL "$GRADLE_URL" -o "$archive"
  unzip -q "$archive" -d "$GRADLE_DIR"
}

install_cmdline_tools() {
  local target="$SDK_DIR/cmdline-tools/latest/bin/sdkmanager"
  if [[ -x "$target" ]]; then
    echo "Android command-line tools already installed"
    return
  fi
  echo "Installing Android command-line tools into $SDK_DIR"
  rm -rf "$SDK_DIR/cmdline-tools"
  mkdir -p "$SDK_DIR/cmdline-tools"
  local archive="$TMP_DIR/cmdline-tools.zip"
  curl -fL "$CMDLINE_TOOLS_URL" -o "$archive"
  local unpack="$TMP_DIR/cmdline-tools-unpack"
  rm -rf "$unpack"
  mkdir -p "$unpack"
  unzip -q "$archive" -d "$unpack"
  if [[ -d "$unpack/cmdline-tools" ]]; then
    mv "$unpack/cmdline-tools" "$SDK_DIR/cmdline-tools/latest"
  else
    local extracted
    extracted="$(find "$unpack" -mindepth 1 -maxdepth 1 -type d | head -n1)"
    if [[ -z "$extracted" ]]; then
      echo "Failed to unpack Android command-line tools"
      exit 1
    fi
    mv "$extracted" "$SDK_DIR/cmdline-tools/latest"
  fi
}

install_jdk
install_gradle
install_cmdline_tools

export JAVA_HOME="$JDK_DIR"
export ANDROID_SDK_ROOT="$SDK_DIR"
export PATH="$JAVA_HOME/bin:$SDK_DIR/cmdline-tools/latest/bin:$SDK_DIR/platform-tools:$PATH"

echo "Accepting SDK licenses"
yes | sdkmanager --sdk_root="$SDK_DIR" --licenses >/dev/null || true

echo "Installing required Android SDK packages"
sdkmanager --sdk_root="$SDK_DIR" \
  "platform-tools" \
  "platforms;android-34" \
  "build-tools;34.0.0" \
  "$NDK_PACKAGE"

mkdir -p "$ANDROID_DIR"
cat >"$ANDROID_DIR/local.properties" <<EOF
sdk.dir=$SDK_DIR
EOF

echo
echo "Android setup complete."
echo "JAVA_HOME=$JAVA_HOME"
echo "ANDROID_SDK_ROOT=$ANDROID_SDK_ROOT"
