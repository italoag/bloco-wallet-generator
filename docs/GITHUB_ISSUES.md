# GitHub Issues for Multi-Chain Wallet Generator

This document outlines the GitHub issues that should be created to track the remaining work for multi-chain support.

## Issue Template

Each issue should include:
- Clear title
- Description of the work
- Acceptance criteria
- Testing requirements
- Related files/components

---

## Issue 1: Add CLI Chain Selection Support

**Title**: Add `--chain` flag for network selection

**Labels**: enhancement, cli, multi-chain

**Description**:
Add command-line flag to select which blockchain network to generate wallets for.

**Tasks**:
- [ ] Add `--chain` flag to root command (accepts: eth, btc, sol)
- [ ] Update CLI help text to document chain selection
- [ ] Pass chain selection to Config
- [ ] Update error messages to be chain-aware
- [ ] Add validation for chain parameter

**Acceptance Criteria**:
- User can run `./bloco-eth --chain eth --prefix abc` for Ethereum
- User can run `./bloco-eth --chain btc --prefix 1ab` for Bitcoin
- User can run `./bloco-eth --chain sol --prefix Ab` for Solana
- Invalid chain name shows helpful error message
- Help text clearly explains chain options

**Files to Modify**:
- `internal/cli/commands.go`
- `internal/cli/logging_test.go` (if needed)

**Testing**:
- Unit tests for flag parsing
- Integration tests for each chain selection
- Error case testing for invalid chains

