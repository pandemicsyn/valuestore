package valuestore

import (
	"log"
	"math"
	"math/rand"
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/gholt/ring"
	"github.com/gholt/valuelocmap"
)

type LogFunc func(format string, v ...interface{})

// Config represents the set of values for configuring a ValueStore. Note that
// changing the values (shallow changes) in this structure will have no effect
// on existing ValueStores; but deep changes (such as reconfiguring an existing
// Logger) will.
type Config struct {
	// LogCritical sets the func to use for critical messages. Defaults logging
	// to os.Stderr.
	LogCritical LogFunc
	// LogError sets the func to use for error messages. Defaults logging to
	// os.Stderr.
	LogError LogFunc
	// LogWarning sets the func to use for warning messages. Defaults logging
	// to os.Stderr.
	LogWarning LogFunc
	// LogInfo sets the func to use for info messages. Defaults logging to
	// os.Stdout.
	LogInfo LogFunc
	// LogDebug sets the func to use for debug messages. Defaults not logging
	// debug messages.
	LogDebug LogFunc
	// Rand sets the rand.Rand to use as a random data source. Defaults to a
	// new randomizer based on the current time.
	Rand *rand.Rand
	// Path sets the path where values files will be written; tocvalues files
	// will also be written here unless overridden with PathTOC. Defaults to
	// the current working directory.
	Path string
	// PathTOC sets the path where tocvalues files will be written. Defaults to
	// the Path value.
	PathTOC string
	// ValueCap indicates the maximum number of bytes any given value may be.
	// Defaults to 4,194,304 bytes.
	ValueCap int
	// BackgroundInterval indicates the minimum number of seconds between the
	// starts of background passes (such as discarding expired tombstones
	// [deletion markers]). If set to 60 seconds and the passes take 10 seconds
	// to run, they will wait 50 seconds (with a small amount of randomization)
	// between the stop of one run and the start of the next. This is really
	// just meant to keep nearly empty structures from using a lot of resources
	// doing nearly nothing. Normally, you'd want your background passes to be
	// running constantly so that they are as fast as possible and the load
	// constant. The default of 60 seconds is almost always fine.
	BackgroundInterval int
	// Workers indicates how many goroutines may be used for various tasks
	// (processing incoming writes and batching them to disk, background tasks,
	// etc.). This will also have an impact on memory usage. Defaults to
	// GOMAXPROCS.
	Workers int
	// ChecksumInterval indicates how many bytes are output to a file before a
	// 4-byte checksum is also output. Defaults to 65,532 bytes.
	ChecksumInterval int
	// PageSize controls the size of each chunk of memory allocated. Defaults
	// to 4,194,304 bytes.
	PageSize      int
	minValueAlloc int
	// WritePagesPerWorker controls how many pages are created per worker for
	// caching recently written values. Defaults to 3.
	WritePagesPerWorker int
	// ValueLocMap allows overriding the default ValueLocMap, an interface used
	// by ValueStore for tracking the mappings from keys to the locations of
	// their values. Defaults to github.com/gholt/valuelocmap.New().
	ValueLocMap valuelocmap.ValueLocMap
	// MsgRing sets the ring.MsgRing to use for determining the key ranges the
	// ValueStore is responsible for as well as providing methods to send
	// messages to other nodes.
	MsgRing ring.MsgRing
	// MsgCap indicates the maximum bytes for outgoing messages. Defaults to
	// 16,777,216 bytes.
	MsgCap int
	// MsgTimeout indicates the maximum milliseconds a message can be pending
	// before just discarding it. Defaults to 100 milliseconds.
	MsgTimeout int
	// ValuesFileCap indicates how large a values file can be before closing it
	// and opening a new one. Defaults to 4,294,967,295 bytes.
	ValuesFileCap int
	// ValuesFileReaders indicates how many open file descriptors are allowed
	// per values file for reading. Defaults to Workers.
	ValuesFileReaders int
	// RecoveryBatchSize indicates how many keys to set in a batch while
	// performing recovery (initial start up). Defaults to 1,048,576 keys.
	RecoveryBatchSize int
	// TombstoneDiscardInterval overrides the BackgroundInterval value just for
	// discard passes (discarding expired tombstones [deletion markers]).
	TombstoneDiscardInterval int
	// TombstoneDiscardBatchSize indicates how many items to queue up before
	// pausing a scan, issuing the actual discards, and resuming the scan
	// again. Defaults to 1,048,576 items.
	TombstoneDiscardBatchSize int
	// TombstoneAge indicates how many seconds old a deletion marker may be
	// before it is permanently removed. Defaults to 14,400 seconds (4 hours).
	TombstoneAge int
	// ReplicationIgnoreRecent indicates how many seconds old a value should be
	// before it is included in replication processing. Defaults to 60 seconds.
	ReplicationIgnoreRecent int
	// OutPullReplicationInterval overrides the BackgroundInterval value just
	// for outgoing pull replication passes.
	OutPullReplicationInterval int
	// OutPullReplicationWorkers indicates how many goroutines may be used for
	// an outgoing pull replication pass. Defaults to Workers.
	OutPullReplicationWorkers int
	// OutPullReplicationMsgs indicates how many outgoing pull-replication
	// messages can be buffered before blocking on creating more. Defaults to
	// OutPullReplicationWorkers * 4.
	OutPullReplicationMsgs int
	// OutPullReplicationBloomN indicates the N-factor for the outgoing
	// pull-replication bloom filters. This indicates how many keys the bloom
	// filter can reasonably hold and, in combination with the P-factor,
	// affects memory usage. Defaults to 1,000,000.
	OutPullReplicationBloomN int
	// OutPullReplicationBloomP indicates the P-factor for the outgoing
	// pull-replication bloom filters. This indicates the desired percentage
	// chance of a collision within the bloom filter and, in combination with
	// the N-factor, affects memory usage. Defaults to 0.001.
	OutPullReplicationBloomP float64
	// OutPullReplicationMsgTimeout indicates the maximum milliseconds an
	// outgoing pull replication message can be pending before just discarding
	// it. Defaults to MsgTimeout.
	OutPullReplicationMsgTimeout int
	// InPullReplicationWorkers indicates how many incoming pull-replication
	// messages can be processed at the same time. Defaults to Workers.
	InPullReplicationWorkers int
	// InPullReplicationMsgs indicates how many incoming pull-replication
	// messages can be buffered before dropping additional ones. Defaults to
	// InPullReplicationWorkers * 4.
	InPullReplicationMsgs int
	// InPullReplicationResponseMsgTimeout indicates the maximum milliseconds
	// an outgoing response message to an incoming pull replication message can
	// be pending before just discarding it. Defaults to MsgTimeout.
	InPullReplicationResponseMsgTimeout int
	// OutPushReplicationInterval overrides the BackgroundInterval value just
	// for outgoing push replication passes.
	OutPushReplicationInterval int
	// OutPushReplicationWorkers indicates how many goroutines may be used for
	// an outgoing push replication pass. Defaults to Workers.
	OutPushReplicationWorkers int
	// OutPushReplicationMsgs indicates how many outgoing push-replication
	// messages can be buffered before blocking on creating more. Defaults to
	// OutPushReplicationWorkers * 4.
	OutPushReplicationMsgs int
	// OutPushReplicationMsgTimeout indicates the maximum milliseconds an
	// outgoing push replication message can be pending before just discarding
	// it. Defaults to MsgTimeout.
	OutPushReplicationMsgTimeout int
	// BulkSetMsgCap indicates the maximum bytes for bulk-set messages.
	// Defaults to MsgCap.
	BulkSetMsgCap int
	// OutBulkSetMsgs indicates how many outgoing bulk-set messages can be
	// buffered before blocking on creating more. Defaults to
	// OutPushReplicationWorkers * 4.
	OutBulkSetMsgs int
	// InBulkSetWorkers indicates how many incoming bulk-set messages can be
	// processed at the same time. Defaults to Workers.
	InBulkSetWorkers int
	// InBulkSetMsgs indicates how many incoming bulk-set messages can be
	// buffered before dropping additional ones. Defaults to InBulkSetWorkers *
	// 4.
	InBulkSetMsgs int
	// InBulkSetResponseMsgTimeout indicates the maximum milliseconds a
	// response message to an incoming bulk-set message can be pending before
	// just discarding it. Defaults to MsgTimeout.
	InBulkSetResponseMsgTimeout int
	// BulkSetAckMsgCap indicates the maximum bytes for bulk-set-ack messages.
	// Defaults to MsgCap.
	BulkSetAckMsgCap int
	// InBulkSetAckWorkers indicates how many incoming bulk-set-ack messages
	// can be processed at the same time. Defaults to Workers.
	InBulkSetAckWorkers int
	// InBulkSetAckMsgs indicates how many incoming bulk-set-ack messages can
	// be buffered before dropping additional ones. Defaults to
	// InBulkSetAckWorkers * 4.
	InBulkSetAckMsgs int
	// OutBulkSetAckMsgs indicates how many outgoing bulk-set-ack messages can
	// be buffered before blocking on creating more. Defaults to
	// InBulkSetWorkers * 4.
	OutBulkSetAckMsgs int
	// CompactionInterval overrides the BackgroundInterval value just for
	// compaction passes.
	CompactionInterval int
	// CompactionWorkers indicates how much concurrency is allowed for
	// compaction. Defaults to Workers.
	CompactionWorkers int
	// CompactionThreshold indicates how much waste a given file may have
	// before it is compacted. Defaults to 0.10 (10%).
	CompactionThreshold float64
	// CompactionAgeThreshold indicates how old a given file must be before it
	// is considered for compaction. Defaults to 300 seconds.
	CompactionAgeThreshold int
}

