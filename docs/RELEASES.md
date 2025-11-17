# GitHub Release Workflow Guide

This repository uses **[semantic-release](https://semantic-release.gitbook.io/)** to automate version management and GitHub releases.

## How It Works

When changes are merged to `main`, the workflow automatically:

1. Analyzes commits using conventional commit format
2. Determines the next version (MAJOR.MINOR.PATCH)
3. Generates release notes with categorized commits
4. Creates a GitHub release with version tag and assets
5. Uploads binary and Docker image to the release

**No manual version management required!**

## Semantic Version Calculation

The workflow automatically determines the next version by analyzing **conventional commits** since the last tag:

| Commit Type | Version Bump | Example |
|-------------|--------------|---------|
| `feat!:` or `BREAKING CHANGE:` | **MAJOR** (v1.0.0 → v2.0.0) | Breaking API changes |
| `feat:` | **MINOR** (v1.0.0 → v1.1.0) | New features (backward compatible) |
| `fix:` | **PATCH** (v1.0.0 → v1.0.1) | Bug fixes |
| `perf:` | **PATCH** (v1.0.0 → v1.0.1) | Performance improvements |
| `refactor:` | **PATCH** (v1.0.0 → v1.0.1) | Code refactoring |

## Conventional Commit Format

Use this format for your commits:

```
<type>[optional scope][!]: <description>

[optional body]

[optional footer(s)]
```

**Examples:**

```bash
# PATCH bump (v1.0.0 → v1.0.1)
git commit -m "fix: correct ownership validation in pizza controller"

# MINOR bump (v1.0.0 → v1.1.0)
git commit -m "feat: add USER role CRUD operations"
git commit -m "feat(auth): enable user role pizza management"

# MAJOR bump (v1.0.0 → v2.0.0)
git commit -m "feat!: migrate API endpoints to simplified structure"
# OR
git commit -m "feat: change API paths

BREAKING CHANGE: Pizza endpoints moved from /protected/admin/pizzas to /pizzas"
```

### Commit Types

- **feat**: New feature → MINOR bump
- **fix**: Bug fix → PATCH bump
- **perf**: Performance improvement → PATCH bump
- **refactor**: Code refactoring → PATCH bump
- **docs**: Documentation only → No release
- **style**: Code formatting → No release
- **test**: Test changes → No release
- **chore**: Maintenance → No release
- **ci**: CI/CD changes → No release

## Example Workflow

```bash
# Develop your feature with conventional commits
git checkout -b feat/user-roles
git commit -m "feat: add USER role support"
git commit -m "feat: implement ownership checks"
git commit -m "test: add USER role integration tests"
git commit -m "docs: update API documentation"
git push origin feat/user-roles

# Create PR and merge to main
# The workflow automatically:
# 1. Analyzes commits (found "feat:" commits)
# 2. Determines MINOR version bump (v1.0.0 → v1.1.0)
# 3. Creates release v1.1.0 with changelog and assets
```

## Release Assets

Each release automatically includes:

1. **Binary** (`pizza-api-linux-amd64`)
   - Compiled Go binary for Linux
   
2. **Docker Image** (`pizza-api-docker-vX.Y.Z.tar.gz`)
   - Compressed Docker image tarball

3. **Release Notes**
   - Automatically generated with categorized commits and emojis

## Using Released Versions

### Docker

```bash
# Download release
wget https://github.com/franciscosanchezn/gin-pizza-api/releases/download/v1.1.0/pizza-api-docker-v1.1.0.tar.gz

# Import to Docker
gunzip pizza-api-docker-v1.1.0.tar.gz
docker load < pizza-api-docker-v1.1.0.tar

# Run
docker run -p 8080:8080 pizza-api:latest
```

### MicroK8s

```bash
# Import to MicroK8s
microk8s ctr image import pizza-api-docker-v1.1.0.tar

# Update deployment
microk8s kubectl set image deployment/pizza-api pizza-api=pizza-api:latest
microk8s kubectl rollout status deployment/pizza-api
```

### Binary

```bash
# Download and run
wget https://github.com/franciscosanchezn/gin-pizza-api/releases/download/v1.1.0/pizza-api-linux-amd64
chmod +x pizza-api-linux-amd64
./pizza-api-linux-amd64
```

## Skip Release

To commit without triggering a release, add `[skip ci]` or use non-release commit types:

```bash
git commit -m "docs: update README [skip ci]"
git commit -m "chore: update dependencies"  # No release (chore type)
```

## Configuration

Release automation is configured in:

- **`.releaserc.json`**: Semantic-release configuration
- **`.github/workflows/release.yml`**: GitHub Actions workflow

## Viewing Releases

Browse all releases at:
```
https://github.com/franciscosanchezn/gin-pizza-api/releases
```

## Troubleshooting

**No release created after merge:**
- Check commit messages follow conventional format
- Verify at least one `feat:` or `fix:` commit exists
- Review GitHub Actions logs for errors

**Wrong version bump:**
- Use `feat:` for MINOR bumps
- Use `fix:` for PATCH bumps  
- Use `feat!:` or `BREAKING CHANGE:` for MAJOR bumps

## References

- [Semantic Release Documentation](https://semantic-release.gitbook.io/)
- [Conventional Commits Specification](https://www.conventionalcommits.org/)
- [Conventional Commits Guide](./CONVENTIONAL_COMMITS.md)

```

### MicroK8s

```bash
# Import to MicroK8s
microk8s ctr image import pizza-api-docker-v1.1.0.tar

# Update deployment
microk8s kubectl set image deployment/pizza-api pizza-api=pizza-api:latest
microk8s kubectl rollout status deployment/pizza-api
```

### Binary

```bash
# Download and run
wget https://github.com/franciscosanchezn/gin-pizza-api/releases/download/v1.1.0/pizza-api-linux-amd64
chmod +x pizza-api-linux-amd64
./pizza-api-linux-amd64
```

## Skip Release

To commit without triggering a release, add `[skip ci]` or use non-release commit types:

```bash
git commit -m "docs: update README [skip ci]"
git commit -m "chore: update dependencies"  # No release (chore type)
```

## Configuration

Release automation is configured in:

- **`.releaserc.json`**: Semantic-release configuration
- **`.github/workflows/release.yml`**: GitHub Actions workflow

## Viewing Releases

Browse all releases at:
```
https://github.com/franciscosanchezn/gin-pizza-api/releases
```

## Troubleshooting

**No release created after merge:**
- Check commit messages follow conventional format
- Verify at least one `feat:` or `fix:` commit exists
- Review GitHub Actions logs for errors

**Wrong version bump:**
- Use `feat:` for MINOR bumps
- Use `fix:` for PATCH bumps  
- Use `feat!:` or `BREAKING CHANGE:` for MAJOR bumps

## References

- [Semantic Release Documentation](https://semantic-release.gitbook.io/)
- [Conventional Commits Specification](https://www.conventionalcommits.org/)
- [Conventional Commits Guide](./CONVENTIONAL_COMMITS.md)
