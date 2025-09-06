# Wheregoes

A simple tool to find out where a URL redirects

### Installation

```shell
go install github.com/jorgejr568/wheregoes@latest
```

### Usage

```shell
wheregoes https://maps.google.com

# Output:
# ❯ wheregoes https://maps.google.com
# URL: https://www.google.com/maps
# Final URL: https://www.google.com/maps
#
# 1 ....... https://maps.google.com (302)
# 2 ....... https://maps.google.com/maps (302)
# 3 ....... https://www.google.com/maps (200)
```

#### Works with local URLs too:

```shell
wheregoes http://localhost:8080

# Output:
# ❯ wheregoes http://localhost:8080
# URL: http://localhost:8080
# Final URL: http://localhost:8080
#
# 1 ....... http://localhost:8080 (200)
```

### Server Mode

You can also run wheregoes as a server:

```shell
wheregoes serve
```

This starts an HTTP server on port 8080 with REST API and WebSocket support for real-time URL tracking.

## Development

### Testing

This project has comprehensive test coverage. See [TESTING.md](TESTING.md) for detailed information about:

- Test structure and coverage reports
- Running different types of tests
- CI/CD pipeline
- Test philosophy and best practices

**Quick test commands:**
```bash
# Run all tests
make test

# Generate coverage report
make coverage

# Run comprehensive test suite
./scripts/test.sh
```

**Current test coverage:** 95.5% on core functionality
