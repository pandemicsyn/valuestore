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

type config struct {
	logCritical                 *log.Logger
	logError                    *log.Logger
	logWarning                  *log.Logger
	logInfo                     *log.Logger
	logDebug                    *log.Logger
	rand                        *rand.Rand
	path                        string
	pathtoc                     string
	vlm                         valuelocmap.ValueLocMap
	workers                     int
	backgroundInterval          int
	msgTimeout                  int
	msgCap                      int
	recoveryBatchSize           int
	tombstoneDiscardInterval    int
	tombstoneDiscardBatchSize   int
	inPullReplicationMsgs       int
	inPullReplicationWorkers    int
	inPullReplicationMsgTimeout int
	outPullReplicationWorkers   int
	outPullReplicationInterval  int
	outPullReplicationMsgs      int
	outPullReplicationBloomN    int
	outPullReplicationBloomP    float64
	outPushReplicationWorkers   int
	outPushReplicationInterval  int
	outPushReplicationMsgs      int
	outPushReplicationMsgCap    int
	valueCap                    int
	pageSize                    int
	minValueAlloc               int
	writePagesPerWorker         int
	tombstoneAge                int
	valuesFileCap               int
	valuesFileReaders           int
	checksumInterval            int
	msgRing                     ring.MsgRing
	replicationIgnoreRecent     int
	inBulkSetMsgs               int
	inBulkSetWorkers            int
	inBulkSetMsgTimeout         int
	outBulkSetMsgs              int
	outBulkSetMsgCap            int
	inBulkSetAckMsgs            int
	inBulkSetAckWorkers         int
	inBulkSetAckMsgTimeout      int
	outBulkSetAckMsgs           int
	outBulkSetAckMsgCap         int
	compactionInterval          int
	compactionThreshold         float64
	compactionAgeThreshold      int
	compactionWorkers           int
}

