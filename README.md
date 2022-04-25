# sof-packager

## Installation

1.  Install [go](https://go.dev/) 1.18 or later
2.  `go install github.com/afq984/sof-packager/cmd/...@latest`
3.  The `sof-packager` executable is installed to `$(go env GOPATH)/bin` (which defaults to `~/go/bin`)

## Use

1.  Write a build config. See `examples/` directory for examples.
2.  `sof-packager <path-to-config>`.

    For example: `~/go/bin/sof-packager examples/mt8195-main.textproto`

3.  `sof-packager` outputs:
    *   a tarball containing the built artifacts
    *   a config textproto with pinned versions to reproduce the build
    *   a ebuild manifest file

## Dev

unordered commands of frequent use:

```
go build -o bin/ ./cmd/
tools/gen-pb.bash
```
