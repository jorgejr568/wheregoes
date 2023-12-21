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
