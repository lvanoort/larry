# larry
[![Go Reference](https://pkg.go.dev/badge/github.com/lvanoort/larry.svg)](https://pkg.go.dev/github.com/lvanoort/larry)

Larry is a minimal frontend for OpenTelemetry's Log SDK. This is inherently an unstable library because
logs are only in Beta at the time of writing. This is intentionally a very thin wrapper for the Log SDK
that just provides typical logging semantics. It has almost no features because the expectation is that any
significant features would make more sense to implement via the OpenTelemetry Logs SDK instead of 
in this library. The only reason you should use this library (once logs go stable and this library is
updated for those changes) instead of something like slog+slog bridge is if it bothers you on a personal
level to log with slog only for it to be translated into OTel logs instead of using Otel types directly.


## Future Improvements
* Remove the attribute.KeyValue -> log.KeyValue translation. This can be done once 
  https://github.com/open-telemetry/opentelemetry-go/issues/7034 is completed and Record gets native support for
  KeyValue