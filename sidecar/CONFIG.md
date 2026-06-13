# Environment Variables

This document describes the environment variables used by `sidecar`.

| Name                            | Usage                       | Description                                       |
| ------------------------------- | --------------------------- | ------------------------------------------------- |
| [`SIDECAR_DATABASE_DSN`]        | defaults to `test.db`       | the database data source name                     |
| [`SIDECAR_DATABASE_ENGINE`]     | defaults to `sqlite`        | the database engine used by the sidecar           |
| [`SIDECAR_GRPC_LISTEN_ADDRESS`] | defaults to `0.0.0.0:50051` | the host and port that the gRPC server listens on |

> [!TIP]
> If an environment variable is set to an empty value, `sidecar` behaves as if
> that variable is left undefined.

## `SIDECAR_DATABASE_DSN`

> the database data source name

The `SIDECAR_DATABASE_DSN` variable **MAY** be left undefined, in which case the
default value of `test.db` is used.

```bash
export SIDECAR_DATABASE_DSN=test.db # (default)
```

## `SIDECAR_DATABASE_ENGINE`

> the database engine used by the sidecar

The `SIDECAR_DATABASE_ENGINE` variable **MAY** be left undefined, in which case
the default value of `sqlite` is used. Otherwise, the value must be sqlite.

```bash
export SIDECAR_DATABASE_ENGINE=sqlite # (default)
```

## `SIDECAR_GRPC_LISTEN_ADDRESS`

> the host and port that the gRPC server listens on

The `SIDECAR_GRPC_LISTEN_ADDRESS` variable **MAY** be left undefined, in which
case the default value of `0.0.0.0:50051` is used. Otherwise, the value **MUST**
be a valid network address.

```bash
export SIDECAR_GRPC_LISTEN_ADDRESS=0.0.0.0:50051          # (default)
export SIDECAR_GRPC_LISTEN_ADDRESS=192.168.0.1:8080       # (non-normative) an IPv4 address with a port
export SIDECAR_GRPC_LISTEN_ADDRESS='[::1]:8080'           # (non-normative) an IPv6 address with a port
export SIDECAR_GRPC_LISTEN_ADDRESS=host.example.org:https # (non-normative) a named host with an IANA service name
```

<details>
<summary>Network address syntax</summary>

Addresses may be specified as `<host>:<port>`, where `<host>` is a hostname or
IP address and `<port>` is a numeric port number or an IANA service name. IPv6
addresses must be enclosed in square brackets, e.g. `[::1]:8080`.

</details>

---

> [!NOTE]
> This document only describes environment variables declared using [Ferrite].
> `sidecar` may consume other undocumented environment variables.

> [!IMPORTANT]
> Some of the example values given in this document are **non-normative**.
> Although these values are syntactically valid, they may not be meaningful to
> `sidecar`.

<!-- references -->

[ferrite]: https://github.com/dogmatiq/ferrite
[`sidecar_database_dsn`]: #sidecar_database_dsn
[`sidecar_database_engine`]: #sidecar_database_engine
[`sidecar_grpc_listen_address`]: #sidecar_grpc_listen_address
