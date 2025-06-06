# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go test project that validates FTP server functionality using Docker containers. The project uses `dockertest` to spin up a vsftpd FTP server, mounts test data, and verifies file listings.

## Commands

### Testing
```bash
go test -v        # Run all tests with verbose output
go test -run TestFTPServer  # Run specific test (once implemented)
```

### Dependencies
```bash
go mod download   # Download dependencies
go mod tidy       # Clean up and verify dependencies
```

## Architecture

The project follows a test-driven approach with these key components:

- **Docker Integration**: Uses `github.com/ory/dockertest/v3` to programmatically manage Docker containers
- **FTP Server**: Runs `wildscamp/vsftpd` Docker image as the FTP server for testing
- **Test Data**: The `testdata/` directory is mounted into the Docker container and contains test files that should be accessible via FTP
- **Verification**: Tests connect to the FTP server and verify that `file1.txt` and `file2.txt` can be listed

## Key Implementation Notes

- The main test file should be `main_test.go` (not yet created)
- FTP server credentials and connection parameters should be configured when setting up the Docker container
- The test should clean up Docker resources after completion using `defer` statements