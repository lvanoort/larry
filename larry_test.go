package larry

import (
	"context"
	"fmt"
	"testing"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/log/logtest"
)

type disabledCtxKeyTyp string

const disabledCtxKey disabledCtxKeyTyp = "disableCtxKey"

func TestLogger_GlobalProvider(t *testing.T) {
	recordingProvider := logtest.NewRecorder(
		logtest.WithEnabledFunc(func(ctx context.Context, parameters log.EnabledParameters) bool {
			return ctx.Value(disabledCtxKey) == nil
		}),
	)
	disableCtx := func(ctx context.Context) context.Context {
		return context.WithValue(ctx, disabledCtxKey, true)
	}
	global.SetLoggerProvider(recordingProvider)

	testTime := time.Now().Add(1 * time.Hour)
	timeProvider = func() time.Time { return testTime }
	t.Cleanup(func() {
		timeProvider = time.Now
	})

	testSchema := "https://example.com/testschema"
	testAttr := attribute.String("test.name", t.Name())
	logger := New(
		t.Name(),
		WithLoggerOpts(log.WithSchemaURL(testSchema)),
		WithAttributes(testAttr),
	)

	ctx := context.Background()
	// call the methods on logger to log messages with small attr counts
	logger.Trace(ctx, "trace message", attribute.String("key", "trace-val"))
	logger.Debug(ctx, "debug message", attribute.Int("count", 42))
	logger.Info(ctx, "info message", attribute.Bool("success", true))
	logger.Warn(ctx, "warn message", attribute.Float64("rate", 0.5))
	logger.Error(ctx, "error message", attribute.String("error", "something bad"))

	// call messages that will be ignored due to the disabled context
	disabledContext := disableCtx(ctx)
	logger.Trace(disabledContext, "disabled trace message", attribute.String("key", "trace-val"))
	logger.Debug(disabledContext, "disabled debug message", attribute.Int("count", 42))
	logger.Info(disabledContext, "disabled info message", attribute.Bool("success", true))
	logger.Warn(disabledContext, "disabled warn message", attribute.Float64("rate", 0.5))
	logger.Error(disabledContext, "disabled error message", attribute.String("error", "something bad"))

	// call methods with large attr counts
	manyAttrs := createAttrs(6)
	logger.Trace(ctx, "trace many attrs", manyAttrs...)
	logger.Debug(ctx, "debug many attrs", manyAttrs...)
	logger.Info(ctx, "info many attrs", manyAttrs...)
	logger.Warn(ctx, "warn many attrs", manyAttrs...)
	logger.Error(ctx, "error many attrs", manyAttrs...)

	recording := recordingProvider.Result()

	testLogAttr := log.String(string(testAttr.Key), testAttr.Value.AsString())
	manyLogAttrs := []log.KeyValue{
		testLogAttr,
		log.String("key0", "value0"),
		log.String("key1", "value1"),
		log.String("key2", "value2"),
		log.String("key3", "value3"),
		log.String("key4", "value4"),
		log.String("key5", "value5"),
	}
	// check that all logs are present as expected on recording
	want := logtest.Recording{
		logtest.Scope{
			Name:      t.Name(),
			SchemaURL: testSchema,
		}: []logtest.Record{
			{
				Context:    ctx,
				Severity:   log.SeverityTrace,
				Body:       log.StringValue("trace message"),
				Attributes: []log.KeyValue{testLogAttr, log.String("key", "trace-val")},
				Timestamp:  testTime,
			},
			{
				Context:    ctx,
				Severity:   log.SeverityDebug,
				Body:       log.StringValue("debug message"),
				Attributes: []log.KeyValue{testLogAttr, log.Int64("count", 42)},
				Timestamp:  testTime,
			},
			{
				Context:    ctx,
				Severity:   log.SeverityInfo,
				Body:       log.StringValue("info message"),
				Attributes: []log.KeyValue{testLogAttr, log.Bool("success", true)},
				Timestamp:  testTime,
			},
			{
				Context:    ctx,
				Severity:   log.SeverityWarn,
				Body:       log.StringValue("warn message"),
				Attributes: []log.KeyValue{testLogAttr, log.Float64("rate", 0.5)},
				Timestamp:  testTime,
			},
			{
				Context:    ctx,
				Severity:   log.SeverityError,
				Body:       log.StringValue("error message"),
				Attributes: []log.KeyValue{testLogAttr, log.String("error", "something bad")},
				Timestamp:  testTime,
			},
			{
				Context:    ctx,
				Severity:   log.SeverityTrace,
				Body:       log.StringValue("trace many attrs"),
				Attributes: manyLogAttrs,
				Timestamp:  testTime,
			},
			{
				Context:    ctx,
				Severity:   log.SeverityDebug,
				Body:       log.StringValue("debug many attrs"),
				Attributes: manyLogAttrs,
				Timestamp:  testTime,
			},
			{
				Context:    ctx,
				Severity:   log.SeverityInfo,
				Body:       log.StringValue("info many attrs"),
				Attributes: manyLogAttrs,
				Timestamp:  testTime,
			},
			{
				Context:    ctx,
				Severity:   log.SeverityWarn,
				Body:       log.StringValue("warn many attrs"),
				Attributes: manyLogAttrs,
				Timestamp:  testTime,
			},
			{
				Context:    ctx,
				Severity:   log.SeverityError,
				Body:       log.StringValue("error many attrs"),
				Attributes: manyLogAttrs,
				Timestamp:  testTime,
			},
		},
	}
	logtest.AssertEqual(t, want, recording)
}

