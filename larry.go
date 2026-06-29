package larry

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
)

var timeProvider = time.Now

type Logger struct {
	// lp is stored on the logger so that we can reuse the parent's LoggerProvider when
	// creating a child so that it does not have to be re-supplied if a non-global logger
	// is in-use, which is more intuitive.
	lp log.LoggerProvider

	parentAttrs [][]attribute.KeyValue
	logger      log.Logger

	attrs []attribute.KeyValue
}

type loggerConfig struct {
	lp         log.LoggerProvider
	loggerOpts []log.LoggerOption
	childAttrs []attribute.KeyValue
}

type Option interface {
	apply(l *loggerConfig)
}

type optionFunc func(f *loggerConfig)

func (f optionFunc) apply(l *loggerConfig) {
	f(l)
}

func WithLoggerProvider(p log.LoggerProvider) Option {
	return optionFunc(func(f *loggerConfig) {
		f.lp = p
	})
}

func WithLoggerOpts(opts ...log.LoggerOption) Option {
	return optionFunc(func(f *loggerConfig) {
		f.loggerOpts = append(f.loggerOpts, opts...)
	})
}

func WithAttributes(attrs ...attribute.KeyValue) Option {
	return optionFunc(func(f *loggerConfig) {
		f.childAttrs = append(f.childAttrs, attrs...)
	})
}

func New(name string, opts ...Option) *Logger {
	lcfg := &loggerConfig{
		lp: global.GetLoggerProvider(),
	}
	for _, f := range opts {
		f.apply(lcfg)
	}

	l := &Logger{
		lp: lcfg.lp,

		logger: lcfg.lp.Logger(name, lcfg.loggerOpts...),
		attrs:  lcfg.childAttrs,
	}

	return l
}

func (o *Logger) Trace(ctx context.Context, msg string, attrs ...attribute.KeyValue) {
	o.log(ctx, log.SeverityTrace, msg, attrs...)
}

func (o *Logger) Debug(ctx context.Context, msg string, attrs ...attribute.KeyValue) {
	o.log(ctx, log.SeverityDebug, msg, attrs...)
}

func (o *Logger) Info(ctx context.Context, msg string, attrs ...attribute.KeyValue) {
	o.log(ctx, log.SeverityInfo, msg, attrs...)
}

func (o *Logger) Warn(ctx context.Context, msg string, attrs ...attribute.KeyValue) {
	o.log(ctx, log.SeverityWarn, msg, attrs...)
}

func (o *Logger) Error(ctx context.Context, msg string, attrs ...attribute.KeyValue) {
	o.log(ctx, log.SeverityError, msg, attrs...)
}

func (o *Logger) Fatal(ctx context.Context, msg string, attrs ...attribute.KeyValue) {
	o.log(ctx, log.SeverityFatal, msg, attrs...)
}

// With returns a logger with the same configuration as this logger, but with additional attributes.
// It reuses the logger from the original logger. This should not be confused with Child(), which should be
// used to create a _new_ logger that inherits the parent's attributes.
func (o *Logger) With(attrs ...attribute.KeyValue) *Logger {
	parentAttrs := make([][]attribute.KeyValue, 0, len(o.parentAttrs)+1)
	parentAttrs = append(parentAttrs, o.parentAttrs...)
	if len(o.attrs) > 0 {
		parentAttrs = append(parentAttrs, o.attrs)
	}
	return &Logger{
		lp:          o.lp,
		parentAttrs: parentAttrs,
		logger:      o.logger,
		attrs:       attrs,
	}
}

// Child creates a new logger that will share the attributes from the parent logger.
func (o *Logger) Child(name string, opts ...Option) *Logger {
	lcfg := &loggerConfig{
		lp: o.lp,
	}
	for _, f := range opts {
		f.apply(lcfg)
	}

	parentAttrs := make([][]attribute.KeyValue, 0, len(o.parentAttrs)+1)
	parentAttrs = append(parentAttrs, o.parentAttrs...)
	if len(o.attrs) > 0 {
		parentAttrs = append(parentAttrs, o.attrs)
	}

	l := &Logger{
		lp:          lcfg.lp,
		parentAttrs: parentAttrs,
		logger:      lcfg.lp.Logger(name, lcfg.loggerOpts...),
		attrs:       lcfg.childAttrs,
	}

	return l
}

func (o *Logger) log(ctx context.Context, level log.Severity, msg string, attrs ...attribute.KeyValue) {
	if !o.logger.Enabled(ctx, log.EnabledParameters{
		Severity: level,
	}) {
		return
	}

	rec := &log.Record{}
	rec.SetSeverity(level)
	rec.SetBody(log.StringValue(msg))
	rec.SetTimestamp(timeProvider())
	addAttrsToRecordAdaptive(rec, o.attrs, attrs, o.parentAttrs...)
	o.logger.Emit(ctx, *rec)
}

// addAttrsToRecordAdaptive converts the attrs to log.KeyValue and adds them to the record. In theory, it should
// not be needed once native support for attribute.KeyValue lands in the SDK. It switches between two mechanisms
// depending on how many attributes need to be added, this is a perf optimization that takes advantage of our
// knowledge of an internal perf optimization in the OTel logs SDK that works for small attribute counts.
// It may be worth keeping the logic here that switches methods of setting attributes even after we have native
// attribute support as a perf optimization, but that should be validated at that time.
// As a further optimization, we could probably also make a non-variadic form of this function for cases where
// the Logger is not carrying any attributes of its own.
//
// see: https://github.com/open-telemetry/opentelemetry-go/issues/7034 for when native attribute support will land
func addAttrsToRecordAdaptive(rec *log.Record, one, two []attribute.KeyValue, rest ...[]attribute.KeyValue) {
	fullLen := len(one) + len(two)
	for _, attrList := range rest {
		fullLen += len(attrList)
	}

	if fullLen > 5 {
		addAttrsToRecordSlice(rec, fullLen, one, two, rest...)
	} else {
		addAttrsToRecordDirect(rec, one, two, rest...)
	}
}

func addAttrsToRecordDirect(rec *log.Record, one, two []attribute.KeyValue, rest ...[]attribute.KeyValue) {
	for _, a := range one {
		rec.AddAttributes(log.KeyValue{
			Key:   string(a.Key),
			Value: log.ValueFromAttribute(a.Value),
		})
	}
	for _, a := range two {
		rec.AddAttributes(log.KeyValue{
			Key:   string(a.Key),
			Value: log.ValueFromAttribute(a.Value),
		})
	}
	for _, attrs := range rest {
		for _, a := range attrs {
			rec.AddAttributes(log.KeyValue{
				Key:   string(a.Key),
				Value: log.ValueFromAttribute(a.Value),
			})
		}
	}
}

func addAttrsToRecordSlice(rec *log.Record, fullLen int, one, two []attribute.KeyValue, rest ...[]attribute.KeyValue) {
	converted := make([]log.KeyValue, fullLen)

	idx := 0
	for _, a := range one {
		converted[idx].Key = string(a.Key)
		converted[idx].Value = log.ValueFromAttribute(a.Value)
		idx++
	}
	for _, a := range two {
		converted[idx].Key = string(a.Key)
		converted[idx].Value = log.ValueFromAttribute(a.Value)
		idx++
	}
	for _, attrList := range rest {
		for _, a := range attrList {
			converted[idx].Key = string(a.Key)
			converted[idx].Value = log.ValueFromAttribute(a.Value)
			idx++
		}
	}
	rec.AddAttributes(converted...)
}
