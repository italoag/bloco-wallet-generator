---
inclusion: always
---

# Testing Guidelines

## Core Testing Principles
- **Test-First Approach**: Always run existing tests before making changes to ensure baseline functionality
- **Incremental Testing**: Test after each implementation step to catch issues early
- **Regression Prevention**: Verify that new changes don't break existing functionality
- **Cross-Platform Validation**: Test on different platforms when possible

## Testing Workflow for Each Implementation Task
1. **Before Implementation**: Run `make test` to ensure all existing tests pass
2. **During Implementation**: Add unit tests for new functionality as you implement
3. **After Implementation**: Run full test suite including new tests
4. **Integration Testing**: Test the actual CLI commands manually to verify behavior
5. **Performance Validation**: Run benchmarks if performance-critical code is modified

## Required Test Commands

```bash
# Basic test suite
make test

# Test with race detection
make test-race

# Run benchmarks
make bench

# Build and test the binary
make build
./bloco-eth --help

# Test specific functionality
./bloco-eth --prefix abc --suffix a --count 1 --progress
./bloco-eth benchmark --attempts 1000
./bloco-eth stats --prefix abc
```

## Test Coverage Requirements
- All new functions must have corresponding unit tests
- Integration points (like Fang integration) must have integration tests
- Error handling paths must be tested
- Backward compatibility must be verified through existing test suite

## Failure Handling
- If any test fails during implementation, stop and fix the issue before proceeding
- Document any test modifications needed due to new functionality
- Ensure test failures are meaningful and help identify the root cause

## Performance Testing
- Run benchmarks before and after changes to detect performance regressions
- For CLI enhancements, measure startup time and memory usage
- Verify that new dependencies don't significantly impact performance
- We are working on Mac OS, command `timeout` don't exists never try to use it!