func TestBasicHappyPath_ExplicitProvider(t *testing.T) {
	// this test is basically identical to the previous one. At some point they should be refactored
	// so it's all common, but at the moment there's only two of them and the library is so simple
	// that just repeating it here is easier than doing that.
	recordingProvider := logtest.NewRecorder(
		logtest.WithEnabledFunc(func(ctx context.Context, parameters log.EnabledParameters) bool {
			return ctx.Value(disabledCtxKey) == nil
		}),
	)
	disableCtx := func(ctx context.Context) context.Context {
		return context.WithValue(ctx, disabledCtxKey, true)
	}

	testTime := time.Now().Add(1 * time.Hour)
	timeProvider = func() time.Time { return testTime }
	t.Cleanup(func() {
		timeProvider = time.Now
	})

	schemaUrl := "https://example.com/testschema"
	testAttr := attribute.String("test.name", t.Name())

	ctx := context.Background()
	logger := New(
		t.Name(),
		WithLoggerProvider(recordingProvider),
		WithLoggerOpts(log.WithSchemaURL(schemaUrl)),
		WithAttributes(testAttr),
	)

	// call the methods on logger to log messages with a small attr count
	logger.Trace(ctx, "trace message", attribute.String("key", "trace-val"))
	logger.Debug(ctx, "debug message", attribute.Int("count", 42))
	logger.Info(ctx, "info message", attribute.Bool("success", true))
	logger.Warn(ctx, "warn message", attribute.Float64("rate", 0.5))
	logger.Error(ctx, "error message", attribute.String("error", "something bad"))

	// call messages that will be ignored due to the disabled context
	disabledContext := disableCtx(ctx)
	logger.Trace(disabledContext, "disabled trace message", attribute.String("key", "trace-val"))
	logger.Debug(disabledContext, "disabled debug message", attribute.Int("count", 42))
	logger.Info(disabledContext, "disabled info message", attribute.Bool("success", true))
	logger.Warn(disabledContext, "disabled warn message", attribute.Float64("rate", 0.5))
	logger.Error(disabledContext, "disabled error message", attribute.String("error", "something bad"))

	// call methods with a large attr count
	manyAttrs := createAttrs(6)
	logger.Trace(ctx, "trace many attrs", manyAttrs...)
	logger.Debug(ctx, "debug many attrs", manyAttrs...)
	logger.Info(ctx, "info many attrs", manyAttrs...)
	logger.Warn(ctx, "warn many attrs", manyAttrs...)
	logger.Error(ctx, "error many attrs", manyAttrs...)

	recording := recordingProvider.Result()

	testLogAttr := log.String(string(testAttr.Key), testAttr.Value.AsString())
	manyLogAttrs := []log.KeyValue{
		testLogAttr,
		log.String("key0", "value0"),
		log.String("key1", "value1"),
		log.String("key2", "value2"),
		log.String("key3", "value3"),
		log.String("key4", "value4"),
		log.String("key5", "value5"),
	}
	// check that all logs are present as expected on recording
	want := logtest.Recording{
		logtest.Scope{
			Name:      t.Name(),
			SchemaURL: schemaUrl,
		}: []logtest.Record{
			{
				Context:    ctx,
				Severity:   log.SeverityTrace,
				Body:       log.StringValue("trace message"),
				Attributes: []log.KeyValue{testLogAttr, log.String("key", "trace-val")},
				Timestamp:  testTime,
			},
			{
				Context:    ctx,
				Severity:   log.SeverityDebug,
				Body:       log.StringValue("debug message"),
				Attributes: []log.KeyValue{testLogAttr, log.Int64("count", 42)},
				Timestamp:  testTime,
			},
			{
				Context:    ctx,
				Severity:   log.SeverityInfo,
				Body:       log.StringValue("info message"),
				Attributes: []log.KeyValue{testLogAttr, log.Bool("success", true)},
				Timestamp:  testTime,
			},
			{
				Context:    ctx,
				Severity:   log.SeverityWarn,
				Body:       log.StringValue("warn message"),
				Attributes: []log.KeyValue{testLogAttr, log.Float64("rate", 0.5)},
				Timestamp:  testTime,
			},
			{
				Context:    ctx,
				Severity:   log.SeverityError,
				Body:       log.StringValue("error message"),
				Attributes: []log.KeyValue{testLogAttr, log.String("error", "something bad")},
				Timestamp:  testTime,
			},
			{
				Context:    ctx,
				Severity:   log.SeverityTrace,
				Body:       log.StringValue("trace many attrs"),
				Attributes: manyLogAttrs,
				Timestamp:  testTime,
			},
			{
				Context:    ctx,
				Severity:   log.SeverityDebug,
				Body:       log.StringValue("debug many attrs"),
				Attributes: manyLogAttrs,
				Timestamp:  testTime,
			},
			{
				Context:    ctx,
				Severity:   log.SeverityInfo,
				Body:       log.StringValue("info many attrs"),
				Attributes: manyLogAttrs,
				Timestamp:  testTime,
			},
			{
				Context:    ctx,
				Severity:   log.SeverityWarn,
				Body:       log.StringValue("warn many attrs"),
				Attributes: manyLogAttrs,
				Timestamp:  testTime,
			},
			{
				Context:    ctx,
				Severity:   log.SeverityError,
				Body:       log.StringValue("error many attrs"),
				Attributes: manyLogAttrs,
				Timestamp:  testTime,
			},
		},
	}
	logtest.AssertEqual(t, want, recording)
}