func resolveConfig(opts ...func(*config)) *config {
	cfg := &config{}
	cfg.path = os.Getenv("VALUESTORE_PATH")
	cfg.pathtoc = os.Getenv("VALUESTORE_PATHTOC")
	cfg.workers = runtime.GOMAXPROCS(0)
	if env := os.Getenv("VALUESTORE_WORKERS"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			cfg.workers = val
		}
	}
	cfg.backgroundInterval = 60
	if env := os.Getenv("VALUESTORE_BACKGROUNDINTERVAL"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			cfg.backgroundInterval = val
		}
	}
	cfg.msgTimeout = 300
	if env := os.Getenv("VALUESTORE_MSGTIMEOUT"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			cfg.msgTimeout = val
		}
	}
	cfg.msgCap = 16 * 1024 * 1024
	if env := os.Getenv("VALUESTORE_MSGCAP"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			cfg.msgCap = val
		}
	}
	cfg.recoveryBatchSize = 1024 * 1024
	if env := os.Getenv("VALUESTORE_RECOVERYBATCHSIZE"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			cfg.recoveryBatchSize = val
		}
	}
	cfg.tombstoneDiscardInterval = cfg.backgroundInterval
	if env := os.Getenv("VALUESTORE_TOMBSTONEDISCARDINTERVAL"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			cfg.tombstoneDiscardInterval = val
		}
	}
	cfg.tombstoneDiscardBatchSize = 1024 * 1024
	if env := os.Getenv("VALUESTORE_TOMBSTONEDISCARDBATCHSIZE"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			cfg.tombstoneDiscardBatchSize = val
		}
	}
	cfg.inPullReplicationMsgs = 128
	if env := os.Getenv("VALUESTORE_INPULLREPLICATIONMSGS"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			cfg.inPullReplicationMsgs = val
		}
	}
	cfg.inPullReplicationWorkers = 40
	if env := os.Getenv("VALUESTORE_INPULLREPLICATIONWORKERS"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			cfg.inPullReplicationWorkers = val
		}
	}
	cfg.inPullReplicationMsgTimeout = cfg.msgTimeout
	if env := os.Getenv("VALUESTORE_INPULLREPLICATIONMSGTIMEOUT"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			cfg.inPullReplicationMsgTimeout = val
		}
	}
	cfg.outPullReplicationWorkers = cfg.workers
	if env := os.Getenv("VALUESTORE_OUTPULLREPLICATIONWORKERS"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			cfg.outPullReplicationWorkers = val
		}
	}
	cfg.outPullReplicationInterval = cfg.backgroundInterval
	if env := os.Getenv("VALUESTORE_OUTPULLREPLICATIONINTERVAL"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			cfg.outPullReplicationInterval = val
		}
	}
	cfg.outPullReplicationMsgs = 128
	if env := os.Getenv("VALUESTORE_OUTPULLREPLICATIONMSGS"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			cfg.outPullReplicationMsgs = val
		}
	}
	cfg.outPullReplicationBloomN = 1000000
	if env := os.Getenv("VALUESTORE_OUTPULLREPLICATIONBLOOMN"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			cfg.outPullReplicationBloomN = val
		}
	}
	cfg.outPullReplicationBloomP = 0.001
	if env := os.Getenv("VALUESTORE_OUTPULLREPLICATIONBLOOMP"); env != "" {
		if val, err := strconv.ParseFloat(env, 64); err == nil {
			cfg.outPullReplicationBloomP = val
		}
	}
	cfg.outPushReplicationWorkers = cfg.workers
	if env := os.Getenv("VALUESTORE_OUTPUSHREPLICATIONWORKERS"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			cfg.outPushReplicationWorkers = val
		}
	}
	cfg.outPushReplicationInterval = cfg.backgroundInterval
	if env := os.Getenv("VALUESTORE_OUTPUSHREPLICATIONINTERVAL"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			cfg.outPushReplicationInterval = val
		}
	}
	cfg.outPushReplicationMsgs = 128
	if env := os.Getenv("VALUESTORE_OUTPUSHREPLICATIONMSGS"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			cfg.outPushReplicationMsgs = val
		}
	}
	cfg.valueCap = 4 * 1024 * 1024
	if env := os.Getenv("VALUESTORE_VALUECAP"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			cfg.valueCap = val
		}
	}
	cfg.pageSize = 4 * 1024 * 1024
	if env := os.Getenv("VALUESTORE_PAGESIZE"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			cfg.pageSize = val
		}
	}
	cfg.writePagesPerWorker = 3
	if env := os.Getenv("VALUESTORE_WRITEPAGESPERWORKER"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			cfg.writePagesPerWorker = val
		}
	}
	cfg.tombstoneAge = 4 * 60 * 60
	if env := os.Getenv("VALUESTORE_TOMBSTONEAGE"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			cfg.tombstoneAge = val
		}
	}
	cfg.valuesFileCap = math.MaxUint32
	if env := os.Getenv("VALUESTORE_VALUESFILECAP"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			cfg.valuesFileCap = val
		}
	}
	cfg.valuesFileReaders = cfg.workers
	if env := os.Getenv("VALUESTORE_VALUESFILEREADERS"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			cfg.valuesFileReaders = val
		}
	}
	cfg.checksumInterval = 65532
	if env := os.Getenv("VALUESTORE_CHECKSUMINTERVAL"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			cfg.checksumInterval = val
		}
	}
	cfg.replicationIgnoreRecent = 60
	if env := os.Getenv("VALUESTORE_REPLICATIONIGNORERECENT"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			cfg.replicationIgnoreRecent = val
		}
	}
	cfg.inBulkSetMsgs = 128
	if env := os.Getenv("VALUESTORE_INBULKSETMSGS"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			cfg.inBulkSetMsgs = val
		}
	}
	cfg.inBulkSetWorkers = 40
	if env := os.Getenv("VALUESTORE_INBULKSETWORKERS"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			cfg.inBulkSetWorkers = val
		}
	}
	cfg.inBulkSetMsgTimeout = cfg.msgTimeout
	if env := os.Getenv("VALUESTORE_INBULKSETMSGTIMEOUT"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			cfg.inBulkSetMsgTimeout = val
		}
	}
	cfg.outBulkSetMsgs = 128
	if env := os.Getenv("VALUESTORE_OUTBULKSETMSGS"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			cfg.outBulkSetMsgs = val
		}
	}
	cfg.outBulkSetMsgCap = cfg.msgCap
	if env := os.Getenv("VALUESTORE_OUTBULKSETMSGCAP"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			cfg.outBulkSetMsgCap = val
		}
	}
	cfg.inBulkSetAckMsgs = 128
	if env := os.Getenv("VALUESTORE_INBULKSETACKMSGS"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			cfg.inBulkSetAckMsgs = val
		}
	}
	cfg.inBulkSetAckWorkers = 40
	if env := os.Getenv("VALUESTORE_INBULKSETACKWORKERS"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			cfg.inBulkSetAckWorkers = val
		}
	}
	cfg.inBulkSetAckMsgTimeout = cfg.msgTimeout
	if env := os.Getenv("VALUESTORE_INBULKSETACKMSGTIMEOUT"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			cfg.inBulkSetAckMsgTimeout = val
		}
	}
	cfg.outBulkSetAckMsgs = 128
	if env := os.Getenv("VALUESTORE_OUTBULKSETACKMSGS"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			cfg.outBulkSetAckMsgs = val
		}
	}
	cfg.outBulkSetAckMsgCap = cfg.msgCap
	if env := os.Getenv("VALUESTORE_OUTBULKSETACKMSGCAP"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			cfg.outBulkSetAckMsgCap = val
		}
	}
	cfg.compactionInterval = cfg.backgroundInterval
	if env := os.Getenv("VALUESTORE_COMPACTIONINTERVAL"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			cfg.compactionInterval = val
		}
	}
	cfg.compactionThreshold = 0.10
	if env := os.Getenv("VALUESTORE_COMPACTIONTHRESHOLD"); env != "" {
		if val, err := strconv.ParseFloat(env, 64); err == nil {
			cfg.compactionThreshold = val
		}
	}
	cfg.compactionAgeThreshold = 300
	if env := os.Getenv("VALUESTORE_COMPACTIONAGETHRESHOLD"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			cfg.compactionAgeThreshold = val
		}
	}
	cfg.compactionWorkers = 1
	if env := os.Getenv("VALUESTORE_COMPACTIONWORKERS"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			cfg.compactionWorkers = val
		}
	}
	for _, opt := range opts {
		opt(cfg)
	}
	if cfg.logCritical == nil {
		cfg.logCritical = log.New(os.Stderr, "ValueStore ", log.LstdFlags)
	}
	if cfg.logError == nil {
		cfg.logError = log.New(os.Stderr, "ValueStore ", log.LstdFlags)
	}
	if cfg.logWarning == nil {
		cfg.logWarning = log.New(os.Stderr, "ValueStore ", log.LstdFlags)
	}
	if cfg.logInfo == nil {
		cfg.logInfo = log.New(os.Stdout, "ValueStore ", log.LstdFlags)
	}
	if cfg.rand == nil {
		cfg.rand = rand.New(rand.NewSource(time.Now().UnixNano()))
	}
	if cfg.path == "" {
		cfg.path = "."
	}
	if cfg.pathtoc == "" {
		cfg.pathtoc = cfg.path
	}
	if cfg.workers < 1 {
		cfg.workers = 1
	}
	if cfg.backgroundInterval < 1 {
		cfg.backgroundInterval = 1
	}
	if cfg.msgTimeout < 1 {
		cfg.msgTimeout = 1
	}
	// TODO: This minimum needs to be the message cap plus max overhead.
	if cfg.msgCap < 1 {
		cfg.msgCap = 1
	}
	if cfg.recoveryBatchSize < 1 {
		cfg.recoveryBatchSize = 1
	}
	if cfg.tombstoneDiscardInterval < 1 {
		cfg.tombstoneDiscardInterval = 1
	}
	if cfg.tombstoneDiscardBatchSize < 1 {
		cfg.tombstoneDiscardBatchSize = 1
	}
	if cfg.inPullReplicationMsgs < 1 {
		cfg.inPullReplicationMsgs = 1
	}
	if cfg.inPullReplicationWorkers < 1 {
		cfg.inPullReplicationWorkers = 1
	}
	if cfg.inPullReplicationMsgTimeout < 1 {
		cfg.inPullReplicationMsgTimeout = 1
	}
	if cfg.outPullReplicationWorkers < 1 {
		cfg.outPullReplicationWorkers = 1
	}
	if cfg.outPullReplicationInterval < 1 {
		cfg.outPullReplicationInterval = 1
	}
	if cfg.outPullReplicationMsgs < 1 {
		cfg.outPullReplicationMsgs = 1
	}
	if cfg.outPullReplicationBloomN < 1 {
		cfg.outPullReplicationBloomN = 1
	}
	if cfg.outPullReplicationBloomP < 0.000001 {
		cfg.outPullReplicationBloomP = 0.000001
	}
	if cfg.outPushReplicationWorkers < 1 {
		cfg.outPushReplicationWorkers = 1
	}
	if cfg.outPushReplicationInterval < 1 {
		cfg.outPushReplicationInterval = 1
	}
	if cfg.outPushReplicationMsgs < 1 {
		cfg.outPushReplicationMsgs = 1
	}
	if cfg.valueCap < 0 {
		cfg.valueCap = 0
	}
	if cfg.valueCap > math.MaxUint32 {
		cfg.valueCap = math.MaxUint32
	}
	if cfg.checksumInterval < 1 {
		cfg.checksumInterval = 1
	}
	// Ensure each page will have at least checksumInterval worth of data in it
	// so that each page written will at least flush the previous page's data.
	if cfg.pageSize < cfg.valueCap+cfg.checksumInterval {
		cfg.pageSize = cfg.valueCap + cfg.checksumInterval
	}
	// Absolute minimum: timestampnano leader plus at least one TOC entry
	if cfg.pageSize < 40 {
		cfg.pageSize = 40
	}
	// The max is MaxUint32-1 because we use MaxUint32 to indicate push
	// replication local removal.
	if cfg.pageSize > math.MaxUint32-1 {
		cfg.pageSize = math.MaxUint32 - 1
	}
	// Ensure a full TOC page will have an associated data page of at least
	// checksumInterval in size, again so that each page written will at least
	// flush the previous page's data.
	cfg.minValueAlloc = cfg.checksumInterval/(cfg.pageSize/32+1) + 1
	if cfg.writePagesPerWorker < 2 {
		cfg.writePagesPerWorker = 2
	}
	if cfg.tombstoneAge < 0 {
		cfg.tombstoneAge = 0
	}
	if cfg.valuesFileCap < 48+cfg.valueCap { // header value trailer
		cfg.valuesFileCap = 48 + cfg.valueCap
	}
	if cfg.valuesFileCap > math.MaxUint32 {
		cfg.valuesFileCap = math.MaxUint32
	}
	if cfg.valuesFileReaders < 1 {
		cfg.valuesFileReaders = 1
	}
	if cfg.replicationIgnoreRecent < 0 {
		cfg.replicationIgnoreRecent = 0
	}
	if cfg.inBulkSetMsgs < 1 {
		cfg.inBulkSetMsgs = 1
	}
	if cfg.inBulkSetWorkers < 1 {
		cfg.inBulkSetWorkers = 1
	}
	if cfg.inBulkSetMsgTimeout < 1 {
		cfg.inBulkSetMsgTimeout = 1
	}
	if cfg.outBulkSetMsgs < 1 {
		cfg.outBulkSetMsgs = 1
	}
	if cfg.outBulkSetMsgCap < 1 {
		cfg.outBulkSetMsgCap = 1
	}
	if cfg.inBulkSetAckMsgs < 1 {
		cfg.inBulkSetAckMsgs = 1
	}
	if cfg.inBulkSetAckWorkers < 1 {
		cfg.inBulkSetAckWorkers = 1
	}
	if cfg.inBulkSetAckMsgTimeout < 1 {
		cfg.inBulkSetAckMsgTimeout = 1
	}
	if cfg.outBulkSetAckMsgs < 1 {
		cfg.outBulkSetAckMsgs = 1
	}
	if cfg.outBulkSetAckMsgCap < 1 {
		cfg.outBulkSetAckMsgCap = 1
	}
	if cfg.compactionInterval < 1 {
		cfg.compactionInterval = 1
	}
	if cfg.compactionWorkers < 1 {
		cfg.compactionWorkers = 1
	}
	if cfg.compactionThreshold >= 1.0 || cfg.compactionThreshold <= 0.01 {
		cfg.compactionThreshold = 0.10
	}
	if cfg.compactionAgeThreshold < 1 {
		cfg.compactionAgeThreshold = 1
	}
	return cfg
}

