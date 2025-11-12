# Contributing

Thank you for considering contributing to this Go API Starter template!

## Development Setup

1. **Fork and clone the repository**

   ```bash
   git clone https://github.com/yourusername/go-chi-postgres-starter.git
   cd go-chi-postgres-starter
   ```

2. **Install dependencies**

   ```bash
   go mod tidy
   ```

3. **Set up environment**

   ```bash
   cp .env.example .env
   # Edit .env with your configuration
   # Note: .env.example is provided as a template
   ```

4. **Set up database**

   ```bash
   createdb go_api_starter
   make migrate-up
   ```

5. **Run the application**

   ```bash
   make run
   ```

## Making Changes

1. Create a feature branch

   ```bash
   git checkout -b feature/your-feature-name
   ```

2. Make your changes following the project's architecture:
   - **Handlers** - HTTP request/response handling
   - **Services** - Business logic
   - **Repository** - Database operations
   - **Models** - Data structures

3. **Run tests**

   ```bash
   make test
   ```

4. **Run linters**

   ```bash
   make lint
   ```

5. **Format code**

   ```bash
   make fmt
   ```

6. **Commit your changes**

   ```bash
   git commit -m "Add: description of your changes"
   ```

7. **Push and create a Pull Request**

## Code Style

- Follow Go conventions and best practices
- Use meaningful variable and function names
- Add comments for exported functions and types
- Keep functions focused and small
- Handle errors explicitly (no silent failures)

## Testing

- Write tests for new features
- Ensure all tests pass before submitting
- Aim for good test coverage

## Pull Request Process

1. Update the README.md if needed
2. Ensure all tests pass
3. Update documentation if adding new features
4. Request review from maintainers

Thank you for contributing! 🎉