func TestLogger_With(t *testing.T) {
	recordingProvider := logtest.NewRecorder(
		logtest.WithEnabledFunc(func(ctx context.Context, parameters log.EnabledParameters) bool {
			return ctx.Value(disabledCtxKey) == nil
		}),
	)
	disableCtx := func(ctx context.Context) context.Context {
		return context.WithValue(ctx, disabledCtxKey, true)
	}

	testTime := time.Now().Add(1 * time.Hour)
	timeProvider = func() time.Time { return testTime }
	t.Cleanup(func() {
		timeProvider = time.Now
	})

	schemaUrl := "https://example.com/testschema"
	parentAttr := attribute.String("parent.name", t.Name())
	withAttr := attribute.String("with.name", t.Name())

	ctx := context.Background()
	parent := New(
		t.Name(),
		WithLoggerProvider(recordingProvider),
		WithLoggerOpts(log.WithSchemaURL(schemaUrl)),
		WithAttributes(parentAttr),
	)
	logger := parent.With(withAttr)

	// call the methods on the With() logger to log messages with a small attr count
	logger.Trace(ctx, "trace message", attribute.String("key", "trace-val"))
	logger.Debug(ctx, "debug message", attribute.Int("count", 42))
	logger.Info(ctx, "info message", attribute.Bool("success", true))
	logger.Warn(ctx, "warn message", attribute.Float64("rate", 0.5))
	logger.Error(ctx, "error message", attribute.String("error", "something bad"))

	// call messages that will be ignored due to the disabled context
	disabledContext := disableCtx(ctx)
	logger.Trace(disabledContext, "disabled trace message", attribute.String("key", "trace-val"))
	logger.Debug(disabledContext, "disabled debug message", attribute.Int("count", 42))
	logger.Info(disabledContext, "disabled info message", attribute.Bool("success", true))
	logger.Warn(disabledContext, "disabled warn message", attribute.Float64("rate", 0.5))
	logger.Error(disabledContext, "disabled error message", attribute.String("error", "something bad"))

	// call methods with a large attr count
	manyAttrs := createAttrs(6)
	logger.Trace(ctx, "trace many attrs", manyAttrs...)
	logger.Debug(ctx, "debug many attrs", manyAttrs...)
	logger.Info(ctx, "info many attrs", manyAttrs...)
	logger.Warn(ctx, "warn many attrs", manyAttrs...)
	logger.Error(ctx, "error many attrs", manyAttrs...)

	recording := recordingProvider.Result()

	// the With() logger emits its own attributes first, then the call-time attributes,
	// then the parent logger's attributes
	parentLogAttr := log.String(string(parentAttr.Key), parentAttr.Value.AsString())
	withLogAttr := log.String(string(withAttr.Key), withAttr.Value.AsString())
	manyLogAttrs := []log.KeyValue{
		withLogAttr,
		log.String("key0", "value0"),
		log.String("key1", "value1"),
		log.String("key2", "value2"),
		log.String("key3", "value3"),
		log.String("key4", "value4"),
		log.String("key5", "value5"),
		parentLogAttr,
	}
	// check that all logs are present as expected on recording, with both the parent
	// and With() attributes included at every log level
	want := logtest.Recording{
		logtest.Scope{
			Name:      t.Name(),
			SchemaURL: schemaUrl,
		}: []logtest.Record{
			{
				Context:    ctx,
				Severity:   log.SeverityTrace,
				Body:       log.StringValue("trace message"),
				Attributes: []log.KeyValue{withLogAttr, log.String("key", "trace-val"), parentLogAttr},
				Timestamp:  testTime,
			},
			{
				Context:    ctx,
				Severity:   log.SeverityDebug,
				Body:       log.StringValue("debug message"),
				Attributes: []log.KeyValue{withLogAttr, log.Int64("count", 42), parentLogAttr},
				Timestamp:  testTime,
			},
			{
				Context:    ctx,
				Severity:   log.SeverityInfo,
				Body:       log.StringValue("info message"),
				Attributes: []log.KeyValue{withLogAttr, log.Bool("success", true), parentLogAttr},
				Timestamp:  testTime,
			},
			{
				Context:    ctx,
				Severity:   log.SeverityWarn,
				Body:       log.StringValue("warn message"),
				Attributes: []log.KeyValue{withLogAttr, log.Float64("rate", 0.5), parentLogAttr},
				Timestamp:  testTime,
			},
			{
				Context:    ctx,
				Severity:   log.SeverityError,
				Body:       log.StringValue("error message"),
				Attributes: []log.KeyValue{withLogAttr, log.String("error", "something bad"), parentLogAttr},
				Timestamp:  testTime,
			},
			{
				Context:    ctx,
				Severity:   log.SeverityTrace,
				Body:       log.StringValue("trace many attrs"),
				Attributes: manyLogAttrs,
				Timestamp:  testTime,
			},
			{
				Context:    ctx,
				Severity:   log.SeverityDebug,
				Body:       log.StringValue("debug many attrs"),
				Attributes: manyLogAttrs,
				Timestamp:  testTime,
			},
			{
				Context:    ctx,
				Severity:   log.SeverityInfo,
				Body:       log.StringValue("info many attrs"),
				Attributes: manyLogAttrs,
				Timestamp:  testTime,
			},
			{
				Context:    ctx,
				Severity:   log.SeverityWarn,
				Body:       log.StringValue("warn many attrs"),
				Attributes: manyLogAttrs,
				Timestamp:  testTime,
			},
			{
				Context:    ctx,
				Severity:   log.SeverityError,
				Body:       log.StringValue("error many attrs"),
				Attributes: manyLogAttrs,
				Timestamp:  testTime,
			},
		},
	}
	logtest.AssertEqual(t, want, recording)
}