// OptList returns a slice with the opts given; useful if you want to possibly
// append more options to the list before using it with New(list...).
func OptList(opts ...func(*config)) []func(*config) {
	return opts
}

// OptLogCritical sets the log.Logger to use for critical messages. Defaults
// logging to os.Stderr.
func OptLogCritical(l *log.Logger) func(*config) {
	return func(cfg *config) {
		cfg.logCritical = l
	}
}

// OptLogError sets the log.Logger to use for error messages. Defaults logging
// to os.Stderr.
func OptLogError(l *log.Logger) func(*config) {
	return func(cfg *config) {
		cfg.logError = l
	}
}

// OptLogWarning sets the log.Logger to use for warning messages. Defaults
// logging to os.Stderr.
func OptLogWarning(l *log.Logger) func(*config) {
	return func(cfg *config) {
		cfg.logWarning = l
	}
}

// OptLogInfo sets the log.Logger to use for info messages. Defaults logging to
// os.Stdout.
func OptLogInfo(l *log.Logger) func(*config) {
	return func(cfg *config) {
		cfg.logInfo = l
	}
}

// OptLogDebug sets the log.Logger to use for debug messages. Defaults not
// logging debug messages.
func OptLogDebug(l *log.Logger) func(*config) {
	return func(cfg *config) {
		cfg.logDebug = l
	}
}

