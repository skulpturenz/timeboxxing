STATUS: WIP

# Overview

We want to prevent unauthorized access to the SQLite database when we do officially support plugins. This is primarily to ensure the stability of the system and it also ensures personal user data is stored securely. Secure storing is not the main motivator and if it weren't for plugins the preference would have been to leave it unencrypted because of the complexity required to implement it. Many browsers also do not encrypt browser history. The reason for including it in v1 is because this would likely be a breaking change.

The purpose of any plugin system we add would be to add to the "smarts" of the system via MCP tools. Our case is simple, there is only one book to study, the foreground process event store, and once all those events are processed, any insights can be provided as a tool. This leads us to only providing the plugin system access to the event store: plugins should be responsible for storing their data and keeping track of what has been processed. Since plugins should not crash the sidecar, any exceptions can just be bubbled up. Encryption and denying access to tables other than the foreground event store also allows us to sidestep having to enforce table level restrictions (don't think this is supported by SQLite at all) using authentication while also minimizing the services the sidecar needs to provide plugins 

# Design goals

- Allow plugins to process foreground events while ensuring the stability of the system

- Store application usage history securely

# Approach

- NEED A POC

- There are a few implementations of [SQLCipher](https://github.com/sqlcipher/sqlcipher) in Golang but none seem to have gained enough traction for LTS. Fork?
   - https://github.com/mattn/go-sqlite3/issues/1337
   - https://github.com/mattn/go-sqlite3/pull/1109 (2022 ;-;)

- https://sqlite.org/com/see.html is official but it is US$2000

- Encryption key: randomly generated and store it in OS keychain. Kotlin shows the key for users to write down if they want to