func TestLogger_Child(t *testing.T) {
	recordingProvider := logtest.NewRecorder(
		logtest.WithEnabledFunc(func(ctx context.Context, parameters log.EnabledParameters) bool {
			return ctx.Value(disabledCtxKey) == nil
		}),
	)
	disableCtx := func(ctx context.Context) context.Context {
		return context.WithValue(ctx, disabledCtxKey, true)
	}

	testTime := time.Now().Add(1 * time.Hour)
	timeProvider = func() time.Time { return testTime }
	t.Cleanup(func() {
		timeProvider = time.Now
	})

	// give the parent and child distinct scopes so we can prove the child uses its own
	// backing OTel logger rather than reusing the parent's
	parentName := t.Name() + "/parent"
	childName := t.Name() + "/child"
	parentSchemaUrl := "https://example.com/parentschema"
	childSchemaUrl := "https://example.com/childschema"
	parentAttr := attribute.String("parent.name", t.Name())
	childAttr := attribute.String("child.name", t.Name())

	ctx := context.Background()
	parent := New(
		parentName,
		WithLoggerProvider(recordingProvider),
		WithLoggerOpts(log.WithSchemaURL(parentSchemaUrl)),
		WithAttributes(parentAttr),
	)
	logger := parent.Child(
		childName,
		WithLoggerProvider(recordingProvider),
		WithLoggerOpts(log.WithSchemaURL(childSchemaUrl)),
		WithAttributes(childAttr),
	)

	// the parent emits its own message so the recording contains a separate scope,
	// demonstrating the child does not reuse the parent's backing logger
	parent.Info(ctx, "parent message", attribute.String("scope", "parent"))

	// call the methods on the child logger to log messages with a small attr count
	logger.Trace(ctx, "trace message", attribute.String("key", "trace-val"))
	logger.Debug(ctx, "debug message", attribute.Int("count", 42))
	logger.Info(ctx, "info message", attribute.Bool("success", true))
	logger.Warn(ctx, "warn message", attribute.Float64("rate", 0.5))
	logger.Error(ctx, "error message", attribute.String("error", "something bad"))

	// call messages that will be ignored due to the disabled context
	disabledContext := disableCtx(ctx)
	logger.Trace(disabledContext, "disabled trace message", attribute.String("key", "trace-val"))
	logger.Debug(disabledContext, "disabled debug message", attribute.Int("count", 42))
	logger.Info(disabledContext, "disabled info message", attribute.Bool("success", true))
	logger.Warn(disabledContext, "disabled warn message", attribute.Float64("rate", 0.5))
	logger.Error(disabledContext, "disabled error message", attribute.String("error", "something bad"))

	// call methods with a large attr count
	manyAttrs := createAttrs(6)
	logger.Trace(ctx, "trace many attrs", manyAttrs...)
	logger.Debug(ctx, "debug many attrs", manyAttrs...)
	logger.Info(ctx, "info many attrs", manyAttrs...)
	logger.Warn(ctx, "warn many attrs", manyAttrs...)
	logger.Error(ctx, "error many attrs", manyAttrs...)

	recording := recordingProvider.Result()

	// the child logger emits its own attributes first, then the call-time attributes,
	// then the parent logger's attributes
	parentLogAttr := log.String(string(parentAttr.Key), parentAttr.Value.AsString())
	childLogAttr := log.String(string(childAttr.Key), childAttr.Value.AsString())
	manyLogAttrs := []log.KeyValue{
		childLogAttr,
		log.String("key0", "value0"),
		log.String("key1", "value1"),
		log.String("key2", "value2"),
		log.String("key3", "value3"),
		log.String("key4", "value4"),
		log.String("key5", "value5"),
		parentLogAttr,
	}
	// check that the parent and child each recorded under their own scope, and that the
	// child's messages include both the parent and child attributes at every log level
	want := logtest.Recording{
		logtest.Scope{
			Name:      parentName,
			SchemaURL: parentSchemaUrl,
		}: []logtest.Record{
			{
				Context:    ctx,
				Severity:   log.SeverityInfo,
				Body:       log.StringValue("parent message"),
				Attributes: []log.KeyValue{parentLogAttr, log.String("scope", "parent")},
				Timestamp:  testTime,
			},
		},
		logtest.Scope{
			Name:      childName,
			SchemaURL: childSchemaUrl,
		}: []logtest.Record{
			{
				Context:    ctx,
				Severity:   log.SeverityTrace,
				Body:       log.StringValue("trace message"),
				Attributes: []log.KeyValue{childLogAttr, log.String("key", "trace-val"), parentLogAttr},
				Timestamp:  testTime,
			},
			{
				Context:    ctx,
				Severity:   log.SeverityDebug,
				Body:       log.StringValue("debug message"),
				Attributes: []log.KeyValue{childLogAttr, log.Int64("count", 42), parentLogAttr},
				Timestamp:  testTime,
			},
			{
				Context:    ctx,
				Severity:   log.SeverityInfo,
				Body:       log.StringValue("info message"),
				Attributes: []log.KeyValue{childLogAttr, log.Bool("success", true), parentLogAttr},
				Timestamp:  testTime,
			},
			{
				Context:    ctx,
				Severity:   log.SeverityWarn,
				Body:       log.StringValue("warn message"),
				Attributes: []log.KeyValue{childLogAttr, log.Float64("rate", 0.5), parentLogAttr},
				Timestamp:  testTime,
			},
			{
				Context:    ctx,
				Severity:   log.SeverityError,
				Body:       log.StringValue("error message"),
				Attributes: []log.KeyValue{childLogAttr, log.String("error", "something bad"), parentLogAttr},
				Timestamp:  testTime,
			},
			{
				Context:    ctx,
				Severity:   log.SeverityTrace,
				Body:       log.StringValue("trace many attrs"),
				Attributes: manyLogAttrs,
				Timestamp:  testTime,
			},
			{
				Context:    ctx,
				Severity:   log.SeverityDebug,
				Body:       log.StringValue("debug many attrs"),
				Attributes: manyLogAttrs,
				Timestamp:  testTime,
			},
			{
				Context:    ctx,
				Severity:   log.SeverityInfo,
				Body:       log.StringValue("info many attrs"),
				Attributes: manyLogAttrs,
				Timestamp:  testTime,
			},
			{
				Context:    ctx,
				Severity:   log.SeverityWarn,
				Body:       log.StringValue("warn many attrs"),
				Attributes: manyLogAttrs,
				Timestamp:  testTime,
			},
			{
				Context:    ctx,
				Severity:   log.SeverityError,
				Body:       log.StringValue("error many attrs"),
				Attributes: manyLogAttrs,
				Timestamp:  testTime,
			},
		},
	}
	logtest.AssertEqual(t, want, recording)
}