// OptRand sets the rand.Rand to use as a random data source. Defaults to a new
// randomizer based on the current time.
func OptRand(r *rand.Rand) func(*config) {
	return func(cfg *config) {
		cfg.rand = r
	}
}

// OptPath sets the path where values files will be written; tocvalues files
// will also be written here unless overridden with OptPathTOC. Defaults to env
// VALUESTORE_PATH or the current working directory.
func OptPath(dirpath string) func(*config) {
	return func(cfg *config) {
		cfg.path = dirpath
	}
}

// OptPathTOC sets the path where tocvalues files will be written. Defaults to
// env VALUESTORE_PATHTOC or the OptPath value.
func OptPathTOC(dirpath string) func(*config) {
	return func(cfg *config) {
		cfg.pathtoc = dirpath
	}
}

// OptValueLocMap allows overriding the default ValueLocMap, an interface used
// by ValueStore for tracking the mappings from keys to the locations of their
// values. Defaults to github.com/gholt/valuelocmap.New().
func OptValueLocMap(vlm valuelocmap.ValueLocMap) func(*config) {
	return func(cfg *config) {
		cfg.vlm = vlm
	}
}

// OptWorkers indicates how many goroutines may be used for various tasks
// (processing incoming writes and batching them to disk, background tasks,
// etc.). Defaults to env VALUESTORE_WORKERS or GOMAXPROCS.
func OptWorkers(count int) func(*config) {
	return func(cfg *config) {
		cfg.workers = count
	}
}

