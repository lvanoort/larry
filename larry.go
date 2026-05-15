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
	pro log.Logger

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
		pro:   lcfg.lp.Logger(name, lcfg.loggerOpts...),
		attrs: lcfg.childAttrs,
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

func (o *Logger) log(ctx context.Context, level log.Severity, msg string, attrs ...attribute.KeyValue) {
	if !o.pro.Enabled(ctx, log.EnabledParameters{
		Severity: level,
	}) {
		return
	}

	rec := &log.Record{}
	rec.SetSeverity(level)
	rec.SetBody(log.StringValue(msg))
	rec.SetTimestamp(timeProvider())
	addAttrsToRecordAdaptive(rec, o.attrs, attrs)
	o.pro.Emit(ctx, *rec)
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
func addAttrsToRecordAdaptive(rec *log.Record, attrs ...[]attribute.KeyValue) {
	fullLen := 0
	for _, attrList := range attrs {
		fullLen += len(attrList)
	}

	if fullLen > 5 {
		addAttrsToRecordSlice(rec, fullLen, attrs...)
	} else {
		addAttrsToRecordDirect(rec, attrs...)
	}
}

func addAttrsToRecordDirect(rec *log.Record, attrLists ...[]attribute.KeyValue) {
	for _, attrs := range attrLists {
		for _, a := range attrs {
			rec.AddAttributes(log.KeyValue{
				Key:   string(a.Key),
				Value: log.ValueFromAttribute(a.Value),
			})
		}
	}
}

func addAttrsToRecordSlice(rec *log.Record, fullLen int, attrLists ...[]attribute.KeyValue) {
	converted := make([]log.KeyValue, fullLen)

	idx := 0
	for _, attrList := range attrLists {
		for _, a := range attrList {
			converted[idx].Key = string(a.Key)
			converted[idx].Value = log.ValueFromAttribute(a.Value)
			idx++
		}
	}
	rec.AddAttributes(converted...)
}
