STATUS: WIP

# About

*timeboxxing* allows you to track your usage history throughout the day and export them as entries to generate invoices and timesheets. All your data is stored locally and you can query it (BYO credits) to see what you spend time on

# Disclaimer

- It's MVP so expect bugs! I've only steered technical decisions and provided functionality requirements, most of the code is written by GPT 5.5 on extra high. The current implementation is barely reviewed and I've only tested it on macOS

# Tech stack

*timeboxxing* runs as a bundled Kotlin application and a sidecar process. The actual tracking is done by the sidecar process. The Kotlin application and sidecar process communicate over GRPC.

We use an embedding model (up to 4096 dimensions) to store embeddings of each usage event and an LLM to allow users to query their usage history. Models can be sourced from either [OpenRouter](https://openrouter.ai/) or a privately hosted [Ollama](https://ollama.com/) server. I've only tested OpenRouter. Secrets for either are stored using the OS's keychain

Sidecar:
- Golang
- SQLite + [sqlite-vector](https://github.com/sqliteai/sqlite-vector)
   - `sqlite-vector` requires `CGO` to be enabled. It was chosen because it implements TurboQuant
   - TurboQuant is required because embeddings take up a lot of space
- [varmq](https://github.com/goptics/varmq)
   - This is a quick hack. SQLite does not allow concurrent writes (even with WAL enabled there are errors sometimes) so we use a queue to process things one at a time while being able to accept concurrently
   - We need to use a queue but the implementation could be better

App:
- [Kotlin Multiplatform](https://kotlinlang.org/multiplatform/)
- [Composables](https://composables.com)

# Alternatives

There are a few closed and open source alternatives such as:
- [memtime](https://www.memtime.com/)
- [ActivityWatch](https://activitywatch.net)

# Demo

https://github.com/user-attachments/assets/c4d5069a-4499-4e6b-8a7f-3733d4de448a