// OptBackgroundInterval indicates the minimum number of seconds between the
// starts of background passes (such as discarding expired tombstones [deletion
// markers]). If set to 60 seconds and the passes take 10 seconds to run, they
// will wait 50 seconds (with a small amount of randomization) between the stop
// of one run and the start of the next. This is really just meant to keep
// nearly empty structures from using a lot of resources doing nearly nothing.
// Normally, you'd want your background passes to be running constantly so that
// they are as fast as possible and the load constant. The default of 60
// seconds is almost always fine. Defaults to env VALUESTORE_BACKGROUNDINTERVAL
// or 60.
func OptBackgroundInterval(seconds int) func(*config) {
	return func(cfg *config) {
		cfg.backgroundInterval = seconds
	}
}

// OptMsgTimeout indicates the maximum seconds an incoming message can be
// pending before just discarding it. Defaults to env VALUESTORE_MSGTIMEOUT
// or 300.
func OptMsgTimeout(seconds int) func(*config) {
	return func(cfg *config) {
		cfg.msgTimeout = seconds
	}
}

// OptMsgCap indicates the maximum bytes for outgoing messages. Defaults
// to env VALUESTORE_MSGCAP or 16,777,216.
func OptMsgCap(bytes int) func(*config) {
	return func(cfg *config) {
		cfg.msgCap = bytes
	}
}

// OptRecoveryBatchSize indicates how many keys to set in a batch while
// performing recovery (initial start up). Defaults to env
// VALUESTORE_RECOVERYBATCHSIZE or 1,048,576.
func OptRecoveryBatchSize(count int) func(*config) {
	return func(cfg *config) {
		cfg.recoveryBatchSize = count
	}
}

// OptTombstoneDiscardInterval indicates the minimum number of seconds between
// the starts of discard passes (discarding expired tombstones [deletion
// markers]). If set to 60 seconds and the passes take 10 seconds to run, they
// will wait 50 seconds (with a small amount of randomization) between the stop
// of one run and the start of the next. This is really just meant to keep
// nearly empty structures from using a lot of resources doing nearly nothing.
// Normally, you'd want your discard passes to be running constantly so that
// they are as fast as possible and the load constant. The default of 60
// seconds is almost always fine. Defaults to env
// VALUESTORE_TOMBSTONEDISCARDINTERVAL, VALUESTORE_BACKGROUNDINTERVAL, or 60.
func OptTombstoneDiscardInterval(seconds int) func(*config) {
	return func(cfg *config) {
		cfg.tombstoneDiscardInterval = seconds
	}
}

// OptTombstoneDiscardBatchSize indicates how many items to queue up before
// pausing a scan, issuing the actual discards, and resuming the scan again.
// Defaults to env VALUESTORE_TOMBSTONEDISCARDBATCHSIZE or 1,048,576.
func OptTombstoneDiscardBatchSize(count int) func(*config) {
	return func(cfg *config) {
		cfg.tombstoneDiscardBatchSize = count
	}
}

