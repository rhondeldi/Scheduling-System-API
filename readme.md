# Scheduling System Backend

![build](https://github.com/mrdcvlsc/scheduling-system-backend/actions/workflows/build.yml/badge.svg)
![tests](https://github.com/mrdcvlsc/scheduling-system-backend/actions/workflows/tests.yml/badge.svg)
![bench](https://github.com/mrdcvlsc/scheduling-system-backend/actions/workflows/bench.yml/badge.svg)

Backend REST API service for a **Subject Scheduling System Using Genetic Algorithms**. This system optimizes university course scheduling by leveraging genetic algorithms with artificial neural network enhancements to automatically generate conflict-free timetables.

---

## 📋 Overview

This backend service provides a complete REST API for managing university scheduling resources including subjects, instructors, rooms, curricula, and departments. It uses genetic algorithms to intelligently generate optimal schedules while respecting constraints such as instructor availability, room capacity, and curriculum requirements.

### Key Features

- 🧬 **Genetic Algorithm Optimization** - Automated schedule generation using GA with crossover, mutation, and fitness evaluation
- 🏫 **Multi-Department Support** - Manage multiple departments with separate authentication
- 📚 **Resource Management** - Complete CRUD operations for subjects, instructors, rooms, and curricula
- 🔐 **Session-Based Authentication** - Secure department and admin login system
- 💾 **Flexible Persistence** - Support for both JSON file storage and MongoDB
- 🌐 **CORS Enabled** - Ready for cross-origin frontend integration
- ⚡ **High Performance** - Built with Go and Gin framework
- 🧪 **Well Tested** - Comprehensive test suite with CI/CD integration

---

## 🛠️ Technologies

- **Language**: Go 1.24.0+
- **Framework**: Gin Web Framework
- **Database**: MongoDB (optional) or JSON file storage
- **Session Store**: gorilla/sessions
- **CORS**: gin-contrib/cors

---

## 📦 Prerequisites

- **Go** v1.24.0 or higher
- **Git**
- **Node.js** v22.11.0+ (for building frontend)
- **MongoDB** (optional, for database persistence)
- **Make** (for Linux/Mac development)

---

## 🚀 Quick Start

### Installation

Detailed platform-specific setup instructions are available:

- [Windows Setup Guide](WindowsSetup.MD)
- [Ubuntu/Linux Setup Guide](UbuntuLinuxSetup.MD)

### Basic Setup

```bash
# Clone the repository
git clone https://github.com/mrdcvlsc/scheduling-system-backend.git
cd scheduling-system-backend

# Install dependencies
go mod download

# Initialize empty data directory
cp -r scheduling-system-temporary-data-empty scheduling-system-temporary-data

# Build the application
go build -tags netgo -ldflags '-s -w' -o app

# Set environment variables
export GIN_MODE=debug
export PORT=3000
export SESSION_SECRET=your-32-character-secret-here
export AUTH=enable

# Run the server
./app
```

---

## 🔧 Configuration

Configure the application using environment variables:

| Variable         | Description                                 | Default   |
| ---------------- | ------------------------------------------- | --------- |
| `PORT`           | Server port                                 | `3000`    |
| `GIN_MODE`       | Gin mode (`debug`, `release`)               | `debug`   |
| `DEV_MODE`       | Development mode (`local_release`)          | -         |
| `USE_DATABASE`   | Database type (`MongoDB` or empty for JSON) | JSON      |
| `SESSION_SECRET` | 32-char session encryption key              | Required  |
| `MASTER_KEY`     | Admin master key for initial setup          | Optional  |
| `AUTH`           | Enable authentication (`enable`/`disable`)  | `disable` |

---

## 📚 API Endpoints

### Resources Management

#### Subjects

- `GET /api/v1/subjects` - Get all subjects
- `POST /api/v1/subjects` - Create new subject
- `PATCH /api/v1/subjects/:id` - Update subject
- `DELETE /api/v1/subjects/:id` - Delete subject

#### Instructors

- `GET /api/v1/instructors` - Get all instructors
- `POST /api/v1/instructors` - Create new instructor
- `PATCH /api/v1/instructors/:id` - Update instructor
- `DELETE /api/v1/instructors/:id` - Delete instructor

#### Rooms

- `GET /api/v1/rooms` - Get all rooms
- `POST /api/v1/rooms` - Create new room
- `PATCH /api/v1/rooms/:id` - Update room
- `DELETE /api/v1/rooms/:id` - Delete room

#### Curricula

- `GET /api/v1/curricula` - Get all curricula
- `POST /api/v1/curricula` - Create new curriculum
- `PATCH /api/v1/curricula/:id` - Update curriculum
- `DELETE /api/v1/curricula/:id` - Delete curriculum

#### Departments

- `GET /api/v1/departments` - Get all departments
- `POST /api/v1/departments` - Create new department
- `PATCH /api/v1/departments/:id` - Update department
- `DELETE /api/v1/departments/:id` - Delete department

### Schedule Generation

- `POST /api/v1/schedule` - Generate a new schedule
- `GET /api/v1/schedule` - Get department schedule
- `GET /api/v1/schedule/status` - Check schedule generation status
- `DELETE /api/v1/schedule/:id` - Delete schedule
- `GET /api/v1/university-schedule` - Get complete university schedule
- `POST /api/v1/university-schedule` - Generate university-wide schedule

### Authentication

- `POST /api/v1/login` - Department login
- `POST /api/v1/admin/login` - Admin login
- `POST /api/v1/logout` - Logout

---

## 📂 Project Structure

```
scheduling-system-backend/
├── Auth/                       # Authentication & session management
├── GeneticAlgorithm/          # GA implementation
│   ├── Crossover.go
│   ├── FitnessFunction.go
│   ├── RandomMutation.go
│   └── GeneticAlgorithm.go
├── Resources/                  # Resource definitions
│   ├── Curriculum/
│   ├── Departments/
│   ├── Instructors/
│   └── Rooms/
├── Routes/                     # API route handlers
│   ├── RoutesV1/
│   └── RoutesV2/
├── RouteGlobals/              # Global route state
├── Schedule/                   # Schedule data structures
├── StorageResources/          # Resource persistence layer
├── StorageSchedule/           # Schedule persistence layer
├── Utils/                      # Utility functions
├── main.go                     # Application entry point
├── go.mod                      # Go module definition
└── makefile                    # Build automation
```

---

## 🧪 Testing

Run the test suite:

```bash
# Clean test cache
go clean -testcache

# Run all tests
make test

# Run specific test
go test -run TestNewPopulationFirstSem ./GeneticAlgorithm -timeout 0
```

---

## 🔨 Development

### Building

```bash
# Development build
make build_dev

# Production build
make build

# Generate UML diagram
make uml
```

### Download Resources

```bash
# Download test data from GitHub releases
make devr

# Download frontend build
make frontend
```

---

## 🤝 Contributing

Contributions are welcome! Please ensure:

1. All tests pass before submitting PR
2. Follow Go best practices and conventions
3. Update documentation for new features
4. Add tests for new functionality

---

## 📄 License

This project is part of an academic research system for automated university scheduling using genetic algorithms.

---

## 🔗 Related Repositories

- [Frontend Repository](https://github.com/mrdcvlsc/scheduling-system-frontend)
- [Test Data Repository](https://github.com/mrdcvlsc/scheduling-system-temporary-data)

---

## 🐛 Troubleshooting

### Common Issues

**Port already in use:**

```bash
# Change the PORT environment variable
export PORT=3001
```

**Session store initialization failed:**

- Ensure `SESSION_SECRET` is exactly 32 characters
- Check file system permissions for session storage

**MongoDB connection failed:**

- Verify MongoDB is running
- Check MongoDB connection string in environment
- Ensure database credentials are correct

---

## 📞 Support

For issues and questions, please open an issue on the [GitHub repository](https://github.com/mrdcvlsc/scheduling-system-backend/issues).
