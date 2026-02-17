@echo off
SETLOCAL ENABLEDELAYEDEXPANSION

REM Set your repo details
SET github_user=mrdcvlsc
SET github_repo=scheduling-system-backend
SET target=win-64-build.zip
SET extract_dir=_temp_extract_dir

REM Clean up previous extract folder if exists
IF EXIST "%extract_dir%" rd /S /Q "%extract_dir%"

FOR /f "tokens=1,* delims=:" %%A IN (
  'curl -s https://api.github.com/repos/%github_user%/%github_repo%/releases/latest ^| findstr "browser_download_url"'
) DO (
  SET url=%%B
  IF NOT "!url:%target%=!"=="!url!" (
    ECHO Downloading !url!
    curl -L -o "%target%" !url!

    REM --- extract ZIP into temporary folder ---
    powershell -NoProfile -Command ^
      "Expand-Archive -LiteralPath '%target%' -DestinationPath '%extract_dir%' -Force"

    REM --- copy contents of 'win-64-build' subfolder into current dir ---
    xcopy "%extract_dir%\win-64-build\*" "." /E /H /Y

    REM --- cleanup ---
    REM del "%target%"
    rd /S /Q "%extract_dir%"
  )
)

pause
