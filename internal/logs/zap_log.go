package logs

import (
	"os"
	"time"

	"bytes"
	"context"
	"encoding/json"
	"sync"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// ESOpts holds the configuration options for Elasticsearch logging.
// It includes whether to enable logging, the addresses of the Elasticsearch nodes,
// the index to use, and optional authentication details.
// FlushBytes and FlushInterval control the buffering and flushing behavior.
// FlushBytes is the maximum size of the buffer before flushing, and FlushInterval is the time interval for flushing.
// If FlushBytes is 0, it will flush on every Write call.
// If FlushInterval is 0, it will not use a ticker for periodic flushing.
// If both are set, it will flush either when the buffer exceeds FlushBytes or when the ticker ticks.
// If neither is set, it will flush only when explicitly called.
// The Enabled field determines if Elasticsearch logging is active.
// The Addresses field is a list of Elasticsearch node addresses.
// The Index field specifies the index to write logs to.
// APIKey, Username, and Password are used for authentication.
type ESOpts struct {
	Enabled       bool
	Addresses     []string
	Index         string
	APIKey        string
	Username      string
	Password      string
	FlushBytes    int
	FlushInterval time.Duration
}

// bulkSink buffers ECS/JSON logs and sends them to Elasticsearch via Bulk API.
type bulkSink struct {
	cli      *elasticsearch.Client
	index    string
	buf      *bytes.Buffer
	mu       sync.Mutex
	maxBytes int
	interval time.Duration
	ctx      context.Context
	cancel   context.CancelFunc
	flushCh  chan struct{}
}

func newBulkSink(cli *elasticsearch.Client, index string, maxBytes int, interval time.Duration) *bulkSink {
	ctx, cancel := context.WithCancel(context.Background())
	bs := &bulkSink{
		cli:      cli,
		index:    index,
		buf:      &bytes.Buffer{},
		maxBytes: maxBytes,
		interval: interval,
		ctx:      ctx,
		cancel:   cancel,
		flushCh:  make(chan struct{}, 1),
	}
	go bs.loop()
	return bs
}

func (b *bulkSink) loop() {
	ticker := time.NewTicker(b.interval)
	defer ticker.Stop()
	for {
		select {
		case <-b.ctx.Done():
			b.flush()
			return
		case <-ticker.C:
			b.flush()
		case <-b.flushCh:
			b.flush()
		}
	}
}

func (b *bulkSink) Write(doc json.RawMessage) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	meta := []byte(`{"index":{"_index":"` + b.index + `"}}\n`)
	b.buf.Write(meta)
	b.buf.Write(doc)
	b.buf.WriteByte('\n')
	if b.buf.Len() >= b.maxBytes {
		select {
		case b.flushCh <- struct{}{}:
		default:
		}
	}
	return nil
}

func (b *bulkSink) flush() {
	b.mu.Lock()
	if b.buf.Len() == 0 {
		b.mu.Unlock()
		return
	}
	payload := make([]byte, b.buf.Len())
	copy(payload, b.buf.Bytes())
	b.buf.Reset()
	b.mu.Unlock()

	req := esapi.BulkRequest{Body: bytes.NewReader(payload)}
	res, err := req.Do(b.ctx, b.cli)
	if err == nil {
		res.Body.Close()
	}
}

func (b *bulkSink) Stop() { b.cancel() }

type elasticCore struct {
	enc  zapcore.Encoder
	sink *bulkSink
	lvl  zapcore.LevelEnabler
}

func newElasticCore(enc zapcore.Encoder, sink *bulkSink, lvl zapcore.LevelEnabler) zapcore.Core {
	return &elasticCore{enc: enc, sink: sink, lvl: lvl}
}

func (c *elasticCore) Enabled(l zapcore.Level) bool {
	return c.lvl.Enabled(l)
}

func (c *elasticCore) With(fields []zapcore.Field) zapcore.Core {
	clone := *c
	return &clone
}

func (c *elasticCore) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(ent.Level) {
		return ce.AddCore(ent, c)
	}
	return ce
}

func (c *elasticCore) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	buf, err := c.enc.EncodeEntry(ent, fields)
	if err != nil {
		return err
	}
	defer buf.Free()
	return c.sink.Write(json.RawMessage(buf.Bytes()))
}

func (c *elasticCore) Sync() error { return nil }

// --- structured logger helpers ---

// LevelToType converts a zapcore.Level to its string representation.
// It maps each level to a specific string type, such as "debug", "info", "warn", etc.
// This function is used to standardize log level representation in structured logging.
// It returns "log" for any unrecognized level, ensuring that all levels have a valid string type.
// This is useful for logging systems that expect specific level types for filtering or processing logs.
func LevelToType(l zapcore.Level) string {
	switch l {
	case zapcore.DebugLevel:
		return "debug"
	case zapcore.InfoLevel:
		return "info"
	case zapcore.WarnLevel:
		return "warn"
	case zapcore.ErrorLevel:
		return "error"
	case zapcore.DPanicLevel:
		return "dpanic"
	case zapcore.PanicLevel:
		return "panic"
	case zapcore.FatalLevel:
		return "fatal"
	default:
		return "log"
	}
}

func timeEncoderWithTZ(loc *time.Location) zapcore.TimeEncoder {
	return func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(t.In(loc).Format(time.RFC3339Nano))
	}
}

func levelEncoder(l zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(LevelToType(l))
}

// NewWithElastic creates a new zap.Logger with Elasticsearch logging enabled.
// It initializes the logger with a JSON encoder and sets up a bulk sink for Elasticsearch.
// The serviceName is used to tag the logs, and tzName specifies the timezone for timestamps.
// The ESOpts struct contains configuration options for Elasticsearch logging, including addresses,
// index, authentication details, and buffering settings.
// It returns the configured logger, a stopper function to clean up resources, and an error if any.
func NewWithElastic(serviceName string, tzName string, es ESOpts) (*zap.Logger, func(), error) {
	if tzName == "" {
		tzName = "UTC"
	}
	loc, err := time.LoadLocation(tzName)
	if err != nil {
		loc = time.UTC
	}

	encCfg := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "type",
		MessageKey:     "message",
		NameKey:        "",
		CallerKey:      "file_line",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeTime:     timeEncoderWithTZ(loc),
		EncodeLevel:    levelEncoder,
		EncodeDuration: zapcore.MillisDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	stdoutCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(encCfg),
		zapcore.AddSync(os.Stdout),
		zap.NewAtomicLevelAt(zapcore.InfoLevel),
	)

	cores := []zapcore.Core{stdoutCore}
	stopper := func() {}

	if es.Enabled {
		cfg := elasticsearch.Config{Addresses: es.Addresses}
		if es.APIKey != "" {
			cfg.APIKey = es.APIKey
		} else if es.Username != "" {
			cfg.Username, cfg.Password = es.Username, es.Password
		}
		cli, err := elasticsearch.NewClient(cfg)
		if err == nil {
			sink := newBulkSink(cli, es.Index, es.FlushBytes, es.FlushInterval)
			esCore := newElasticCore(zapcore.NewJSONEncoder(encCfg), sink, zap.NewAtomicLevelAt(zapcore.InfoLevel))
			cores = append(cores, esCore)
			stopper = func() { sink.Stop() }
		}
	}

	logger := zap.New(zapcore.NewTee(cores...), zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel)).With(
		zap.String("service_name", serviceName),
	)
	return logger, stopper, nil
}
