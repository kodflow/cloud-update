# CI/CD Workflow Structure

## Workflow Execution Flow

```mermaid
graph LR
    Start([Push/PR]) --> Version[Version]
    Version --> Analyze[Analyze]
    Version --> Test[Test]
    
    Analyze --> Build[Build]
    Test --> Build
    
    Build --> E2E1[E2E Alpine]
    Build --> E2E2[E2E Ubuntu]
    Build --> E2E3[E2E Debian]
    
    E2E1 --> Release[Release]
    E2E2 --> Release
    E2E3 --> Release
    
    style Version fill:#f9f,stroke:#333,stroke-width:2px
    style Analyze fill:#bbf,stroke:#333,stroke-width:2px
    style Test fill:#bbf,stroke:#333,stroke-width:2px
    style Build fill:#bfb,stroke:#333,stroke-width:2px
    style E2E1 fill:#fbf,stroke:#333,stroke-width:2px
    style E2E2 fill:#fbf,stroke:#333,stroke-width:2px
    style E2E3 fill:#fbf,stroke:#333,stroke-width:2px
    style Release fill:#fbb,stroke:#333,stroke-width:2px
```

## Jobs Description

### 1. **Version** (5s)
- Determines version number
- Outputs version for all other jobs

### 2. **Analyze** & **Test** (in parallel ~1m)
- **Analyze**: Code quality checks with golangci-lint
- **Test**: Unit tests with Bazel

### 3. **Build** (1m)
- Waits for both Analyze and Test to succeed
- Builds Linux binary for E2E tests
- Creates artifact for E2E tests

### 4. **E2E** (in parallel ~1m30s)
- Tests on Alpine (OpenRC)
- Tests on Ubuntu (systemd)
- Tests on Debian (sysvinit)

### 5. **Release** (only on main)
- Creates multi-platform binaries
- Creates GitHub release

## Total Time
- **PR**: ~3-4 minutes (Version → Analyze/Test → Build → E2E)
- **Main**: +2 minutes for Release

## Optimizations
- ✅ Parallel execution of Analyze and Test
- ✅ Parallel execution of E2E tests
- ✅ Single binary build reused for all E2E tests
- ✅ Cached dependencies with Bazel
- ✅ Matrix strategy for E2E tests