func createAttrs(count int) []attribute.KeyValue {
	result := make([]attribute.KeyValue, count)
	for idx := range result {
		result[idx].Key = attribute.Key(fmt.Sprintf("key%d", idx))
		result[idx].Value = attribute.StringValue(fmt.Sprintf("value%d", idx))
	}

	return result
}

func benchAddAttrSlice(b *testing.B, count int) {
	atA := count / 3
	atB := (count - atA) / 2
	atC := count - (atA + atB)
	testAttrsA := createAttrs(atA)
	testAttrsB := createAttrs(atB)
	testAttrsC := createAttrs(atC)
	b.ResetTimer()
	for b.Loop() {
		record := &log.Record{}
		addAttrsToRecordSlice(record, count, testAttrsA, testAttrsB, testAttrsC)
	}
}

func benchAddAttrDirect(b *testing.B, count int) {
	atA := count / 3
	atB := (count - atA) / 2
	atC := count - (atA + atB)
	testAttrsA := createAttrs(atA)
	testAttrsB := createAttrs(atB)
	testAttrsC := createAttrs(atC)
	b.ResetTimer()
	for b.Loop() {
		record := &log.Record{}
		addAttrsToRecordDirect(record, testAttrsA, testAttrsB, testAttrsC)
	}
}

func benchAddAttrAdaptive(b *testing.B, count int) {
	atA := count / 3
	atB := (count - atA) / 2
	atC := count - (atA + atB)
	testAttrsA := createAttrs(atA)
	testAttrsB := createAttrs(atB)
	testAttrsC := createAttrs(atC)
	b.ResetTimer()
	for b.Loop() {
		record := &log.Record{}
		addAttrsToRecordAdaptive(record, testAttrsA, testAttrsB, testAttrsC)
	}
}

