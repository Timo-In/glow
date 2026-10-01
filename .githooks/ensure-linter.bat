@echo off
setlocal enabledelayedexpansion

:: Git-Plus Linter Installer for Windows CMD & PowerShell
set "VERSION=v2.0.1"

for /f "delims=" %%i in ('git rev-parse --show-toplevel 2^>nul') do set "REPO_ROOT=%%i"
if "%REPO_ROOT%"=="" set "REPO_ROOT=%~dp0.."

set "HOOKS_DIR=%REPO_ROOT%\.githooks"
set "BIN_DIR=%HOOKS_DIR%\bin"

where pommitlint >nul 2>&1
if %ERRORLEVEL% EQU 0 exit /b 0

if exist "%BIN_DIR%\pommitlint.exe" exit /b 0

where commitlint >nul 2>&1
if %ERRORLEVEL% EQU 0 exit /b 0

echo [Git-Plus] 正在为 Windows 环境准备 Commit 规范校验工具 (pommitlint.exe)...

if not exist "%BIN_DIR%" mkdir "%BIN_DIR%"

set "ARCH=amd64"
if "%PROCESSOR_ARCHITECTURE%"=="ARM64" set "ARCH=arm64"

set "DOWNLOAD_URL=https://github.com/shuymn/pommitlint/releases/download/%VERSION%/pommitlint_%VERSION%_windows_%ARCH%.zip"
set "ZIP_FILE=%TEMP%\pommitlint_%RANDOM%.zip"
set "EXTRACT_DIR=%TEMP%\pommitlint_ext_%RANDOM%"

curl -fsSL "%DOWNLOAD_URL%" -o "%ZIP_FILE%"
if %ERRORLEVEL% NEQ 0 (
    echo [Git-Plus] 下载失败，请检查网络连接。
    exit /b 1
)

mkdir "%EXTRACT_DIR%" 2>nul
powershell -Command "Expand-Archive -Path '%ZIP_FILE%' -DestinationPath '%EXTRACT_DIR%' -Force" 2>nul
if not exist "%EXTRACT_DIR%\pommitlint.exe" (
    tar -xf "%ZIP_FILE%" -C "%EXTRACT_DIR%" 2>nul
)

if exist "%EXTRACT_DIR%\pommitlint.exe" (
    copy /y "%EXTRACT_DIR%\pommitlint.exe" "%BIN_DIR%\pommitlint.exe" >nul
    echo [Git-Plus] 成功安装至 %BIN_DIR%\pommitlint.exe
)

del /f /q "%ZIP_FILE%" 2>nul
rd /s /q "%EXTRACT_DIR%" 2>nul

exit /b 0