// OptInPullReplicationMsgs indicates how many incoming pull-replication
// messages can be buffered before dropping additional ones. Defaults to env
// VALUESTORE_INPULLREPLICATIONMSGS or 128.
func OptInPullReplicationMsgs(count int) func(*config) {
	return func(cfg *config) {
		cfg.inPullReplicationMsgs = count
	}
}

// OptInPullReplicationWorkers indicates how many incoming pull-replication
// messages can be processed at the same time. Defaults to env
// VALUESTORE_INPULLREPLICATIONWORKERS or 40.
func OptInPullReplicationWorkers(count int) func(*config) {
	return func(cfg *config) {
		cfg.inPullReplicationWorkers = count
	}
}

// OptInPullReplicationMsgTimeout indicates the maximum seconds an incoming
// pull-replication message can be pending before just discarding it. Defaults
// to env VALUESTORE_INPULLREPLICATIONMSGTIMEOUT, VALUESTORE_MSGTIMEOUT, or
// 300.
func OptInPullReplicationMsgTimeout(seconds int) func(*config) {
	return func(cfg *config) {
		cfg.inPullReplicationMsgTimeout = seconds
	}
}

// OptOutPullReplicationWorkers indicates how many goroutines may be used for
// an outgoing pull replication pass. Defaults to env
// VALUESTORE_OUTPULLREPLICATIONWORKERS or VALUESTORE_WORKERS.
func OptOutPullReplicationWorkers(workers int) func(*config) {
	return func(cfg *config) {
		cfg.outPullReplicationWorkers = workers
	}
}

// OptOutPullReplicationInterval indicates the minimum number of seconds
// between the starts of outgoing pull replication passes. If set to 60 seconds
// and the passes take 10 seconds to run, they will wait 50 seconds (with a
// small amount of randomization) between the stop of one run and the start of
// the next. This is really just meant to keep nearly empty structures from
// using a lot of resources doing nearly nothing. Normally, you'd want your
// outgoing pull replication passes to be running constantly so that
// replication is as fast as possible and the load constant. The default of 60
// seconds is almost always fine. Defaults to env
// VALUESTORE_OUTPULLREPLICATIONINTERVAL, VALUESTORE_BACKGROUNDINTERVAL, or 60.
func OptOutPullReplicationInterval(seconds int) func(*config) {
	return func(cfg *config) {
		cfg.outPullReplicationInterval = seconds
	}
}

// OptOutPullReplicationMsgs indicates how many outgoing pull-replication
// messages can be buffered before blocking on creating more. Defaults to env
// VALUESTORE_OUTPULLREPLICATIONMSGS or 128.
func OptOutPullReplicationMsgs(count int) func(*config) {
	return func(cfg *config) {
		cfg.outPullReplicationMsgs = count
	}
}

// OptOutPullReplicationBloomN indicates the N-factor for the outgoing
// pull-replication bloom filters. This indicates how many keys the bloom
// filter can reasonably hold and, in combination with the P-factor, affects
// memory usage. Defaults to env VALUESTORE_OUTPULLREPLICATIONBLOOMN or
// 1000000.
func OptOutPullReplicationBloomN(count int) func(*config) {
	return func(cfg *config) {
		cfg.outPullReplicationBloomN = count
	}
}

// OptOutPullReplicationBloomP indicates the P-factor for the outgoing
// pull-replication bloom filters. This indicates the desired percentage chance
// of a collision within the bloom filter and, in combination with the
// N-factor, affects memory usage. Defaults to env
// VALUESTORE_OUTPULLREPLICATIONBLOOMP or 0.001.
func OptOutPullReplicationBloomP(percentage float64) func(*config) {
	return func(cfg *config) {
		cfg.outPullReplicationBloomP = percentage
	}
}

// OptOutPushReplicationWorkers indicates how many goroutines may be used for
// an outgoing push replication pass. Defaults to env
// VALUESTORE_OUTPUSHREPLICATIONWORKERS or VALUESTORE_WORKERS.
func OptOutPushReplicationWorkers(workers int) func(*config) {
	return func(cfg *config) {
		cfg.outPushReplicationWorkers = workers
	}
}

// OptOutPushReplicationInterval indicates the minimum number of seconds
// between the starts of outgoing push replication passes. If set to 60 seconds
// and the passes take 10 seconds to run, they will wait 50 seconds (with a
// small amount of randomization) between the stop of one run and the start of
// the next. This is really just meant to keep nearly empty structures from
// using a lot of resources doing nearly nothing. Normally, you'd want your
// outgoing push replication passes to be running constantly so that
// replication is as fast as possible and the load constant. The default of 60
// seconds is almost always fine. Defaults to env
// VALUESTORE_OUTPUSHREPLICATIONINTERVAL, VALUESTORE_BACKGROUNDINTERVAL, or 60.
func OptOutPushReplicationInterval(seconds int) func(*config) {
	return func(cfg *config) {
		cfg.outPushReplicationInterval = seconds
	}
}

