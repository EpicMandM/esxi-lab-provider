# Conventional Commits

This project uses [Conventional Commits](https://www.conventionalcommits.org/) specification for commit messages.

## Format

```
<type>(<scope>): <subject>

<body>

<footer>
```

## Types

- **feat**: A new feature
- **fix**: A bug fix
- **docs**: Documentation only changes
- **style**: Changes that don't affect code meaning (formatting, whitespace)
- **refactor**: Code change that neither fixes a bug nor adds a feature
- **perf**: Performance improvement
- **test**: Adding missing tests or correcting existing tests
- **chore**: Changes to build process or auxiliary tools

## Examples

```bash
feat(email): add SMTP email support for password notifications

fix(vmware): resolve permission denied error for lab-user2

docs(readme): update Gmail setup instructions

chore(deps): update golangci-lint to v2.5.0

refactor(service): replace Gmail API with SMTP implementation
```

## Scope (optional)

Common scopes in this project:
- `email` - Email service
- `vmware` - VMware/vSphere integration
- `calendar` - Google Calendar integration
- `docker` - Docker/containerization
- `terraform` - Infrastructure as code
- `deps` - Dependencies

## Breaking Changes

Add `BREAKING CHANGE:` in footer or `!` after type:

```bash
feat(api)!: change password rotation API structure

BREAKING CHANGE: Password rotation now returns map instead of array
```
