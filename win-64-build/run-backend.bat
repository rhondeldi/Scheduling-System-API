@echo off
REM ───────────────────────────────────────────────────────────────
REM run-backend.bat - this is only for local setups
REM ── Session‑only env vars ─────────────────────────────────────
setlocal

echo > go.mod
set GIN_MODE=debug
set DEV_MODE=local_release
set AUTH=enable
set PORT=3000

REM 32‑char secrets (change secrets when setting up)
set SESSION_SECRET=xyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxy
set MASTER_KEY=

REM ── Loop to keep the app running ───────────────────────────────
:RESTART
echo [%DATE% %TIME%] Starting app.exe...
scheduling-backend.exe

echo.
echo --------------------------------------------
echo scheduling-backend.exe exited with code %ERRORLEVEL%.
echo Restarting in 3 seconds…
timeout /t 3 /nobreak >nul
echo.
goto RESTART