**Related**:
- Depends on Bitcoin adapter (Issue #2)
- Depends on Solana adapter (Issue #3)

---

## Issue 2: Implement Bitcoin Wallet Generation

**Title**: Implement Bitcoin adapter for multi-chain support

**Labels**: enhancement, bitcoin, crypto, multi-chain

**Description**:
Implement ChainAdapter for Bitcoin wallet generation with BIP-39/BIP-32/BIP-44 support and Base58Check encoding.

**Tasks**:
- [ ] Create `internal/crypto/bitcoin.go` with `BitcoinAdapter` struct
- [ ] Implement key generation (secp256k1)
- [ ] Implement Base58Check address encoding
- [ ] Support P2PKH format (addresses starting with '1')
- [ ] Implement BIP-44 derivation path (m/44'/0'/0'/0/0)
- [ ] Generate WIF (Wallet Import Format) private keys
- [ ] Support mainnet/testnet version bytes
- [ ] Implement pattern validation for Base58
- [ ] Update difficulty calculations for Base58 character set
- [ ] Create comprehensive test suite

**Acceptance Criteria**:
- Bitcoin addresses are valid P2PKH format (start with '1')
- Addresses pass Base58Check validation
- Pattern matching works with Base58 characters
- WIF private keys can be imported into Bitcoin wallets
- Mainnet and testnet both supported
- BIP-44 derivation path correct
- All tests pass with >90% coverage

**Files to Create**:
- `internal/crypto/bitcoin.go`
- `internal/crypto/bitcoin_test.go`

**Files to Modify**:
- `internal/crypto/factory.go` (update GetAdapter)
- Update `README.md` with Bitcoin examples

**Technical Notes**:
- Base58 alphabet: `123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz`
- Difficulty is higher than hex: 58^n vs 16^n
- May need optimized Base58 library (consider `github.com/btcsuite/btcutil/base58`)

**Testing Requirements**:
- Unit tests for address generation
- Unit tests for Base58 encoding/decoding
- Unit tests for checksum validation
- Unit tests for pattern matching
- Integration tests with worker pool
- Test with known test vectors from Bitcoin specification
- Performance benchmarks comparing to Ethereum

**Dependencies**:
Recommended libraries:
- `github.com/btcsuite/btcutil` - Bitcoin utilities including Base58
- `github.com/tyler-smith/go-bip32` - Already in project for BIP-32
- `github.com/tyler-smith/go-bip39` - Already in project for BIP-39

---

## Issue 3: Implement Solana Wallet Generation

**Title**: Implement Solana adapter for multi-chain support

**Labels**: enhancement, solana, crypto, multi-chain

**Description**:
Implement ChainAdapter for Solana wallet generation using ed25519 curve and Base58 encoding.

**Tasks**:
- [ ] Create `internal/crypto/solana.go` with `SolanaAdapter` struct
- [ ] Implement ed25519 key generation
- [ ] Implement Base58 address encoding (public key as address)
- [ ] Implement BIP-44 derivation for ed25519 (m/44'/501'/0'/0')
- [ ] Support case-sensitive pattern matching
- [ ] Update pattern validation for Base58
- [ ] Create comprehensive test suite
- [ ] Document Solana-specific considerations

**Acceptance Criteria**:
- Solana addresses are valid Base58 format (32-byte public key)
- ed25519 keys generated correctly
- Pattern matching is case-sensitive
- BIP-44 path correct for Solana (coin type 501)
- Addresses compatible with Solana CLI/wallets
- All tests pass with >90% coverage

**Files to Create**:
- `internal/crypto/solana.go`
- `internal/crypto/solana_test.go`

**Files to Modify**:
- `internal/crypto/factory.go` (update GetAdapter)
- Update `README.md` with Solana examples

**Technical Notes**:
- Solana uses ed25519, NOT secp256k1
- Address = Base58(public_key) - no hashing
- 32 bytes → typically 44 Base58 characters
- Case-sensitive pattern matching required
- BIP-44 derivation for ed25519 may require special library

**Testing Requirements**:
- Unit tests for ed25519 key generation
- Unit tests for Base58 address formatting
- Unit tests for case-sensitive pattern matching
- Unit tests for BIP-44 derivation
- Integration tests with worker pool
- Test with known Solana test vectors
- Performance benchmarks

**Dependencies**:
Recommended libraries:
- `crypto/ed25519` - Standard library (for key generation)
- `github.com/mr-tron/base58` - Fast Base58 encoding
- May need ed25519 BIP-44 derivation library (research needed)

**Research Needed**:
- Best library for ed25519 BIP-44 derivation
- Solana-specific derivation path requirements
- Compatibility with Solana wallet formats

---

## Issue 4: Add Mnemonic Support to Adapters

**Title**: Implement mnemonic phrase generation in ChainAdapter pattern

**Labels**: enhancement, crypto, security, multi-chain

**Description**:
Add optional mnemonic phrase generation to all chain adapters following BIP-39/BIP-44 standards.

**Tasks**:
- [ ] Add mnemonic generation method to ChainAdapter interface
- [ ] Implement for EthereumAdapter (m/44'/60'/0'/0/0)
- [ ] Implement for BitcoinAdapter (m/44'/0'/0'/0/0)
- [ ] Implement for SolanaAdapter (m/44'/501'/0'/0')
- [ ] Update worker pool to support UseMnemonic flag
- [ ] Store mnemonic securely in Wallet struct
- [ ] Add comprehensive tests for each chain
- [ ] Document mnemonic security best practices

**Acceptance Criteria**:
- Each adapter can generate from mnemonic
- Derivation paths correct per BIP-44
- Mnemonics are BIP-39 compatible
- Same mnemonic produces consistent addresses
- Mnemonic securely stored/handled
- Tests verify deterministic generation
- CLI supports mnemonic generation flag

**Files to Modify**:
- `internal/crypto/adapter.go` (interface update)
- `internal/crypto/ethereum.go`
- `internal/crypto/bitcoin.go`
- `internal/crypto/solana.go`
- `internal/worker/pool.go`
- `pkg/wallet/types.go`

**Security Considerations**:
- Never log mnemonics
- Secure memory handling
- Clear warning about mnemonic backup
- Document recovery process

**Testing Requirements**:
- Test deterministic generation for all chains
- Test with known BIP-39 test vectors
- Test cross-chain mnemonic compatibility
- Security tests (no logging of mnemonics)

**Dependencies**:
- Already have: `github.com/tyler-smith/go-bip39`
- Already have: `github.com/tyler-smith/go-bip32`
- May need: ed25519 derivation library for Solana

---

## Issue 5: Update Documentation for Multi-Chain Support

**Title**: Update documentation to reflect multi-chain capabilities

**Labels**: documentation, multi-chain

**Description**:
Update all documentation to explain multi-chain support and provide examples for each supported blockchain.

**Tasks**:
- [ ] Update README.md with multi-chain overview
- [ ] Add section explaining each supported chain
- [ ] Provide examples for Ethereum, Bitcoin, and Solana
- [ ] Document `--chain` flag usage
- [ ] Explain encoding differences (hex vs Base58)
- [ ] Update performance notes for different chains
- [ ] Add troubleshooting section for each chain
- [ ] Update security considerations
- [ ] Add architecture documentation for ChainAdapter pattern

**Acceptance Criteria**:
- README clearly explains multi-chain support
- Each chain has usage examples
- Performance implications documented
- Security best practices updated
- Architecture clearly explained
- All examples tested and working

**Files to Modify**:
- `README.md`
- `docs/ARCHITECTURE.md` (if exists, or create)
- `docs/SECURITY.md` (if exists, or create)
- `docs/MULTI_CHAIN_IMPLEMENTATION.md` (already created)

**Content to Include**:
```bash
# Ethereum example
./bloco-eth --chain eth --prefix abc --suffix 123

# Bitcoin example
./bloco-eth --chain btc --prefix 1ab

# Solana example (case-sensitive)
./bloco-eth --chain sol --prefix Ab
```

**Testing**:
- Verify all examples in documentation work
- Test on clean installation
- Get feedback from users

---

## Issue 6: Performance Benchmarks for Multi-Chain Generation

**Title**: Create performance benchmarks for all supported chains

**Labels**: performance, testing, multi-chain

**Description**:
Create comprehensive benchmarks to measure and compare wallet generation performance across different blockchains.

**Tasks**:
- [ ] Benchmark key generation speed per chain
- [ ] Benchmark pattern matching speed per encoding
- [ ] Compare difficulty calculations (hex vs Base58)
- [ ] Benchmark with different pattern lengths
- [ ] Test with different thread counts
- [ ] Create performance comparison documentation
- [ ] Identify optimization opportunities

**Acceptance Criteria**:
- Benchmarks for each chain (Ethereum, Bitcoin, Solana)
- Results documented with charts/tables
- Performance characteristics explained
- Optimization recommendations provided
- CI/CD integration for regression detection

**Files to Create**:
- `internal/crypto/bench_test.go` (or per-chain files)
- `docs/PERFORMANCE.md`

**Metrics to Measure**:
- Keys/second per chain
- Pattern matches/second
- Memory usage
- CPU utilization
- Scalability (speedup vs threads)

**Testing**:
```bash
go test -bench=. -benchmem ./internal/crypto
```

---

## Issue 7: CLI Integration Tests for Multi-Chain

**Title**: Add end-to-end CLI tests for all supported chains

**Labels**: testing, cli, multi-chain

**Description**:
Create comprehensive integration tests that exercise the CLI with all supported chains.

**Tasks**:
- [ ] Test Ethereum wallet generation via CLI
- [ ] Test Bitcoin wallet generation via CLI
- [ ] Test Solana wallet generation via CLI
- [ ] Test chain selection validation
- [ ] Test pattern validation per chain
- [ ] Test error messages
- [ ] Test with various flag combinations
- [ ] Automate in CI/CD

**Acceptance Criteria**:
- All chains tested via CLI
- Invalid inputs properly rejected
- Error messages chain-specific
- Success cases verified
- Tests run in CI/CD
- Tests pass on Linux, macOS, Windows

**Testing Approach**:
Use Go's testing framework to:
1. Build the binary
2. Execute with various arguments
3. Verify output and exit codes
4. Check generated wallet files

**Example Test**:
```go
func TestCLI_EthereumGeneration(t *testing.T) {
    cmd := exec.Command("./bloco-eth", "--chain", "eth", "--prefix", "abc")
    output, err := cmd.CombinedOutput()
    // verify output...
}
```

---

## Issue 8: Migration Guide for Existing Users

**Title**: Create migration guide for users upgrading from Ethereum-only version

**Labels**: documentation, migration

**Description**:
Write a comprehensive guide to help existing users understand changes and migrate to the multi-chain version.

**Tasks**:
- [ ] Document what's changed
- [ ] Explain backward compatibility
- [ ] Provide upgrade instructions
- [ ] Address common questions
- [ ] Include examples of old vs new commands
- [ ] Explain new configuration options

**Acceptance Criteria**:
- Clear explanation of changes
- Backward compatibility confirmed
- Migration steps documented
- FAQs included
- Examples for common scenarios

**File to Create**:
- `docs/MIGRATION.md`

**Content Sections**:
1. What's New
2. Breaking Changes (if any)
3. Backward Compatibility
4. Migration Steps
5. Updated Configuration
6. FAQs
7. Troubleshooting

---

## Implementation Order Recommendation

Suggested order for tackling these issues:

1. **Issue #1** (CLI Chain Selection) - Foundation for user interaction
2. **Issue #2** (Bitcoin Adapter) - First new chain, establishes pattern
3. **Issue #5** (Documentation) - Document as we go
4. **Issue #3** (Solana Adapter) - Second new chain, different curve
5. **Issue #4** (Mnemonic Support) - Enhancement to all chains
6. **Issue #6** (Performance Benchmarks) - Measure and optimize
7. **Issue #7** (CLI Integration Tests) - Comprehensive testing
8. **Issue #8** (Migration Guide) - Final polish for users

---

## Success Criteria for Project Completion

The multi-chain wallet generator project will be considered complete when:

- ✅ All 8 issues closed
- ✅ All tests passing (unit, integration, CLI)
- ✅ Documentation complete and accurate
- ✅ Performance benchmarks documented
- ✅ CI/CD pipeline green
- ✅ Code review approved
- ✅ Security review passed
- ✅ User feedback incorporated

---

## Notes for Issue Creation in GitHub

When creating these issues in GitHub:

1. Use the labels suggested in each issue
2. Assign appropriate milestone (e.g., "Multi-Chain v2.0")
3. Set priority based on implementation order
4. Link related issues with "Depends on #X" or "Blocks #X"
5. Add to project board if using GitHub Projects
6. Consider creating a meta-issue linking all others

Example meta-issue:
**Title**: Multi-Chain Wallet Generator - Master Tracking Issue
**Body**: Links to all 8 issues with checkboxes for tracking overall progress