func resolveConfig(c *Config) *Config {
	cfg := &Config{}
	if c != nil {
		*cfg = *c
	}
	if cfg.LogCritical == nil {
		cfg.LogCritical = log.New(os.Stderr, "ValueStore ", log.LstdFlags).Printf
	}
	if cfg.LogError == nil {
		cfg.LogError = log.New(os.Stderr, "ValueStore ", log.LstdFlags).Printf
	}
	if cfg.LogWarning == nil {
		cfg.LogWarning = log.New(os.Stderr, "ValueStore ", log.LstdFlags).Printf
	}
	if cfg.LogInfo == nil {
		cfg.LogInfo = log.New(os.Stdout, "ValueStore ", log.LstdFlags).Printf
	}
	// LogDebug set as nil is fine and shortcircuits any debug code.
	if cfg.Rand == nil {
		cfg.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))
	}
	if env := os.Getenv("VALUESTORE_PATH"); env != "" {
		cfg.Path = env
	}
	if cfg.Path == "" {
		cfg.Path = "."
	}
	if env := os.Getenv("VALUESTORE_PATH_TOC"); env != "" {
		cfg.PathTOC = env
	}
	if cfg.PathTOC == "" {
		cfg.PathTOC = cfg.Path
	}
	if env := os.Getenv("VALUESTORE_VALUE_CAP"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			cfg.ValueCap = val
		}
	}
	if cfg.ValueCap == 0 {
		cfg.ValueCap = 4 * 1024 * 1024
	}
	if cfg.ValueCap < 0 {
		cfg.ValueCap = 0
	}
	if cfg.ValueCap > math.MaxUint32 {
		cfg.ValueCap = math.MaxUint32
	}
	if env := os.Getenv("VALUESTORE_BACKGROUND_INTERVAL"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			cfg.BackgroundInterval = val
		}
	}
	if cfg.BackgroundInterval == 0 {
		cfg.BackgroundInterval = 60
	}
	if cfg.BackgroundInterval < 1 {
		cfg.BackgroundInterval = 1
	}
	if env := os.Getenv("VALUESTORE_WORKERS"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			cfg.Workers = val
		}
	}
	if cfg.Workers == 0 {
		cfg.Workers = runtime.GOMAXPROCS(0)
	}
	if cfg.Workers < 1 {
		cfg.Workers = 1
	}
	if env := os.Getenv("VALUESTORE_CHECKSUM_INTERVAL"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			cfg.ChecksumInterval = val
		}
	}
	if cfg.ChecksumInterval == 0 {
		cfg.ChecksumInterval = 64*1024 - 4
	}
	if cfg.ChecksumInterval < 1 {
		cfg.ChecksumInterval = 1
	}
	if env := os.Getenv("VALUESTORE_PAGE_SIZE"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			cfg.PageSize = val
		}
	}
	if cfg.PageSize == 0 {
		cfg.PageSize = 4 * 1024 * 1024
	}
	// Ensure each page will have at least ChecksumInterval worth of data in it
	// so that each page written will at least flush the previous page's data.
	if cfg.PageSize < cfg.ValueCap+cfg.ChecksumInterval {
		cfg.PageSize = cfg.ValueCap + cfg.ChecksumInterval
	}
	// Absolute minimum: timestampnano leader plus at least one TOC entry
	// TODO: Make this 40 a const
	if cfg.PageSize < 40 {
		cfg.PageSize = 40
	}
	// The max is MaxUint32-1 because we use MaxUint32 to indicate push
	// replication local removal.
	if cfg.PageSize > math.MaxUint32-1 {
		cfg.PageSize = math.MaxUint32 - 1
	}
	// Ensure a full TOC page will have an associated data page of at least
	// checksumInterval in size, again so that each page written will at least
	// flush the previous page's data.
	// TODO: Make the 32 a const
	cfg.minValueAlloc = cfg.ChecksumInterval/(cfg.PageSize/32+1) + 1
	if env := os.Getenv("VALUESTORE_WRITE_PAGES_PER_WORKER"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			cfg.WritePagesPerWorker = val
		}
	}
	if cfg.WritePagesPerWorker == 0 {
		cfg.WritePagesPerWorker = 3
	}
	if cfg.WritePagesPerWorker < 2 {
		cfg.WritePagesPerWorker = 2
	}
	if env := os.Getenv("VALUESTORE_MSG_CAP"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			cfg.MsgCap = val
		}
	}
	if cfg.MsgCap == 0 {
		cfg.MsgCap = 16 * 1024 * 1024
	}
	// TODO: This minimum needs to be the max overhead.
	if cfg.MsgCap < 1 {
		cfg.MsgCap = 1
	}
	if env := os.Getenv("VALUESTORE_MSG_TIMEOUT"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			cfg.MsgTimeout = val
		}
	}
	if cfg.MsgTimeout == 0 {
		cfg.MsgTimeout = 100
	}
	if cfg.MsgTimeout < 1 {
		cfg.MsgTimeout = 100
	}
	if env := os.Getenv("VALUESTORE_VALUES_FILE_CAP"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			cfg.ValuesFileCap = val
		}
	}
	if cfg.ValuesFileCap == 0 {
		cfg.ValuesFileCap = math.MaxUint32
	}
	// TODO: Make the 40 and 8 consts
	if cfg.ValuesFileCap < 48+cfg.ValueCap { // header value trailer
		cfg.ValuesFileCap = 48 + cfg.ValueCap
	}
	if cfg.ValuesFileCap > math.MaxUint32 {
		cfg.ValuesFileCap = math.MaxUint32
	}
	if env := os.Getenv("VALUESTORE_VALUES_FILE_READERS"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			cfg.ValuesFileReaders = val
		}
	}
	if cfg.ValuesFileReaders == 0 {
		cfg.ValuesFileReaders = cfg.Workers
	}
	if cfg.ValuesFileReaders < 1 {
		cfg.ValuesFileReaders = 1
	}
	if env := os.Getenv("VALUESTORE_RECOVERY_BATCH_SIZE"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			cfg.RecoveryBatchSize = val
		}
	}
	if cfg.RecoveryBatchSize == 0 {
		cfg.RecoveryBatchSize = 1024 * 1024
	}
	if cfg.RecoveryBatchSize < 1 {
		cfg.RecoveryBatchSize = 1
	}
	if env := os.Getenv("VALUESTORE_TOMBSTONE_DISCARD_INTERVAL"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			cfg.TombstoneDiscardInterval = val
		}
	}
	if cfg.TombstoneDiscardInterval == 0 {
		cfg.TombstoneDiscardInterval = cfg.BackgroundInterval
	}
	if cfg.TombstoneDiscardInterval < 1 {
		cfg.TombstoneDiscardInterval = 1
	}
	if env := os.Getenv("VALUESTORE_TOMBSTONE_DISCARD_BATCH_SIZE"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			cfg.TombstoneDiscardBatchSize = val
		}
	}
	if cfg.TombstoneDiscardBatchSize == 0 {
		cfg.TombstoneDiscardBatchSize = 1024 * 1024
	}
	if cfg.TombstoneDiscardBatchSize < 1 {
		cfg.TombstoneDiscardBatchSize = 1
	}
	if env := os.Getenv("VALUESTORE_TOMBSTONE_AGE"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			cfg.TombstoneAge = val
		}
	}
	if cfg.TombstoneAge == 0 {
		cfg.TombstoneAge = 4 * 60 * 60
	}
	if cfg.TombstoneAge < 0 {
		cfg.TombstoneAge = 0
	}
	if env := os.Getenv("VALUESTORE_REPLICATION_IGNORE_RECENT"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			cfg.ReplicationIgnoreRecent = val
		}
	}
	if cfg.ReplicationIgnoreRecent == 0 {
		cfg.ReplicationIgnoreRecent = 60
	}
	if cfg.ReplicationIgnoreRecent < 0 {
		cfg.ReplicationIgnoreRecent = 0
	}
	if env := os.Getenv("VALUESTORE_OUT_PULL_REPLICATION_INTERVAL"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			cfg.OutPullReplicationInterval = val
		}
	}
	if cfg.OutPullReplicationInterval == 0 {
		cfg.OutPullReplicationInterval = cfg.BackgroundInterval
	}
	if cfg.OutPullReplicationInterval < 1 {
		cfg.OutPullReplicationInterval = 1
	}
	if env := os.Getenv("VALUESTORE_OUT_PULL_REPLICATION_WORKERS"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			cfg.OutPullReplicationWorkers = val
		}
	}
	if cfg.OutPullReplicationWorkers == 0 {
		cfg.OutPullReplicationWorkers = cfg.Workers
	}
	if cfg.OutPullReplicationWorkers < 1 {
		cfg.OutPullReplicationWorkers = 1
	}
	if env := os.Getenv("VALUESTORE_OUT_PULL_REPLICATION_MSGS"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			cfg.OutPullReplicationMsgs = val
		}
	}
	if cfg.OutPullReplicationMsgs == 0 {
		cfg.OutPullReplicationMsgs = cfg.OutPullReplicationWorkers * 4
	}
	if cfg.OutPullReplicationMsgs < 1 {
		cfg.OutPullReplicationMsgs = 1
	}
	if env := os.Getenv("VALUESTORE_OUT_PULL_REPLICATION_BLOOM_N"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			cfg.OutPullReplicationBloomN = val
		}
	}
	if cfg.OutPullReplicationBloomN == 0 {
		cfg.OutPullReplicationBloomN = 1000000
	}
	if cfg.OutPullReplicationBloomN < 1 {
		cfg.OutPullReplicationBloomN = 1
	}
	if env := os.Getenv("VALUESTORE_OUT_PULL_REPLICATION_BLOOM_P"); env != "" {
		if val, err := strconv.ParseFloat(env, 64); err == nil {
			cfg.OutPullReplicationBloomP = val
		}
	}
	if cfg.OutPullReplicationBloomP == 0.0 {
		cfg.OutPullReplicationBloomP = 0.001
	}
	if cfg.OutPullReplicationBloomP < 0.000001 {
		cfg.OutPullReplicationBloomP = 0.000001
	}
	if env := os.Getenv("VALUESTORE_OUT_PULL_REPLICATION_MSG_TIMEOUT"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			cfg.OutPullReplicationMsgTimeout = val
		}
	}
	if cfg.OutPullReplicationMsgTimeout == 0 {
		cfg.OutPullReplicationMsgTimeout = cfg.MsgTimeout
	}
	if cfg.OutPullReplicationMsgTimeout < 1 {
		cfg.OutPullReplicationMsgTimeout = 100
	}
	if env := os.Getenv("VALUESTORE_IN_PULL_REPLICATION_WORKERS"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			cfg.InPullReplicationWorkers = val
		}
	}
	if cfg.InPullReplicationWorkers == 0 {
		cfg.InPullReplicationWorkers = cfg.Workers
	}
	if cfg.InPullReplicationWorkers < 1 {
		cfg.InPullReplicationWorkers = 1
	}
	if env := os.Getenv("VALUESTORE_IN_PULL_REPLICATION_MSGS"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			cfg.InPullReplicationMsgs = val
		}
	}
	if cfg.InPullReplicationMsgs == 0 {
		cfg.InPullReplicationMsgs = cfg.InPullReplicationWorkers * 4
	}
	if cfg.InPullReplicationMsgs < 1 {
		cfg.InPullReplicationMsgs = 1
	}
	if env := os.Getenv("VALUESTORE_IN_PULL_REPLICATION_RESPONSE_MSG_TIMEOUT"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			cfg.InPullReplicationResponseMsgTimeout = val
		}
	}
	if cfg.InPullReplicationResponseMsgTimeout == 0 {
		cfg.InPullReplicationResponseMsgTimeout = cfg.MsgTimeout
	}
	if cfg.InPullReplicationResponseMsgTimeout < 1 {
		cfg.InPullReplicationResponseMsgTimeout = 100
	}
	if env := os.Getenv("VALUESTORE_OUT_PUSH_REPLICATION_INTERVAL"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			cfg.OutPushReplicationInterval = val
		}
	}
	if cfg.OutPushReplicationInterval == 0 {
		cfg.OutPushReplicationInterval = cfg.BackgroundInterval
	}
	if cfg.OutPushReplicationInterval < 1 {
		cfg.OutPushReplicationInterval = 1
	}
	if env := os.Getenv("VALUESTORE_OUT_PUSH_REPLICATION_WORKERS"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			cfg.OutPushReplicationWorkers = val
		}
	}
	if cfg.OutPushReplicationWorkers == 0 {
		cfg.OutPushReplicationWorkers = cfg.Workers
	}
	if cfg.OutPushReplicationWorkers < 1 {
		cfg.OutPushReplicationWorkers = 1
	}
	if env := os.Getenv("VALUESTORE_OUT_PUSH_REPLICATION_MSGS"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			cfg.OutPushReplicationMsgs = val
		}
	}
	if cfg.OutPushReplicationMsgs == 0 {
		cfg.OutPushReplicationMsgs = cfg.OutPushReplicationWorkers * 4
	}
	if cfg.OutPushReplicationMsgs < 1 {
		cfg.OutPushReplicationMsgs = 1
	}
	if env := os.Getenv("VALUESTORE_OUT_PUSH_REPLICATION_MSG_TIMEOUT"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			cfg.OutPushReplicationMsgTimeout = val
		}
	}
	if cfg.OutPushReplicationMsgTimeout == 0 {
		cfg.OutPushReplicationMsgTimeout = cfg.MsgTimeout
	}
	if cfg.OutPushReplicationMsgTimeout < 1 {
		cfg.OutPushReplicationMsgTimeout = 100
	}
	if env := os.Getenv("VALUESTORE_BULK_SET_MSG_CAP"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			cfg.BulkSetMsgCap = val
		}
	}
	if cfg.BulkSetMsgCap == 0 {
		cfg.BulkSetMsgCap = cfg.MsgCap
	}
	if cfg.BulkSetMsgCap < 1 {
		cfg.BulkSetMsgCap = 1
	}
	if env := os.Getenv("VALUESTORE_OUT_BULK_SET_MSGS"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			cfg.OutBulkSetMsgs = val
		}
	}
	if cfg.OutBulkSetMsgs == 0 {
		cfg.OutBulkSetMsgs = cfg.OutPushReplicationWorkers * 4
	}
	if cfg.OutBulkSetMsgs < 1 {
		cfg.OutBulkSetMsgs = 1
	}
	if env := os.Getenv("VALUESTORE_IN_BULK_SET_WORKERS"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			cfg.InBulkSetWorkers = val
		}
	}
	if cfg.InBulkSetWorkers == 0 {
		cfg.InBulkSetWorkers = cfg.Workers
	}
	if cfg.InBulkSetWorkers < 1 {
		cfg.InBulkSetWorkers = 1
	}
	if env := os.Getenv("VALUESTORE_IN_BULK_SET_MSGS"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			cfg.InBulkSetMsgs = val
		}
	}
	if cfg.InBulkSetMsgs == 0 {
		cfg.InBulkSetMsgs = cfg.InBulkSetWorkers * 4
	}
	if cfg.InBulkSetMsgs < 1 {
		cfg.InBulkSetMsgs = 1
	}
	if env := os.Getenv("VALUESTORE_IN_BULK_SET_RESPONSE_MSG_TIMEOUT"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			cfg.InBulkSetResponseMsgTimeout = val
		}
	}
	if cfg.InBulkSetResponseMsgTimeout == 0 {
		cfg.InBulkSetResponseMsgTimeout = cfg.MsgTimeout
	}
	if cfg.InBulkSetResponseMsgTimeout < 1 {
		cfg.InBulkSetResponseMsgTimeout = 100
	}
	if env := os.Getenv("VALUESTORE_OUT_BULK_SET_ACK_MSG_CAP"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			cfg.BulkSetAckMsgCap = val
		}
	}
	if cfg.BulkSetAckMsgCap == 0 {
		cfg.BulkSetAckMsgCap = cfg.MsgCap
	}
	if cfg.BulkSetAckMsgCap < 1 {
		cfg.BulkSetAckMsgCap = 1
	}
	if env := os.Getenv("VALUESTORE_IN_BULK_SET_ACK_WORKERS"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			cfg.InBulkSetAckWorkers = val
		}
	}
	if cfg.InBulkSetAckWorkers == 0 {
		cfg.InBulkSetAckWorkers = cfg.Workers
	}
	if cfg.InBulkSetAckWorkers < 1 {
		cfg.InBulkSetAckWorkers = 1
	}
	if env := os.Getenv("VALUESTORE_IN_BULK_SET_ACK_MSGS"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			cfg.InBulkSetAckMsgs = val
		}
	}
	if cfg.InBulkSetAckMsgs == 0 {
		cfg.InBulkSetAckMsgs = cfg.InBulkSetAckWorkers * 4
	}
	if cfg.InBulkSetAckMsgs < 1 {
		cfg.InBulkSetAckMsgs = 1
	}
	if env := os.Getenv("VALUESTORE_OUT_BULK_SET_ACK_MSGS"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			cfg.OutBulkSetAckMsgs = val
		}
	}
	if cfg.OutBulkSetAckMsgs == 0 {
		cfg.OutBulkSetAckMsgs = cfg.InBulkSetAckWorkers * 4
	}
	if cfg.OutBulkSetAckMsgs < 1 {
		cfg.OutBulkSetAckMsgs = 1
	}
	if env := os.Getenv("VALUESTORE_COMPACTION_INTERVAL"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			cfg.CompactionInterval = val
		}
	}
	if cfg.CompactionInterval == 0 {
		cfg.CompactionInterval = cfg.BackgroundInterval
	}
	if cfg.CompactionInterval < 1 {
		cfg.CompactionInterval = 1
	}
	if env := os.Getenv("VALUESTORE_COMPACTION_WORKERS"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			cfg.CompactionWorkers = val
		}
	}
	if cfg.CompactionWorkers == 0 {
		cfg.CompactionWorkers = cfg.Workers
	}
	if cfg.CompactionWorkers < 1 {
		cfg.CompactionWorkers = 1
	}
	if env := os.Getenv("VALUESTORE_COMPACTION_THRESHOLD"); env != "" {
		if val, err := strconv.ParseFloat(env, 64); err == nil {
			cfg.CompactionThreshold = val
		}
	}
	if cfg.CompactionThreshold == 0.0 {
		cfg.CompactionThreshold = 0.10
	}
	if cfg.CompactionThreshold >= 1.0 || cfg.CompactionThreshold <= 0.01 {
		cfg.CompactionThreshold = 0.10
	}
	if env := os.Getenv("VALUESTORE_COMPACTION_AGE_THRESHOLD"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			cfg.CompactionAgeThreshold = val
		}
	}
	if cfg.CompactionAgeThreshold == 0 {
		cfg.CompactionAgeThreshold = 300
	}
	if cfg.CompactionAgeThreshold < 1 {
		cfg.CompactionAgeThreshold = 1
	}
	return cfg
}
