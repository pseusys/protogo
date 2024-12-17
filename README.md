# protogo

> Originally, a part of [`SeasideVPN`](https://github.com/pseusys/SeasideVPN) project.

`protogo` is an automatization tool for Go + protobuf + gRPC builds!

You can run it with the same arguments as `go` executable, followed by `--` flag and then `protoc` arguments.
Just like this:

```shell
protogo version -- --version
```

Both parts are optional and can be skipped, thus can be used as `protoc` installer.
Protoc executable will be placed into `${PROTOGO_CACHE}/protoc-${PROTOGO_PROTOC_VERSION}/bin`, this directory can be added to `$PATH`.


Protogo will handle everything else, including `protoc` binaries installation, installing required packages, etc.
Use [official gRPC installation guide](https://grpc.io/docs/languages/go/quickstart/#prerequisites) as reference.

Inspired by similar projects for other languages, including (but not limited to) [`protoc-exe`](https://pypi.org/project/protoc-exe/) and [`protoc-prebuilt`](https://crates.io/crates/protoc-prebuilt/).

You can additionally control it with the following environment variables:

  - `PROTOGO_GO_EXECUTABLE`: define `go` executable to use, default: `go`
  - `PROTOGO_PROTOC_VERSION`: defing `protoc` version to use, should match protobuf release tags, default: `latest`
      NB! If 'local' is specified as `protoc` version, local installation will be used
  - `PROTOGO_CACHE`: define cache directory, where 'protobuf' executables will be stored, default: `~/.cache/protogo`
  - `PROTOGO_LOG_LEVEL`: define logging level, the levels match [`logrus`](https://github.com/sirupsen/logrus) ones.
