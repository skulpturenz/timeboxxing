STATUS: WIP

# Overview

We need a way to dynamically register workers and GRPC services since the current implementation only allows including all features at build time via static registration. The value in being able to dynamically load MCP tools and workers at runtime is that it allows third parties to build on top and us the option to not have to share everything. Unlike JVM languages, Golang does not have a cross-platform way of dynamically linking libraries (package [plugin](https://pkg.go.dev/plugin) does provide a less than ideal solution on a few UNIX platforms).

From the documentation, the main drawbacks of the package are:
- Poor portability. In our case, only macOS is supported
- Higher risk of runtime crashes as the main program and the plugin must be built in the same environment
- Poorly supported by the Go race detector

In a plugin system, we're looking for:
- Cross-platform: macOS, Windows, Linux
- Crash isolation: plugins should not be able to crash the sidecar process
- Versioned API: important when opening up to third parties
- Accessible: a low barrier to entry is important as it allows users to build for their use cases
- Security: mainly the ability to whitelist
- Single writer: we need to be able to enforce this so that plugins can't make the sidecar unstable
   - SQLite DB is currently completely accessible. [SQLCipher](https://github.com/sqlcipher/sqlcipher) allows us to encrypt the database, have not looked into it much yet about how we might make it work with Go

The two main options for Golang are:
- [hashicorp/go-plugin](https://pkg.go.dev/github.com/hashicorp/go-plugin)
   - Positives:
      - Well supported
      - Battle tested
      - Strong community
      - Host functions
      - Plugin stdout/stderr to host
      - Host upgrade while plugin is running
   - Negatives:
      - No sandbox: plugin can access anything
      - Process per plugin
- [knqyf263/go-plugin](https://pkg.go.dev/github.com/knqyf263/go-plugin/wasm)
   - Positives:
      - Host functions
      - Plugin stdout/stderr to host
      - Sandboxed: plugin can't just access anything
      - Runs within the sidecar process
      - Poor docs
   - Negatives:
      - Must be written in Go
      - Pending TODOs
      - Network access must be routed through sidecar. Not exactly a bad thing but also a pain to work with.

For our purposes, I think `hashicorp/go-plugin` is a good choice. 

# Approach

- NEED A POC

- Discover plugins through registry. Worker plugins and MCP plugins (TODO)

- Standard interfaces for workers and tools
   - Workers: `Exec` with only the transition event, `ReadQuerier` and `WriteQuerier` (embed?)
   - MCP tools: map? want to leave it relatively open

- Plugins added later overloading the system? If we're polling fast enough to be snappy there's gonna be a lot of events
