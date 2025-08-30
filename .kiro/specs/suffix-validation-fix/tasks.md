# Implementation Plan

- [x] 1. Create comprehensive test cases for suffix validation
  - Write unit tests for `isValidBlocoAddress` function covering all suffix scenarios
  - Create test cases for case-insensitive suffix validation (should already pass)
  - Create test cases for EIP-55 checksum suffix validation (should currently fail)
  - Add test cases for combined prefix+suffix validation with and without checksum
  - _Requirements: 1.1, 2.1, 4.1_

- [x] 2. Run existing tests to establish baseline
  - Execute `make test` to ensure all current tests pass before making changes
  - Document any existing test failures for reference
  - Run benchmark tests to establish performance baseline
  - _Requirements: 4.1, 3.1_

- [x] 3. Fix the suffix validation bug in isEIP55Checksum function
  - Locate the bug in `internal/worker/pool.go` in the `isEIP55Checksum` function
  - Remove the incorrect case comparison `if suffixPart != suffix` that always fails
  - Implement correct EIP-55 suffix validation logic that checks pattern match without requiring exact case match
  - Ensure the fix maintains the existing checksum validation integrity
  - _Requirements: 1.1, 1.2, 1.3_

- [x] 4. Add debug logging for validation process
  - Add optional debug logging using `BLOCO_DEBUG` environment variable
  - Log suffix validation steps when debug mode is enabled
  - Include information about pattern matching and checksum validation results
  - Ensure debug logging doesn't impact performance when disabled
  - _Requirements: 2.2, 2.3_

- [x] 5. Verify the fix with comprehensive testing
  - Run the new unit tests to ensure suffix validation works correctly
  - Test CLI commands with various prefix/suffix combinations
  - Verify that `./bloco-eth --prefix abc --suffix def --count 1` generates correct addresses
  - Test checksum mode: `./bloco-eth --prefix abc --suffix def --checksum --count 1`
  - _Requirements: 1.1, 1.2, 1.3, 1.4_

- [x] 6. Run performance benchmarks to ensure no regression
  - Execute `make bench` to compare performance before and after the fix
  - Verify that the fix doesn't introduce performance degradation
  - Ensure validation complexity remains O(1)
  - Document any performance changes
  - _Requirements: 3.1, 3.2, 3.3_

- [x] 7. Validate all existing functionality still works
  - Run complete test suite with `make test` to ensure no regressions
  - Test prefix-only validation to ensure it still works correctly
  - Test suffix-only validation to ensure the fix resolves the issue
  - Verify that addresses without prefix/suffix requirements still generate correctly
  - _Requirements: 4.1, 4.2, 4.3, 4.4_