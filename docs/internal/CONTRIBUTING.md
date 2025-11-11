# Contributing Guide

Thank you for your interest in contributing to the Pizza API! This guide explains the contribution process, development workflow, and code review standards.

---

## Table of Contents

- [How to Contribute](#how-to-contribute)
- [Development Process](#development-process)
- [Commit Message Format](#commit-message-format)
- [Code Review Checklist](#code-review-checklist)
- [Automated Testing (CI/CD)](#automated-testing-cicd)

---

## How to Contribute

We welcome contributions of all kinds:

- 🐛 **Bug fixes**
- ✨ **New features**
- 📚 **Documentation improvements**
- 🧪 **Test coverage**
- ♻️ **Code refactoring**
- 🎨 **UI/UX improvements** (Swagger docs)

---

## Development Process

### 1. Fork the Repository

Click the "Fork" button on GitHub to create your own copy.

### 2. Clone Your Fork

```bash
git clone https://github.com/YOUR_USERNAME/gin-pizza-api.git
cd gin-pizza-api
```

### 3. Create a Feature Branch

```bash
git checkout -b feature/your-feature-name
```

**Branch naming conventions:**
- `feature/` - New features
- `fix/` - Bug fixes
- `docs/` - Documentation changes
- `test/` - Test additions/improvements
- `refactor/` - Code refactoring

**Examples:**
```bash
git checkout -b feature/add-pizza-search
git checkout -b fix/oauth-token-expiration
git checkout -b docs/update-readme
```

### 4. Make Your Changes

**Before coding:**
- Read the [Development Guide](DEVELOPMENT.md) for project structure and conventions
- Check existing issues/PRs to avoid duplicate work
- Open an issue for discussion if making significant changes

**While coding:**
- Follow Go conventions and code style (see [Development Guide](DEVELOPMENT.md#code-style-and-formatting))
- Add tests for new functionality
- Update documentation if needed

### 5. Add Tests

**Unit tests** for new functionality:
```go
func TestGetPizzaByID(t *testing.T) {
    // Test implementation
}
```

**Run tests:**
```bash
go test ./...
```

### 6. Update Documentation

**Required documentation updates:**
- Add Swagger annotations to new endpoints
- Update README.md if adding user-facing features
- Update internal docs (`docs/internal/`) if changing architecture
- Regenerate Swagger documentation:
  ```bash
  swag init -g cmd/main.go
  ```

### 7. Ensure Tests Pass

```bash
# Run unit tests
go test ./...

# Run integration tests
./scripts/test-api.sh
```

### 8. Format Code

```bash
# Auto-format Go code
gofmt -w .

# Run linter (optional but recommended)
golangci-lint run
```

### 9. Commit Changes

Follow the [Commit Message Format](#commit-message-format) below.

```bash
git add .
git commit -m "feat: add pizza search by name"
```

### 10. Push to Your Fork

```bash
git push origin feature/your-feature-name
```

### 11. Create Pull Request

1. Go to the original repository on GitHub
2. Click "New Pull Request"
3. Select your fork and branch
4. Fill out the PR template:
   - **Description**: What does this PR do?
   - **Motivation**: Why is this change needed?
   - **Testing**: How did you test this?
   - **Screenshots**: If UI changes

---

## Commit Message Format

We follow [Conventional Commits](https://www.conventionalcommits.org/) for clear, semantic commit history.

### Format

```
<type>: <subject>

<body> (optional)

<footer> (optional)
```

### Types

- **`feat:`** - New feature
- **`fix:`** - Bug fix
- **`docs:`** - Documentation changes
- **`test:`** - Adding or updating tests
- **`refactor:`** - Code refactoring (no functional changes)
- **`chore:`** - Maintenance tasks (dependencies, CI config)
- **`style:`** - Code style changes (formatting, whitespace)
- **`perf:`** - Performance improvements

### Examples

**Feature:**
```
feat: add pizza search endpoint

Add GET /api/v1/public/pizzas?name=Margherita endpoint
to search pizzas by name. Supports partial matching.
```

**Bug Fix:**
```
fix: correct OAuth token expiration time

Token expiration was set to 3600ms instead of 3600s.
Fixed to use time.Second instead of time.Millisecond.
```

**Documentation:**
```
docs: update API documentation with query parameters

Added documentation for new query parameter filtering
in API_CONTRACT.md.
```

**Test:**
```
test: add unit tests for pizza service

Add test coverage for GetPizzaByID and CreatePizza methods.
```

**Refactor:**
```
refactor: extract token generation to separate function

Move JWT token generation logic from oauth_server.go
to jwt_generator.go for better separation of concerns.
```

### Subject Line Rules

- Use imperative mood ("add" not "added" or "adds")
- Don't capitalize first letter
- No period at the end
- Keep under 50 characters if possible

### Body (Optional)

- Explain **what** and **why**, not **how**
- Wrap at 72 characters
- Separate from subject with blank line

### Footer (Optional)

**Reference issues:**
```
Closes #123
Fixes #456
```

**Breaking changes:**
```
BREAKING CHANGE: remove deprecated /v1/pizzas endpoint

All clients must migrate to /api/v1/public/pizzas.
```

---

## Code Review Checklist

Before requesting review, verify:

### Functionality
- [ ] Feature works as intended
- [ ] No breaking changes (or documented with migration guide)
- [ ] Edge cases handled
- [ ] Error handling is appropriate

### Code Quality
- [ ] Code follows Go conventions
- [ ] No hardcoded values (use constants/config)
- [ ] Functions are focused and single-purpose
- [ ] Variable names are descriptive
- [ ] No commented-out code

### Testing
- [ ] Unit tests added and passing
- [ ] Integration tests pass (`./scripts/test-api.sh`)
- [ ] Test coverage is adequate
- [ ] Tests cover edge cases

### Documentation
- [ ] Swagger annotations added to new endpoints
- [ ] Swagger docs regenerated (`swag init -g cmd/main.go`)
- [ ] README.md updated if user-facing changes
- [ ] Internal docs updated if architecture changes
- [ ] Code comments explain complex logic

### Security
- [ ] No secrets in code or commit history
- [ ] Input validation on user-provided data
- [ ] SQL injection prevention (use GORM properly)
- [ ] Authorization checks on protected endpoints

### Performance
- [ ] No N+1 query problems
- [ ] Database indexes on frequently queried columns
- [ ] No blocking operations in request handlers
- [ ] Resource cleanup (defer file.Close(), etc.)

---

## Automated Testing (CI/CD)

### Continuous Integration Pipeline

All pull requests automatically trigger our CI pipeline via GitHub Actions (`.github/workflows/ci.yml`).

#### What Gets Tested

1. **Unit Tests**: All Go unit tests with race detector enabled
2. **Build Verification**: Application builds successfully
3. **Integration Tests**: Full `test-api.sh` suite (OAuth + CRUD lifecycle)
4. **Code Linting**: golangci-lint checks code quality

#### Test Execution

**Unit tests:**
```bash
go test -v -race ./...
```

**Integration tests:**
```bash
./scripts/test-api.sh
```

Both must pass for PR approval.

#### Debugging Test Failures

- CI captures server logs on failure
- Download logs from GitHub Actions artifacts (retained 7 days)
- Run tests locally first: `./scripts/test-api.sh`

#### Local CI Simulation

Test your changes locally before pushing:

```bash
# Run what CI will run
go test -v -race ./...
go build -o bin/pizza-api cmd/main.go
./scripts/test-api.sh
```

### Branch Protection

- `main` branch requires passing CI checks
- Cannot merge with failing tests
- At least one approval required (team configuration)

---

## Pull Request Guidelines

### PR Title Format

Follow commit message format:
```
feat: add pizza search by name
fix: correct OAuth token expiration
docs: update CONTRIBUTING.md
```

### PR Description Template

```markdown
## Description
Brief summary of changes.

## Motivation
Why is this change needed? What problem does it solve?

## Changes
- Added pizza search endpoint
- Updated Swagger documentation
- Added unit tests

## Testing
- ✅ Unit tests pass
- ✅ Integration tests pass
- ✅ Tested locally with curl

## Screenshots (if applicable)
[Include screenshots for UI changes]

## Checklist
- [x] Tests added and passing
- [x] Documentation updated
- [x] Code formatted with gofmt
- [x] Swagger docs regenerated
```

### Review Process

1. **Automated checks** run on PR creation
2. **Code review** by maintainers (usually within 2-3 days)
3. **Address feedback** by pushing new commits to your branch
4. **Approval** once all checks pass and feedback addressed
5. **Merge** by maintainer (squash and merge or rebase)

---

## Getting Help

**Questions about contributing?**
- Open a [GitHub Discussion](https://github.com/franciscosanchezn/gin-pizza-api/discussions)
- Check existing [Issues](https://github.com/franciscosanchezn/gin-pizza-api/issues)
- Email: support@pizza-api.local (placeholder)

**Found a bug?**
- Search existing issues first
- Open a new issue with:
  - Clear description
  - Steps to reproduce
  - Expected vs actual behavior
  - Environment details (OS, Go version)

**Want to propose a feature?**
- Open an issue first for discussion
- Explain the use case and benefits
- Wait for maintainer feedback before implementing

---

## Code of Conduct

### Our Standards

- Be respectful and inclusive
- Accept constructive criticism
- Focus on what's best for the project
- Show empathy towards others

### Unacceptable Behavior

- Harassment or discriminatory language
- Trolling or insulting comments
- Publishing others' private information
- Other unprofessional conduct

### Enforcement

Violations may result in temporary or permanent ban from the project.

---

## Recognition

Contributors are recognized in:
- GitHub Contributors page
- CHANGELOG.md (for significant contributions)
- README.md Acknowledgments section

---

## Additional Resources

- [Development Guide](DEVELOPMENT.md) - Project structure, coding standards
- [Operations Guide](OPERATIONS.md) - Deployment and troubleshooting
- [JWT Internals](JWT_INTERNALS.md) - Authentication architecture
- [API Contract](../API_CONTRACT.md) - API specifications

---

**Thank you for contributing to Pizza API!** 🍕
