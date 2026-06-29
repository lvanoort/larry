# larry
[![Go Reference](https://pkg.go.dev/badge/github.com/lvanoort/larry.svg)](https://pkg.go.dev/github.com/lvanoort/larry)

Larry is a minimal frontend for OpenTelemetry's Log SDK. This is inherently an unstable library because
logs are only in Beta at the time of writing. This is intentionally a very thin wrapper for the Log SDK
that just provides typical logging semantics. It has almost no features because the expectation is that any
significant features would make more sense to implement via the OpenTelemetry Logs SDK instead of 
in this library. The only reason you should use this library (once logs go stable and this library is
updated for those changes) instead of something like slog+slog bridge is if it bothers you on a personal
level to log with slog only for it to be translated into OTel logs instead of using Otel types directly.

## FAQ

### Why isn't there a Fatal()?
Typically in Go, calling `Fatal` causes the process to exit via calling `os.Exit(1)`, but doing this bypasses
`defer`red functions, meaning that if one is using a non-synchronous `LoggerProvider`, the log message may
not end up being flushed to the logging backend since, typically, the `LoggerProvider` is flushed in a `defer`red
function. As a result, implementing this in the typical way might result in crashing the program without ever
logging why it crashed, which defeats the point of having a Fatal log level. This leaves us with 3 options
1) Try to flush the LoggerProvider ourselves. We could check if the LoggerProvider has a `ForceFlush` method
   and use it if present, but given that this materially changes how Larry operates in a potentially unexpected
   fashion, we don't do it. If `ForceFlush` was part of the `LoggerProvider` interface, then we would probably
   take this option.
2) Don't crash the program. This is what the first version of this library did, but this is very counter-intuitive
   for Go veterans since it is generally expected that `Fatal` will crash the program, so this approach was abandoned.
3) Throw a panic. A panic doesn't prevent `defer`red functions from running and will (unless `recover`ed) crash the 
   program. This would still behave differently than typical Go `Fatal` calls do since `os.Exit` which cannot be
   `recover`ed from is the typical choice.

As a result, the only options are to do something unexpected or to not do anything at all. Loggers should not be a
source of unexpected behaviour, so we are left with doing nothing at all.


## Future Improvements
* Remove the attribute.KeyValue -> log.KeyValue translation. This can be done once 
  https://github.com/open-telemetry/opentelemetry-go/issues/7034 is completed and Record gets native support for
  KeyValue