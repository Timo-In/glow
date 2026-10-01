#!/bin/sh
# Git-Plus Linter Installer & Dependency Resolver
# 支持 macOS / Linux / Windows (Git Bash / MSYS / Cygwin)

set -e

REPO_ROOT="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
HOOKS_DIR="$REPO_ROOT/.githooks"
BIN_DIR="$HOOKS_DIR/bin"
LOCAL_BIN="$HOME/.local/bin"
VERSION="v2.0.1"

# 1. 查找是否已存在可用 linter
find_linter() {
  if command -v pommitlint >/dev/null 2>&1; then echo "pommitlint"; return 0; fi
  if [ -x "/usr/local/bin/pommitlint" ]; then echo "/usr/local/bin/pommitlint"; return 0; fi
  if [ -x "$LOCAL_BIN/pommitlint" ]; then echo "$LOCAL_BIN/pommitlint"; return 0; fi
  if [ -x "$BIN_DIR/pommitlint" ]; then echo "$BIN_DIR/pommitlint"; return 0; fi
  if command -v pommitlint.exe >/dev/null 2>&1; then echo "pommitlint.exe"; return 0; fi
  if [ -x "$LOCAL_BIN/pommitlint.exe" ]; then echo "$LOCAL_BIN/pommitlint.exe"; return 0; fi
  if [ -x "$BIN_DIR/pommitlint.exe" ]; then echo "$BIN_DIR/pommitlint.exe"; return 0; fi
  if command -v commitlint >/dev/null 2>&1; then echo "commitlint"; return 0; fi
  if [ -x "/usr/local/bin/commitlint" ]; then echo "/usr/local/bin/commitlint"; return 0; fi
  if command -v commitlint.cmd >/dev/null 2>&1; then echo "commitlint.cmd"; return 0; fi
  return 1
}

if find_linter >/dev/null 2>&1; then
  exit 0
fi

echo "[Git-Plus] 首次提交，正在准备 Commit 规范校验工具 (pommitlint)..."

# 2. 识别操作系统与 CPU 架构
OS="$(uname -s 2>/dev/null || echo "Unknown")"
ARCH_RAW="$(uname -m 2>/dev/null || echo "amd64")"

case "$ARCH_RAW" in
  x86_64|amd64) ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *) ARCH="amd64" ;;
esac

add_to_gitignore() {
  ENTRY="$1"
  GIT_IGNORE="$REPO_ROOT/.gitignore"
  if [ -f "$GIT_IGNORE" ]; then
    if ! grep -qF "$ENTRY" "$GIT_IGNORE" 2>/dev/null; then
      echo "$ENTRY" >> "$GIT_IGNORE"
    fi
  else
    echo "$ENTRY" > "$GIT_IGNORE"
  fi
}

