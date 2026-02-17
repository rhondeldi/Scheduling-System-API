# Windows 10/11 Setup (CMD)

> All commands below are for the default Command Prompt (CMD.exe).  
> Paste each line and press Enter; comments start with `REM`.

---

## Requirements

1. **Go** v1.24.0 or above  
2. **Git**  
3. **Node.js** v22.11.0 or above  

*(Download/install these before continuing.)*

---

## Local Release Build (with file-system persistence + login)

1. **Open Command Prompt**, then enter the following commands:

```bat
REM create & enter project folder
mkdir scheduling-system
cd scheduling-system

REM clone the backend and frontend repositories
git clone https://github.com/mrdcvlsc/scheduling-system-frontend.git
git clone https://github.com/mrdcvlsc/scheduling-system-backend.git

REM install frontend deps
cd scheduling-system-frontend
npm install

REM prepare backend
cd ..\scheduling-system-backend

REM copy empty data folder (requires /E to include all subfolders, /I to assume destination is a folder)
xcopy .\scheduling-system-temporary-data-empty .\scheduling-system-temporary-data /E /I

REM build frontend backend
go build -tags netgo -ldflags "-s -w" -o app.exe

REM remove old frontend build output if there is any
rmdir /S /Q dist

REM flip DEV flag to “false” in basics.js
powershell -Command ^
  "(Get-Content ..\scheduling-system-frontend\src\js\basics.js) -replace 'const DEV = true','const DEV = false' | Set-Content ..\scheduling-system-frontend\src\js\basics.js"

REM build backend
cd ..\scheduling-system-frontend
npm run build
xcopy dist ..\scheduling-system-backend\dist\*.* /E /I /Y

REM restore DEV flag to “true” in basics.js
powershell -Command ^
  "(Get-Content src\js\basics.js) -replace 'const DEV = false','const DEV = true' | Set-Content src\js\basics.js"

REM go back to the backend folder
cd ..\scheduling-system-backend

REM set session-only environment variables
set GIN_MODE=debug
set DEV_MODE=local_release
set AUTH=enable
set PORT=3000

REM 32-char secrets (change once, then clear MASTER_KEY)
set SESSION_SECRET=xyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxy
set MASTER_KEY=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa

REM run the backend
app.exe
```

2. Copy the displayed IP address and port from the terminal, then paste it into your browser's address bar.
3. Use the `MASTER_KEY` to log in.
4. Create a new department. Remember the Department ID number, Department Code, and Password.
5. Delete the `DEFAULT` department.
6. Stop the `app` process (usually by pressing `Ctrl+C` in the terminal).

7. Remove master key in environment variable then start the app.exe again:

```bash
# remove the master key so it cannot be used by users
set MASTER_KEY=
app.exe
```

8. Login using the new department with password (other departments can create new department if needed, but only the department itself can delete themselves).

Other devices that is also connected to the same network should be able to connect to the same ip address and port (no data needed) using their browsers.

---

## Local Development Build (no login)

Open **two** CMD windows.

### Backend window

```bat
cd scheduling-system\scheduling-system-backend
go build -tags netgo -ldflags "-s -w" -o app.exe

REM session-only variables:
set PORT=3000
set GIN_MODE=debug
set DEV_MODE=
set AUTH=off

REM run:
app.exe
```

### Frontend window

```bat
cd scheduling-system\scheduling-system-frontend
npm run dev
```

Browse to the URL shown (e.g. `http://localhost:5173`).

---

## (Optional) Use MongoDB instead of in-memory

Before running `app.exe`, in CMD:

```bat
set USE_DATABASE=MongoDB
set MONGODB_CONNECTION_STRING=your-Go-MongoDB-connection-string
app.exe
```

---

## Deployment to a Web Service

Configure your host (Azure/AWS/etc.) with these environment variables:

```
AUTH=enable
GIN_MODE=release
MASTER_KEY=<32-hex-chars>        REM remove after initial setup
SESSION_SECRET=<32-hex-chars>
USE_DATABASE=MongoDB
MONGODB_CONNECTION_STRING=<your-connection-string>
```
