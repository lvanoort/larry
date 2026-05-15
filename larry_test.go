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
	logger.Fatal(ctx, "fatal message", attribute.Int("code", 1))

	// call messages that will be ignored due to the disabled context
	disabledContext := disableCtx(ctx)
	logger.Trace(disabledContext, "disabled trace message", attribute.String("key", "trace-val"))
	logger.Debug(disabledContext, "disabled debug message", attribute.Int("count", 42))
	logger.Info(disabledContext, "disabled info message", attribute.Bool("success", true))
	logger.Warn(disabledContext, "disabled warn message", attribute.Float64("rate", 0.5))
	logger.Error(disabledContext, "disabled error message", attribute.String("error", "something bad"))
	logger.Fatal(disabledContext, "disabled fatal message", attribute.Int("code", 1))

	// call methods with large attr counts
	manyAttrs := createAttrs(6)
	logger.Trace(ctx, "trace many attrs", manyAttrs...)
	logger.Debug(ctx, "debug many attrs", manyAttrs...)
	logger.Info(ctx, "info many attrs", manyAttrs...)
	logger.Warn(ctx, "warn many attrs", manyAttrs...)
	logger.Error(ctx, "error many attrs", manyAttrs...)
	logger.Fatal(ctx, "fatal many attrs", manyAttrs...)

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
				Severity:   log.SeverityFatal,
				Body:       log.StringValue("fatal message"),
				Attributes: []log.KeyValue{testLogAttr, log.Int64("code", 1)},
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
			{
				Context:    ctx,
				Severity:   log.SeverityFatal,
				Body:       log.StringValue("fatal many attrs"),
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
	logger.Fatal(ctx, "fatal message", attribute.Int("code", 1))

	// call messages that will be ignored due to the disabled context
	disabledContext := disableCtx(ctx)
	logger.Trace(disabledContext, "disabled trace message", attribute.String("key", "trace-val"))
	logger.Debug(disabledContext, "disabled debug message", attribute.Int("count", 42))
	logger.Info(disabledContext, "disabled info message", attribute.Bool("success", true))
	logger.Warn(disabledContext, "disabled warn message", attribute.Float64("rate", 0.5))
	logger.Error(disabledContext, "disabled error message", attribute.String("error", "something bad"))
	logger.Fatal(disabledContext, "disabled fatal message", attribute.Int("code", 1))

	// call methods with a large attr count
	manyAttrs := createAttrs(6)
	logger.Trace(ctx, "trace many attrs", manyAttrs...)
	logger.Debug(ctx, "debug many attrs", manyAttrs...)
	logger.Info(ctx, "info many attrs", manyAttrs...)
	logger.Warn(ctx, "warn many attrs", manyAttrs...)
	logger.Error(ctx, "error many attrs", manyAttrs...)
	logger.Fatal(ctx, "fatal many attrs", manyAttrs...)

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
				Severity:   log.SeverityFatal,
				Body:       log.StringValue("fatal message"),
				Attributes: []log.KeyValue{testLogAttr, log.Int64("code", 1)},
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
			{
				Context:    ctx,
				Severity:   log.SeverityFatal,
				Body:       log.StringValue("fatal many attrs"),
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
	testAttrs := createAttrs(count)
	b.ResetTimer()
	for b.Loop() {
		record := &log.Record{}
		addAttrsToRecordSlice(record, count, testAttrs)
	}
}
func benchAddAttrDirect(b *testing.B, count int) {
	testAttrs := createAttrs(count)
	b.ResetTimer()
	for b.Loop() {
		record := &log.Record{}
		addAttrsToRecordDirect(record, testAttrs)
	}
}
func benchAddAttrAdaptive(b *testing.B, count int) {
	testAttrs := createAttrs(count)
	b.ResetTimer()
	for b.Loop() {
		record := &log.Record{}
		addAttrsToRecordAdaptive(record, testAttrs)
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