// OptOutPushReplicationMsgs indicates how many outgoing push-replication
// messages can be buffered before blocking on creating more. Defaults to env
// VALUESTORE_OUTPUSHREPLICATIONMSGS or 128.
func OptOutPushReplicationMsgs(count int) func(*config) {
	return func(cfg *config) {
		cfg.outPushReplicationMsgs = count
	}
}

// OptValueCap indicates the maximum number of bytes any given value may
// be. Defaults to env VALUESTORE_VALUECAP or 4,194,304.
func OptValueCap(bytes int) func(*config) {
	return func(cfg *config) {
		cfg.valueCap = bytes
	}
}

// OptPageSize controls the size of each chunk of memory allocated. Defaults to
// env VALUESTORE_PAGESIZE or 4,194,304.
func OptPageSize(bytes int) func(*config) {
	return func(cfg *config) {
		cfg.pageSize = bytes
	}
}

// OptWritePagesPerWorker controls how many pages are created per worker for
// caching recently written values. Defaults to env
// VALUESTORE_WRITEPAGESPERWORKER or 3.
func OptWritePagesPerWorker(number int) func(*config) {
	return func(cfg *config) {
		cfg.writePagesPerWorker = number
	}
}

// OptTombstoneAge indicates how many seconds old a deletion marker may be
// before it is permanently removed. Defaults to env VALUESTORE_TOMBSTONEAGE or
// 14,400 (4 hours).
func OptTombstoneAge(seconds int) func(*config) {
	return func(cfg *config) {
		cfg.tombstoneAge = seconds
	}
}

// OptValuesFileCap indicates how large a values file can be before closing it
// and opening a new one. Defaults to env VALUESTORE_VALUESFILECAP or
// 4,294,967,295.
func OptValuesFileCap(bytes int) func(*config) {
	return func(cfg *config) {
		cfg.valuesFileCap = bytes
	}
}

// OptValuesFileReaders indicates how many open file descriptors are allowed
// per values file for reading. Defaults to env VALUESTORE_VALUESFILEREADERS or
// the configured number of workers.
func OptValuesFileReaders(bytes int) func(*config) {
	return func(cfg *config) {
		cfg.valuesFileReaders = bytes
	}
}

// OptChecksumInterval indicates how many bytes are output to a file before a
// 4-byte checksum is also output. Defaults to env VALUESTORE_CHECKSUMINTERVAL
// or 65532.
func OptChecksumInterval(bytes int) func(*config) {
	return func(cfg *config) {
		cfg.checksumInterval = bytes
	}
}

// OptMsgRing sets the ring.MsgRing to use for determining the key ranges the
// ValueStore is responsible for as well as providing methods to send messages
// to other nodes.
func OptMsgRing(r ring.MsgRing) func(*config) {
	return func(cfg *config) {
		cfg.msgRing = r
	}
}

// OptReplicationIgnoreRecent indicates how many seconds old a value should be
// before it is included in replication processing. Defaults to env
// VALUESTORE_REPLICATIONIGNORERECENT or 60.
func OptReplicationIgnoreRecent(seconds int) func(*config) {
	return func(cfg *config) {
		cfg.replicationIgnoreRecent = seconds
	}
}

// OptInBulkSetMsgs indicates how many incoming bulk-set messages can be
// buffered before dropping additional ones. Defaults to env
// VALUESTORE_INBULKSETMSGS or 128.
func OptInBulkSetMsgs(count int) func(*config) {
	return func(cfg *config) {
		cfg.inBulkSetMsgs = count
	}
}

// OptInBulkSetWorkers indicates how many incoming bulk-set messages can be
// processed at the same time. Defaults to env VALUESTORE_INBULKSETWORKERS or
// 40.
func OptInBulkSetWorkers(count int) func(*config) {
	return func(cfg *config) {
		cfg.inBulkSetWorkers = count
	}
}

// OptInBulkSetMsgTimeout indicates the maximum seconds an incoming bulk-set
// message can be pending before just discarding it. Defaults to env
// VALUESTORE_INBULKSETMSGTIMEOUT, VALUESTORE_MSGTIMEOUT, or 300.
func OptInBulkSetMsgTimeout(seconds int) func(*config) {
	return func(cfg *config) {
		cfg.inBulkSetMsgTimeout = seconds
	}
}

