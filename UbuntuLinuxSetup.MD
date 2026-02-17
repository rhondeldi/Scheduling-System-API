# Ubuntu Linux Setup

## Requirements

1. `go` v1.24.0 or above  
2. `git`  
3. `node` (Node.js) - v22.11.0 or above *(npm is usually included when you install Node.js)*  
4. `make` v4.3 or above

### Local Release Build Commands - File System Persistence

1. Enter the following commands in the terminal, line by line:

```bash
# create the project folder
mkdir scheduling-system

# enter the project folder
cd scheduling-system

# download the frontend code
git clone https://github.com/mrdcvlsc/scheduling-system-frontend.git

# download the backend code
git clone https://github.com/mrdcvlsc/scheduling-system-backend.git

# enter the frontend folder
cd scheduling-system-frontend

# download frontend dependencies
npm install

# go back to the parent project folder
cd ..

# enter the backend folder
cd scheduling-system-backend

# initialize an empty file system persistence
cp -r scheduling-system-temporary-data-empty scheduling-system-temporary-data

# build the backend code
go build -tags netgo -ldflags '-s -w' -o app

# build the frontend code
make frontend_local_release

# set environment variables
export GIN_MODE=debug
export DEV_MODE=local_release

# 32-character session secret - can be anything
# this should not be changed after this initial setup
export SESSION_SECRET=xyzxyzxyzxyzxyzxyzxyzxyzxyzxyzxy

# 32-character master key - can be anything
export MASTER_KEY=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
export AUTH=enable
export PORT=3000

# start serving the web application
./app
````

2. Copy the displayed IP address and port from the terminal, then paste it into your browser's address bar.
3. Use the `MASTER_KEY` to log in.
4. Create a new department. Remember the Department ID number, Department Code, and Password.
5. Delete the `DEFAULT` department.
6. Stop the `app` process (usually by pressing `Ctrl+C` in the terminal).
7. Enter the command below in the terminal:

```bash
# remove the master key so it cannot be used by users
export MASTER_KEY=
```

8. Run the backend `app` process again using `./app`, then log in with the new department credentials.

Other devices that is also connected to the same network should be able to connect to the same ip address and port (no data needed) using their browsers.

(other departments can create new department if needed, but only the department itself can delete themselves).

---

### Local Development Build - File System Persistence (No Login)

Open two terminals.

#### In the Backend Terminal, enter the following commands:

```bash
# enter the backend folder
cd scheduling-system-backend

# build the development server
go build -tags netgo -ldflags '-s -w' -o app

# set environment variables
export PORT=3000
export GIN_MODE=debug
export DEV_MODE=
export AUTH=off

# run the backend server
./app
```

#### In the Frontend Terminal:

1. Enter the following commands:

```bash
# enter the frontend folder
cd scheduling-system-frontend

# run the frontend development server
npm run dev
```

2. Copy the localhost address and port displayed in the terminal, then paste it into your browser.

---

### Use MongoDB

to use mongodb you can set the environment variables below before running the `./app`

```bash
export USE_DATABASE=MongoDB
export MONGODB_CONNECTION_STRING="your-go-lang-mongodb-connection-string"
```

---

### When deploying to a web service

Set the following environment variables

```bash
AUTH=enable
GIN_MODE=release
MASTER_KEY=a-32-hex-character-string-no-white-space-remove-after-setup
USE_DATABASE=MongoDB
MONGODB_CONNECTION_STRING=your-mongodb-connection-string
SESSION_SECRET=a-32-hex-character-string-no-white-space
```