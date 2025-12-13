# Changelog

## [0.5.0](https://github.com/goatx/goat/releases/tag/v0.5.0) - 2025-12-13

### Added
- OpenAPI schema generation support
  - New `openapi` package for generating OpenAPI 3.0 specifications from Go state machines
  - Support for request/response schema generation with `AbstractSchema` interface
  - HTTP method mapping (GET, POST, PUT, DELETE, PATCH) with status code configuration
  - Request/response body and path parameter type definitions
  - Automatic schema reference generation and validation
- E2E test generation for Protocol Buffer services
  - New `GenerateE2ETestSuite` function to generate complete test suites
  - Automatic generation of `main_test.go` with test infrastructure
  - Service-specific test file generation with method test cases
  - Support for custom test input scenarios via `MethodTestCase`
- New internal utility packages

### Changed
- Refactored API naming for OpenAPI and Protocol Buffer packages
- Updated test golden files to use `.golden` extension

### Dependencies
- Updated `actions/checkout` to v6
- Updated `golangci/golangci-lint-action` to v9
- Updated various GitHub Actions workflows

### Fixed
- Disabled repository checkout in govulncheck action to improve CI performance

## [0.4.0](https://github.com/goatx/goat/releases/tag/v0.4.0) - 2025-11-07

### Changed
- Improved model check output for better readability
- **Breaking:** Removed event instance parameters from OnEvent APIs

## [0.3.0](https://github.com/goatx/goat/releases/tag/v0.3.0) - 2025-10-30

### Added
- Temporal rule support
- Implemented sender and recipient in event model

### Changed
- Unified rule registration via WithRules helpers
- Replaced invariants with conditions

### Fixed
- Fixed initialWorld clearing each state machine’s handler builders

## [0.2.0](https://github.com/goatx/goat/releases/tag/v0.2.0) - 2025-09-14

### Added
- Protocol Buffer generation support
  - Support for generating `.proto` files from Go state machines

### Dependencies
- Updated GitHub Actions workflows (setup-go v6, checkout v5)
- Migrated Renovate configuration

## [0.1.1](https://github.com/goatx/goat/releases/tag/v0.1.1) - 2025-08-16

### Added
- Multi state machine invariants support
  - `NewMultiInvariant` function to create invariants that reference multiple state machines
  - `NewInvariant2` and `NewInvariant3` convenience functions for 2 or 3 state machines

## [0.1.0](https://github.com/goatx/goat/releases/tag/v0.1.0) - 2025-07-31

- Initial release of goat