// OptOutBulkSetMsgs indicates how many outgoing bulk-set messages can
// be buffered before blocking on creating more. Defaults to env
// VALUESTORE_OUTBULKSETMSGS or 128.
func OptOutBulkSetMsgs(count int) func(*config) {
	return func(cfg *config) {
		cfg.outBulkSetMsgs = count
	}
}

// OptOutBulkSetMsgCap indicates the maximum bytes for outgoing bulk-set
// messages. Defaults to env VALUESTORE_OUTBULKSETMSGCAP,
// VALUESTORE_MSGCAP, or 16,777,216.
func OptOutBulkSetMsgCap(bytes int) func(*config) {
	return func(cfg *config) {
		cfg.outBulkSetMsgCap = bytes
	}
}

// OptInBulkSetAckMsgs indicates how many incoming bulk-set-ack messages can be
// buffered before dropping additional ones. Defaults to env
// VALUESTORE_INBULKSETACKMSGS or 128.
func OptInBulkSetAckMsgs(count int) func(*config) {
	return func(cfg *config) {
		cfg.inBulkSetAckMsgs = count
	}
}

// OptInBulkSetAckWorkers indicates how many incoming bulk-set-ack messages
// can be processed at the same time. Defaults to env
// VALUESTORE_INBULKSETACKWORKERS or 40.
func OptInBulkSetAckWorkers(count int) func(*config) {
	return func(cfg *config) {
		cfg.inBulkSetAckWorkers = count
	}
}

// OptInBulkSetAckMsgTimeout indicates the maximum seconds an incoming
// bulk-set-ack message can be pending before just discarding it. Defaults to
// env VALUESTORE_INBULKSETACKMSGTIMEOUT, VALUESTORE_MSGTIMEOUT, or 300.
func OptInBulkSetAckMsgTimeout(seconds int) func(*config) {
	return func(cfg *config) {
		cfg.inBulkSetAckMsgTimeout = seconds
	}
}

// OptOutBulkSetAckMsgs indicates how many outgoing bulk-set-ack messages can
// be buffered before blocking on creating more. Defaults to env
// VALUESTORE_OUTBULKSETACKMSGS or 128.
func OptOutBulkSetAckMsgs(count int) func(*config) {
	return func(cfg *config) {
		cfg.outBulkSetAckMsgs = count
	}
}

// OptOutBulkSetAckMsgCap indicates the maximum bytes for outgoing
// bulk-set-ack messages. Defaults to env VALUESTORE_OUTBULKSETACKMSGCAP,
// VALUESTORE_MSGCAP, or 16,777,216.
func OptOutBulkSetAckMsgMsgCap(bytes int) func(*config) {
	return func(cfg *config) {
		cfg.outBulkSetAckMsgCap = bytes
	}
}

// OptCompactionInterval indicates the minimum number of seconds between the
// starts of compaction passes. If set to 60 seconds and the passes take 10
// seconds to run, they will wait 50 seconds (with a small amount of
// randomization) between the stop of one run and the start of the next. This
// is really just meant to keep nearly empty structures from using a lot of
// resources doing nearly nothing. Normally, you'd want your compaction passes
// to be running constantly so that it is as fast as possible and the load
// constant. The default of 60 seconds is almost always fine. Defaults to env
// VALUESTORE_OUTPUSHREPLICATIONINTERVAL, VALUESTORE_BACKGROUNDINTERVAL, or 60.
func OptCompactionInterval(seconds int) func(*config) {
	return func(cfg *config) {
		cfg.compactionInterval = seconds
	}
}

// OptCompactionThreshold indicates how much waste a given file may have before
// it is compacted. Defaults to VALUESTORE_COMPACTIONTHRESHOLD or 0.10 (10%).
func OptCompactionThreshold(ratio float64) func(*config) {
	return func(cfg *config) {
		cfg.compactionThreshold = ratio
	}
}

// OptCompactionAgeThreshold indicates how old a given file must be before it
// is considered for compaction. Defaults to VALUESTORE_COMPACTIONAGETHRESHOLD
// or 300.
func OptCompactionAgeThreshold(seconds int) func(*config) {
	return func(cfg *config) {
		cfg.compactionAgeThreshold = seconds
	}
}

// OptCompactionWorkers indicates how much concurrency is allowed for
// compaction. Defaults to VALUESTORE_COMPACTIONWORKERS or 1.
func OptCompactionWorkers(count int) func(*config) {
	return func(cfg *config) {
		cfg.compactionWorkers = count
	}
}
