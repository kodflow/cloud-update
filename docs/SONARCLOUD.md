# SonarCloud Configuration Guide

This guide explains how to configure SonarCloud for the Cloud Update project to track code quality and test coverage.

## Prerequisites

1. A SonarCloud account (free for open-source projects)
2. GitHub repository owner/admin access
3. SonarCloud organization created

## Setup Steps

### 1. Create SonarCloud Account and Organization

1. Go to [SonarCloud](https://sonarcloud.io/)
2. Sign in with your GitHub account
3. Create a new organization or use existing one (e.g., `kodflow`)
4. Import your repository `kodflow/cloud-update`

### 2. Generate SonarCloud Token

1. In SonarCloud, go to **My Account** → **Security**
2. Generate a new token with a descriptive name (e.g., `cloud-update-ci`)
3. Copy the token (you won't be able to see it again)

### 3. Configure GitHub Repository Secrets

Add the following secret to your GitHub repository:

1. Go to your repository on GitHub
2. Navigate to **Settings** → **Secrets and variables** → **Actions**
3. Click **New repository secret**
4. Add the following secret:
   - Name: `SONAR_TOKEN`
   - Value: _[paste the token from step 2]_

### 4. Configure SonarCloud Project

1. In SonarCloud, go to your project settings
2. Set up the following:

#### Quality Gate

- Use the default "Sonar way" quality gate or create a custom one
- Recommended settings for Go projects:
  - Coverage on new code: 80%
  - Duplicated lines: < 3%
  - Maintainability rating: A
  - Security rating: A
  - Reliability rating: A

#### Analysis Scope

The project is already configured with proper exclusions in `sonar-project.properties`:

- Excludes test files from source analysis
- Excludes vendor and generated files
- Includes test files for test coverage

### 5. Project Configuration Files

#### sonar-project.properties

Already configured in the repository with:

```properties
sonar.projectKey=kodflow_cloud-update
sonar.organization=kodflow
sonar.go.coverage.reportPaths=coverage.out
```

#### GitHub Actions Workflow

The CI workflow (`.github/workflows/ci.yml`) includes:

- Test job that generates coverage report
- SonarCloud job that uploads results
- Automatic analysis on every push and PR

## Usage

### Running Coverage Locally

Generate coverage report locally:

```bash
make test/coverage
```

This will create a `coverage.out` file with the coverage data.

### Viewing Coverage Report Locally

View coverage in terminal:

```bash
go tool cover -func=coverage.out
```

View HTML coverage report:

```bash
go tool cover -html=coverage.out -o coverage.html
open coverage.html  # macOS
# or
xdg-open coverage.html  # Linux
```

### CI/CD Integration

The GitHub Actions workflow automatically:

1. Runs tests with coverage on every push/PR
2. Uploads coverage report to SonarCloud
3. Comments on PRs with quality gate status
4. Blocks merge if quality gate fails (optional)

## Monitoring

### SonarCloud Dashboard

After setup, you can monitor:

- Code coverage percentage
- Code smells and technical debt
- Security vulnerabilities
- Code duplications
- Overall project health

Access your dashboard at:

```
https://sonarcloud.io/dashboard?id=kodflow_cloud-update
```

### PR Decoration

SonarCloud will automatically comment on pull requests with:

- Quality gate status (Passed/Failed)
- New code coverage
- New issues introduced
- Security hotspots

## Troubleshooting

### Coverage Not Showing

- Ensure `coverage.out` is generated correctly
- Check that the file path in workflow matches
- Verify SONAR_TOKEN is set correctly

### Analysis Failing

- Check SonarCloud logs in GitHub Actions
- Verify project key and organization match
- Ensure token has proper permissions

### Quality Gate Failing

- Review the quality gate conditions
- Check coverage on new code (not overall)
- Fix critical/blocker issues first

## Best Practices

1. **Maintain High Coverage**: Aim for >80% coverage on new code
2. **Fix Issues Promptly**: Don't let technical debt accumulate
3. **Review Security Hotspots**: Address security issues immediately
4. **Monitor Trends**: Watch coverage trend over time
5. **Customize Rules**: Adjust rules to match your team's standards

## Additional Resources

- [SonarCloud Documentation](https://docs.sonarcloud.io/)
- [SonarCloud GitHub Action](https://github.com/SonarSource/sonarcloud-github-action)
- [Go Test Coverage](https://go.dev/blog/cover)
- [Quality Gates](https://docs.sonarcloud.io/improving/quality-gates/)