const smallAttrCount = 3      // largely arbitrarily chosen, just important to fit into OTel's small attr count optimization
const typAttrCount = 5        // max out record's small attr optimization
const largeAttrCount = 10     // arbitrarily chosen to exceed the small attr optimization
const veryLargeAttrCount = 20 // arbitrarily chosen to significantly exceed the small attr optimization
const hugeAttrCount = 100     // absurd attr count

func BenchmarkAddAttrsToRecordDirect(b *testing.B) {
	b.Run("smallAttrCount", func(b *testing.B) {
		benchAddAttrDirect(b, smallAttrCount)
	})
	b.Run("typAttrCount", func(b *testing.B) {
		benchAddAttrDirect(b, typAttrCount)
	})
	b.Run("largeAttrCount", func(b *testing.B) {
		benchAddAttrDirect(b, largeAttrCount)
	})
	b.Run("veryLargeAttrCount", func(b *testing.B) {
		benchAddAttrDirect(b, veryLargeAttrCount)
	})
	b.Run("hugeAttrCount", func(b *testing.B) {
		benchAddAttrDirect(b, hugeAttrCount)
	})
}

func BenchmarkAddAttrsToRecordSlice(b *testing.B) {
	b.Run("smallAttrCount", func(b *testing.B) {
		benchAddAttrSlice(b, smallAttrCount)
	})
	b.Run("typAttrCount", func(b *testing.B) {
		benchAddAttrSlice(b, typAttrCount)
	})
	b.Run("largeAttrCount", func(b *testing.B) {
		benchAddAttrSlice(b, largeAttrCount)
	})
	b.Run("veryLargeAttrCount", func(b *testing.B) {
		benchAddAttrSlice(b, veryLargeAttrCount)
	})
	b.Run("hugeAttrCount", func(b *testing.B) {
		benchAddAttrSlice(b, hugeAttrCount)
	})
}

func BenchmarkAddAttrsToRecordAdaptive(b *testing.B) {
	b.Run("smallAttrCount", func(b *testing.B) {
		benchAddAttrAdaptive(b, smallAttrCount)
	})
	b.Run("typAttrCount", func(b *testing.B) {
		benchAddAttrAdaptive(b, typAttrCount)
	})
	b.Run("largeAttrCount", func(b *testing.B) {
		benchAddAttrAdaptive(b, largeAttrCount)
	})
	b.Run("veryLargeAttrCount", func(b *testing.B) {
		benchAddAttrAdaptive(b, veryLargeAttrCount)
	})
	b.Run("hugeAttrCount", func(b *testing.B) {
		benchAddAttrAdaptive(b, hugeAttrCount)
	})
}
