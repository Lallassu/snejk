#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SDK_DIR="$ROOT_DIR/android-sdk"
JDK_DIR="$ROOT_DIR/android-jdk"
APK_PATH="$ROOT_DIR/build/qake-snake-android-game.apk"
PKG_INFO_PATH="$ROOT_DIR/build/qake-snake-android-game.package"
ANDROID_API="${ANDROID_API:-21}"
ANDROID_MIN_SDK="${ANDROID_MIN_SDK:-23}"

"$ROOT_DIR/scripts/android_setup.sh"

export JAVA_HOME="$JDK_DIR"
export ANDROID_SDK_ROOT="$SDK_DIR"
export ANDROID_HOME="$SDK_DIR"

NDK_DIR="$(find "$SDK_DIR/ndk" -mindepth 1 -maxdepth 1 -type d | sort | tail -n1 || true)"
if [[ -z "$NDK_DIR" ]]; then
  echo "Android NDK not found under $SDK_DIR/ndk"
  exit 1
fi
export ANDROID_NDK_HOME="$NDK_DIR"

if ! go list -m golang.org/x/mobile >/dev/null 2>&1; then
  echo "Adding golang.org/x/mobile module"
  go get golang.org/x/mobile@latest
fi

XMOBILE_DIR="$(go list -m -f '{{if .Dir}}{{.Dir}}{{end}}' golang.org/x/mobile 2>/dev/null || true)"
if [[ -z "${XMOBILE_DIR}" || ! -d "${XMOBILE_DIR}" ]]; then
  echo "Downloading golang.org/x/mobile module source"
  go mod download golang.org/x/mobile
  XMOBILE_DIR="$(go list -m -f '{{if .Dir}}{{.Dir}}{{end}}' golang.org/x/mobile 2>/dev/null || true)"
fi
if [[ -z "${XMOBILE_DIR}" || ! -d "${XMOBILE_DIR}" ]]; then
  echo "Could not resolve golang.org/x/mobile module directory"
  exit 1
fi

PATCH_PROFILE="minsdk${ANDROID_MIN_SDK}-landscapefs-v2-u$(id -u)"
PATCHED_XMOBILE_DIR="$ROOT_DIR/.android-tmp/xmobile-patched-${PATCH_PROFILE}"
PATCHED_GOMOBILE_BIN="$ROOT_DIR/.android-tmp/gomobile-${PATCH_PROFILE}"

if [[ ! -x "$PATCHED_GOMOBILE_BIN" ]]; then
  echo "Building patched gomobile (minSdkVersion=${ANDROID_MIN_SDK})"
  rm -rf "$PATCHED_XMOBILE_DIR"
  cp -a "$XMOBILE_DIR" "$PATCHED_XMOBILE_DIR"
  chmod -R u+w "$PATCHED_XMOBILE_DIR"
  sed -i "s/const MinSDK = [0-9][0-9]*/const MinSDK = ${ANDROID_MIN_SDK}/" \
    "$PATCHED_XMOBILE_DIR/internal/binres/sdk.go"

  cat > "$PATCHED_XMOBILE_DIR/app/GoNativeActivity.java" <<'EOF'
package org.golang.app;

import android.app.NativeActivity;
import android.content.pm.ActivityInfo;
import android.content.pm.PackageManager;
import android.os.Bundle;
import android.util.Log;
import android.view.KeyCharacterMap;
import android.view.View;

public class GoNativeActivity extends NativeActivity {
	private static GoNativeActivity goNativeActivity;

	public GoNativeActivity() {
		super();
		goNativeActivity = this;
	}

	String getTmpdir() {
		return getCacheDir().getAbsolutePath();
	}

	static int getRune(int deviceId, int keyCode, int metaState) {
		try {
			int rune = KeyCharacterMap.load(deviceId).get(keyCode, metaState);
			if (rune == 0) {
				return -1;
			}
			return rune;
		} catch (KeyCharacterMap.UnavailableException e) {
			return -1;
		} catch (Exception e) {
			Log.e("Go", "exception reading KeyCharacterMap", e);
			return -1;
		}
	}

	private void load() {
		// Interestingly, NativeActivity uses a different method
		// to find native code to execute, avoiding
		// System.loadLibrary. The result is Java methods
		// implemented in C with JNIEXPORT (and JNI_OnLoad) are not
		// available unless an explicit call to System.loadLibrary
		// is done. So we do it here, borrowing the name of the
		// library from the same AndroidManifest.xml metadata used
		// by NativeActivity.
		try {
			ActivityInfo ai = getPackageManager().getActivityInfo(
					getIntent().getComponent(), PackageManager.GET_META_DATA);
			if (ai.metaData == null) {
				Log.e("Go", "loadLibrary: no manifest metadata found");
				return;
			}
			String libName = ai.metaData.getString("android.app.lib_name");
			System.loadLibrary(libName);
		} catch (Exception e) {
			Log.e("Go", "loadLibrary failed", e);
		}
	}

	private void hideSystemUI() {
		View decor = getWindow().getDecorView();
		decor.setSystemUiVisibility(
				View.SYSTEM_UI_FLAG_LAYOUT_STABLE
						| View.SYSTEM_UI_FLAG_LAYOUT_FULLSCREEN
						| View.SYSTEM_UI_FLAG_FULLSCREEN
						| View.SYSTEM_UI_FLAG_IMMERSIVE_STICKY
		);
	}

	@Override
	public void onCreate(Bundle savedInstanceState) {
		load();
		super.onCreate(savedInstanceState);
		setRequestedOrientation(ActivityInfo.SCREEN_ORIENTATION_SENSOR_LANDSCAPE);
		hideSystemUI();
	}

	@Override
	public void onWindowFocusChanged(boolean hasFocus) {
		super.onWindowFocusChanged(hasFocus);
		if (hasFocus) {
			hideSystemUI();
		}
	}
}
EOF

  (
    cd "$PATCHED_XMOBILE_DIR/cmd/gomobile"
    go build -o "$PATCHED_GOMOBILE_BIN" .
  )
fi
GOMOBILE_BIN="$PATCHED_GOMOBILE_BIN"

echo "Initializing gomobile toolchain"
"$GOMOBILE_BIN" init

mkdir -p "$ROOT_DIR/build"
echo "Building Android game APK with gomobile"
"$GOMOBILE_BIN" build -target=android -androidapi "$ANDROID_API" -o "$APK_PATH" "$ROOT_DIR/cmd/snake"

AAPT_BIN="$SDK_DIR/build-tools/34.0.0/aapt"
if [[ -x "$AAPT_BIN" ]]; then
  PKG_NAME="$("$AAPT_BIN" dump badging "$APK_PATH" | awk -F"'" '/package: name=/{print $2; exit}')"
  if [[ -n "$PKG_NAME" ]]; then
    echo "$PKG_NAME" > "$PKG_INFO_PATH"
    echo "Detected package: $PKG_NAME"
  fi
fi

echo
echo "Game APK built:"
echo "$APK_PATH"