case "$OS" in
  Darwin*)
    # macOS 系统
    mkdir -p "$LOCAL_BIN"
    if command -v go >/dev/null 2>&1; then
      echo "[Git-Plus] macOS 环境检测到 Go，执行 go install github.com/shuymn/pommitlint@latest ..."
      go install github.com/shuymn/pommitlint@latest
      GOBIN_DIR="$(go env GOBIN 2>/dev/null || true)"
      GOPATH_DIR="$(go env GOPATH 2>/dev/null || echo "$HOME/go")"
      SRC_BIN=""
      if [ -n "$GOBIN_DIR" ] && [ -f "$GOBIN_DIR/pommitlint" ]; then
        SRC_BIN="$GOBIN_DIR/pommitlint"
      elif [ -f "$GOPATH_DIR/bin/pommitlint" ]; then
        SRC_BIN="$GOPATH_DIR/bin/pommitlint"
      fi
      if [ -n "$SRC_BIN" ]; then
        cp -f "$SRC_BIN" "$LOCAL_BIN/pommitlint"
        chmod +x "$LOCAL_BIN/pommitlint"
        echo "[Git-Plus] 已复制至 $LOCAL_BIN/pommitlint"
      fi
    else
      echo "[Git-Plus] macOS 未检测到 Go 环境，下载 GitHub Release 预编译包..."
      TMP_DIR="$(mktemp -d 2>/dev/null || mktemp -d -t 'pommitlint')"
      DOWNLOAD_URL="https://github.com/shuymn/pommitlint/releases/download/${VERSION}/pommitlint_${VERSION}_darwin_${ARCH}.tar.gz"
      if curl -fsSL "$DOWNLOAD_URL" -o "$TMP_DIR/pommitlint.tar.gz"; then
        tar -zxf "$TMP_DIR/pommitlint.tar.gz" -C "$TMP_DIR"
        if [ -f "$TMP_DIR/pommitlint" ]; then
          mv -f "$TMP_DIR/pommitlint" "$LOCAL_BIN/pommitlint"
          chmod +x "$LOCAL_BIN/pommitlint"
          echo "[Git-Plus] 成功下载并安装至 $LOCAL_BIN/pommitlint"
        fi
      fi
      rm -rf "$TMP_DIR"
    fi
    ;;

  MINGW*|MSYS*|CYGWIN*|Windows_NT*)
    # Windows 环境 (Git Bash 等)
    mkdir -p "$BIN_DIR"
    add_to_gitignore ".githooks/bin/"
    add_to_gitignore ".githooks/*.exe"
    add_to_gitignore ".githooks/pommitlint*"

    echo "[Git-Plus] Windows 环境，下载 pommitlint.exe 至 .githooks/bin/ ..."
    TMP_DIR="$(mktemp -d 2>/dev/null || mktemp -d -t 'pommitlint')"
    DOWNLOAD_URL="https://github.com/shuymn/pommitlint/releases/download/${VERSION}/pommitlint_${VERSION}_windows_${ARCH}.zip"
    if curl -fsSL "$DOWNLOAD_URL" -o "$TMP_DIR/pommitlint.zip"; then
      if command -v unzip >/dev/null 2>&1; then
        unzip -q -o "$TMP_DIR/pommitlint.zip" -d "$TMP_DIR"
      elif command -v 7z >/dev/null 2>&1; then
        7z x -y "$TMP_DIR/pommitlint.zip" -o"$TMP_DIR" >/dev/null
      fi
      if [ -f "$TMP_DIR/pommitlint.exe" ]; then
        mv -f "$TMP_DIR/pommitlint.exe" "$BIN_DIR/pommitlint.exe"
        chmod +x "$BIN_DIR/pommitlint.exe" 2>/dev/null || true
        echo "[Git-Plus] 成功安装至 $BIN_DIR/pommitlint.exe (已自动加入 .gitignore)"
      fi
    fi
    rm -rf "$TMP_DIR"
    ;;

  *)
    # Linux 或其他 Unix 系统
    # Linux 环境通常没有 ~/.local/bin，优先判定系统级 /usr/local/bin 是否可写，否则使用项目内的 .githooks/bin/
    mkdir -p "$BIN_DIR"
    add_to_gitignore ".githooks/bin/"
    add_to_gitignore ".githooks/pommitlint*"

    TARGET_DIR="$BIN_DIR"
    if [ -w "/usr/local/bin" ]; then
      TARGET_DIR="/usr/local/bin"
    fi

    if command -v go >/dev/null 2>&1; then
      echo "[Git-Plus] Linux 环境检测到 Go，执行编译安装..."
      go install github.com/shuymn/pommitlint@latest
      GOBIN_DIR="$(go env GOBIN 2>/dev/null || true)"
      GOPATH_DIR="$(go env GOPATH 2>/dev/null || echo "$HOME/go")"
      SRC_BIN=""
      if [ -n "$GOBIN_DIR" ] && [ -f "$GOBIN_DIR/pommitlint" ]; then
        SRC_BIN="$GOBIN_DIR/pommitlint"
      elif [ -f "$GOPATH_DIR/bin/pommitlint" ]; then
        SRC_BIN="$GOPATH_DIR/bin/pommitlint"
      fi
      if [ -n "$SRC_BIN" ]; then
        cp -f "$SRC_BIN" "$TARGET_DIR/pommitlint"
        chmod +x "$TARGET_DIR/pommitlint"
        [ "$TARGET_DIR" != "$BIN_DIR" ] && cp -f "$SRC_BIN" "$BIN_DIR/pommitlint" && chmod +x "$BIN_DIR/pommitlint"
        echo "[Git-Plus] 成功安装至 $TARGET_DIR/pommitlint"
      fi
    else
      echo "[Git-Plus] Linux 环境下载 GitHub Release 预编译包..."
      TMP_DIR="$(mktemp -d 2>/dev/null || mktemp -d -t 'pommitlint')"
      DOWNLOAD_URL="https://github.com/shuymn/pommitlint/releases/download/${VERSION}/pommitlint_${VERSION}_linux_${ARCH}.tar.gz"
      if curl -fsSL "$DOWNLOAD_URL" -o "$TMP_DIR/pommitlint.tar.gz"; then
        tar -zxf "$TMP_DIR/pommitlint.tar.gz" -C "$TMP_DIR"
        if [ -f "$TMP_DIR/pommitlint" ]; then
          cp -f "$TMP_DIR/pommitlint" "$TARGET_DIR/pommitlint"
          chmod +x "$TARGET_DIR/pommitlint"
          [ "$TARGET_DIR" != "$BIN_DIR" ] && cp -f "$TMP_DIR/pommitlint" "$BIN_DIR/pommitlint" && chmod +x "$BIN_DIR/pommitlint"
          echo "[Git-Plus] 成功安装至 $TARGET_DIR/pommitlint"
        fi
      fi
      rm -rf "$TMP_DIR"
    fi
    ;;
esac
