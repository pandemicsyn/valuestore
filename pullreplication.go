package valuestore

import (
	"encoding/binary"
	"io"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gholt/brimtime"
	"github.com/gholt/experimental-ring"
)

const _GLH_IN_PULL_REPLICATION_MSGS = 128
const _GLH_IN_PULL_REPLICATION_HANDLERS = 40
const _GLH_IN_PULL_REPLICATION_MSG_TIMEOUT = 300
const _GLH_OUT_PULL_REPLICATION_MSGS = 128
const _GLH_BLOOM_FILTER_N = 1000000
const _GLH_BLOOM_FILTER_P = 0.001
const pullReplicationMsgHeaderBytes = 44

type pullReplicationState struct {
	inMsgChan     chan *pullReplicationMsg
	inFreeMsgChan chan *pullReplicationMsg
	outWorkers    int
	outInterval   int
	outNotifyChan chan *backgroundNotification
	outIteration  uint16
	outAbort      uint32
	outMsgChan    chan *pullReplicationMsg
	outKTBFs      []*ktBloomFilter
}

type pullReplicationMsg struct {
	vs     *DefaultValueStore
	header []byte
	body   []byte
}

func (vs *DefaultValueStore) pullReplicationInit(cfg *config) {
	vs.pullReplicationState.outInterval = cfg.outPullReplicationInterval
	vs.pullReplicationState.outNotifyChan = make(chan *backgroundNotification, 1)
	vs.pullReplicationState.outWorkers = cfg.outPullReplicationWorkers
	vs.pullReplicationState.outIteration = uint16(cfg.rand.Uint32())
	if vs.ring != nil {
		vs.ring.SetMsgHandler(ring.MSG_PULL_REPLICATION, vs.newInPullReplicationMsg)
		vs.pullReplicationState.inMsgChan = make(chan *pullReplicationMsg, _GLH_IN_PULL_REPLICATION_MSGS)
		vs.pullReplicationState.inFreeMsgChan = make(chan *pullReplicationMsg, _GLH_IN_PULL_REPLICATION_MSGS)
		for i := 0; i < cap(vs.pullReplicationState.inFreeMsgChan); i++ {
			vs.pullReplicationState.inFreeMsgChan <- &pullReplicationMsg{
				vs:     vs,
				header: make([]byte, ktBloomFilterHeaderBytes+pullReplicationMsgHeaderBytes),
			}
		}
		for i := 0; i < _GLH_IN_PULL_REPLICATION_HANDLERS; i++ {
			go vs.inPullReplication()
		}
		vs.pullReplicationState.outMsgChan = make(chan *pullReplicationMsg, _GLH_OUT_PULL_REPLICATION_MSGS)
		vs.pullReplicationState.outKTBFs = []*ktBloomFilter{newKTBloomFilter(_GLH_BLOOM_FILTER_N, _GLH_BLOOM_FILTER_P, 0)}
		for i := 0; i < cap(vs.pullReplicationState.outMsgChan); i++ {
			vs.pullReplicationState.outMsgChan <- &pullReplicationMsg{
				vs:     vs,
				header: make([]byte, ktBloomFilterHeaderBytes+pullReplicationMsgHeaderBytes),
				body:   make([]byte, len(vs.pullReplicationState.outKTBFs[0].bits)),
			}
		}
	}
	vs.pullReplicationState.outNotifyChan = make(chan *backgroundNotification, 1)
	go vs.outPullReplicationLauncher()
}

// DisableOutPullReplication will stop any outgoing pull replication requests
// until EnableOutPullReplication is called.
func (vs *DefaultValueStore) DisableOutPullReplication() {
	c := make(chan struct{}, 1)
	vs.pullReplicationState.outNotifyChan <- &backgroundNotification{
		disable:  true,
		doneChan: c,
	}
	<-c
}

// EnableOutPullReplication will resume outgoing pull replication requests.
func (vs *DefaultValueStore) EnableOutPullReplication() {
	c := make(chan struct{}, 1)
	vs.pullReplicationState.outNotifyChan <- &backgroundNotification{
		enable:   true,
		doneChan: c,
	}
	<-c
}

func (vs *DefaultValueStore) inPullReplication() {
	k := make([]uint64, 2*1024*1024)
	v := make([]byte, vs.maxValueSize)
	for {
		prm := <-vs.pullReplicationState.inMsgChan
		k = k[:0]
		cutoff := prm.cutoff()
		tombstoneCutoff := (uint64(brimtime.TimeToUnixMicro(time.Now())) << _TSB_UTIL_BITS) - vs.tombstoneDiscardState.age
		ktbf := prm.ktBloomFilter()
		l := int64(_GLH_OUT_BULK_SET_MSG_SIZE)
		vs.vlm.ScanCallback(prm.rangeStart(), prm.rangeStop(), func(keyA uint64, keyB uint64, timestampbits uint64, length uint32) {
			if l > 0 {
				if timestampbits&_TSB_LOCAL_REMOVAL == 0 && timestampbits < cutoff && (timestampbits&_TSB_DELETION == 0 || timestampbits >= tombstoneCutoff) {
					if !ktbf.mayHave(keyA, keyB, timestampbits) {
						k = append(k, keyA, keyB)
						// bsm: keyA:8, keyB:8, timestampbits:8, length:4,
						//      value:n
						l -= 28 + int64(length)
					}
				}
			}
		})
		nodeID := prm.nodeID()
		vs.pullReplicationState.inFreeMsgChan <- prm
		if len(k) > 0 {
			bsm := vs.newOutBulkSetMsg()
			var t uint64
			var err error
			for i := 0; i < len(k); i += 2 {
				t, v, err = vs.read(k[i], k[i+1], v[:0])
				if err == ErrNotFound {
					if t == 0 {
						continue
					}
				} else if err != nil {
					continue
				}
				if t&_TSB_LOCAL_REMOVAL == 0 {
					if !bsm.add(k[i], k[i+1], t, v) {
						break
					}
				}
			}
			if len(bsm.body) > 0 {
				if !vs.ring.MsgToNode(nodeID, bsm) {
					bsm.Done()
				}
			}
		}
	}
}

// OutPullReplicationPass will immediately execute an outgoing pull replication
// pass rather than waiting for the next interval. If a pass is currently
// executing, it will be stopped and restarted so that a call to this function
// ensures one complete pass occurs. Note that this pass will send the outgoing
// pull replication requests, but all the responses will almost certainly not
// have been received when this function returns. These requests are stateless,
// and so synchronization at that level is not possible.
func (vs *DefaultValueStore) OutPullReplicationPass() {
	atomic.StoreUint32(&vs.pullReplicationState.outAbort, 1)
	c := make(chan struct{}, 1)
	vs.pullReplicationState.outNotifyChan <- &backgroundNotification{doneChan: c}
	<-c
}

func (vs *DefaultValueStore) outPullReplicationLauncher() {
	var enabled bool
	interval := float64(vs.pullReplicationState.outInterval) * float64(time.Second)
	nextRun := time.Now().Add(time.Duration(interval + interval*vs.rand.NormFloat64()*0.1))
	for {
		var notification *backgroundNotification
		sleep := nextRun.Sub(time.Now())
		if sleep > 0 {
			select {
			case notification = <-vs.pullReplicationState.outNotifyChan:
			case <-time.After(sleep):
			}
		} else {
			select {
			case notification = <-vs.pullReplicationState.outNotifyChan:
			default:
			}
		}
		nextRun = time.Now().Add(time.Duration(interval + interval*vs.rand.NormFloat64()*0.1))
		if notification != nil {
			if notification.enable {
				enabled = true
				notification.doneChan <- struct{}{}
				continue
			}
			if notification.disable {
				atomic.StoreUint32(&vs.pullReplicationState.outAbort, 1)
				enabled = false
				notification.doneChan <- struct{}{}
				continue
			}
			atomic.StoreUint32(&vs.pullReplicationState.outAbort, 0)
			vs.outPullReplicationPass()
			notification.doneChan <- struct{}{}
		} else if enabled {
			atomic.StoreUint32(&vs.pullReplicationState.outAbort, 0)
			vs.outPullReplicationPass()
		}
	}
}

func (vs *DefaultValueStore) outPullReplicationPass() {
	if vs.ring == nil {
		return
	}
	if vs.logDebug != nil {
		begin := time.Now()
		defer func() {
			vs.logDebug.Printf("out pull replication pass took %s", time.Now().Sub(begin))
		}()
	}
	if vs.pullReplicationState.outIteration == math.MaxUint16 {
		vs.pullReplicationState.outIteration = 0
	} else {
		vs.pullReplicationState.outIteration++
	}
	ringID := vs.ring.ID()
	partitionPower := vs.ring.PartitionPower()
	partitions := uint32(1) << partitionPower
	for len(vs.pullReplicationState.outKTBFs) < vs.pullReplicationState.outWorkers {
		vs.pullReplicationState.outKTBFs = append(vs.pullReplicationState.outKTBFs, newKTBloomFilter(_GLH_BLOOM_FILTER_N, _GLH_BLOOM_FILTER_P, 0))
	}
	// GLH TODO: Redo this to split up work like tombstoneDiscard does. Also,
	// don't do ScanCount, instead do something like tombstoneDiscard does as
	// well where the ScanCallback stops after max items and reports where it
	// stopped for the next scan. The difference here is that the ScanCallback
	// should continue until done with an entire range (slightly overfilling
	// the bloom filter) instead of stopping immediately.
	f := func(p uint32, ktbf *ktBloomFilter) {
		start := uint64(p) << uint64(64-partitionPower)
		stop := start + (uint64(1)<<(64-partitionPower) - 1)
		pullSize := uint64(1) << (64 - partitionPower)
		for vs.vlm.ScanCount(start, start+(pullSize-1), _GLH_BLOOM_FILTER_N) >= _GLH_BLOOM_FILTER_N {
			pullSize /= 2
		}
		timestampbitsnow := uint64(brimtime.TimeToUnixMicro(time.Now())) << _TSB_UTIL_BITS
		cutoff := timestampbitsnow - vs.replicationIgnoreRecent
		tombstoneCutoff := timestampbitsnow - vs.tombstoneDiscardState.age
		substart := start
		substop := start + (pullSize - 1)
		for atomic.LoadUint32(&vs.pullReplicationState.outAbort) == 0 {
			ktbf.reset(vs.pullReplicationState.outIteration)
			vs.vlm.ScanCallback(substart, substop, func(keyA uint64, keyB uint64, timestampbits uint64, length uint32) {
				if timestampbits&_TSB_LOCAL_REMOVAL == 0 && timestampbits < cutoff && (timestampbits&_TSB_DELETION == 0 || timestampbits >= tombstoneCutoff) {
					ktbf.add(keyA, keyB, timestampbits)
				}
			})
			if atomic.LoadUint32(&vs.pullReplicationState.outAbort) != 0 {
				break
			}
			prm := vs.newOutPullReplicationMsg(ringID, p, cutoff, substart, substop, ktbf)
			if !vs.ring.MsgToOtherReplicas(ringID, p, prm) {
				prm.Done()
			}
			substart += pullSize
			substop += pullSize
			if substop > stop || substop < stop {
				break
			}
		}
	}
	sp := uint32(vs.rand.Intn(int(partitions)))
	wg := &sync.WaitGroup{}
	wg.Add(vs.pullReplicationState.outWorkers)
	for g := 0; g < vs.pullReplicationState.outWorkers; g++ {
		go func(g uint32) {
			ktbf := vs.pullReplicationState.outKTBFs[g]
			for p := uint32(sp + g); p < partitions && atomic.LoadUint32(&vs.pullReplicationState.outAbort) == 0; p += uint32(vs.pullReplicationState.outWorkers) {
				if vs.ring.ID() != ringID {
					break
				}
				if vs.ring.Responsible(p) {
					f(p, ktbf)
				}
			}
			for p := uint32(g); p < sp && atomic.LoadUint32(&vs.pullReplicationState.outAbort) == 0; p += uint32(vs.pullReplicationState.outWorkers) {
				if vs.ring.ID() != ringID {
					break
				}
				if vs.ring.Responsible(p) {
					f(p, ktbf)
				}
			}
			wg.Done()
		}(uint32(g))
	}
	wg.Wait()
}

var toss []byte = make([]byte, 65536)

func (vs *DefaultValueStore) newInPullReplicationMsg(r io.Reader, l uint64) (uint64, error) {
	var prm *pullReplicationMsg
	select {
	case prm = <-vs.pullReplicationState.inFreeMsgChan:
	case <-time.After(_GLH_IN_PULL_REPLICATION_MSG_TIMEOUT * time.Second):
		var n uint64
		var sn int
		var err error
		for n < l {
			sn, err = r.Read(toss)
			n += uint64(sn)
			if err != nil {
				return n, err
			}
		}
		return n, nil
	}
	bl := l - pullReplicationMsgHeaderBytes - uint64(ktBloomFilterHeaderBytes)
	if uint64(cap(prm.body)) < bl {
		prm.body = make([]byte, bl)
	}
	prm.body = prm.body[:bl]
	var n int
	var sn int
	var err error
	for n != len(prm.header) {
		if err != nil {
			return uint64(n), err
		}
		sn, err = r.Read(prm.header[n:])
		n += sn
	}
	n = 0
	for n != len(prm.body) {
		if err != nil {
			return uint64(len(prm.header)) + uint64(n), err
		}
		sn, err = r.Read(prm.body[n:])
		n += sn
	}
	vs.pullReplicationState.inMsgChan <- prm
	return l, nil
}

func (vs *DefaultValueStore) newOutPullReplicationMsg(ringID uint64, partition uint32, cutoff uint64, rangeStart uint64, rangeStop uint64, ktbf *ktBloomFilter) *pullReplicationMsg {
	prm := <-vs.pullReplicationState.outMsgChan
	binary.BigEndian.PutUint64(prm.header, vs.ring.NodeID())
	binary.BigEndian.PutUint64(prm.header[8:], ringID)
	binary.BigEndian.PutUint32(prm.header[16:], partition)
	binary.BigEndian.PutUint64(prm.header[20:], cutoff)
	binary.BigEndian.PutUint64(prm.header[28:], rangeStart)
	binary.BigEndian.PutUint64(prm.header[36:], rangeStop)
	ktbf.toMsg(prm, pullReplicationMsgHeaderBytes)
	return prm
}

func (prm *pullReplicationMsg) MsgType() ring.MsgType {
	return ring.MSG_PULL_REPLICATION
}

func (prm *pullReplicationMsg) MsgLength() uint64 {
	return uint64(len(prm.header)) + uint64(len(prm.body))
}

func (prm *pullReplicationMsg) nodeID() uint64 {
	return binary.BigEndian.Uint64(prm.header)
}

func (prm *pullReplicationMsg) ringID() uint64 {
	return binary.BigEndian.Uint64(prm.header[8:])
}

func (prm *pullReplicationMsg) partition() uint32 {
	return binary.BigEndian.Uint32(prm.header[16:])
}

func (prm *pullReplicationMsg) cutoff() uint64 {
	return binary.BigEndian.Uint64(prm.header[20:])
}

func (prm *pullReplicationMsg) rangeStart() uint64 {
	return binary.BigEndian.Uint64(prm.header[28:])
}

func (prm *pullReplicationMsg) rangeStop() uint64 {
	return binary.BigEndian.Uint64(prm.header[36:])
}

func (prm *pullReplicationMsg) ktBloomFilter() *ktBloomFilter {
	return newKTBloomFilterFromMsg(prm, pullReplicationMsgHeaderBytes)
}

func (prm *pullReplicationMsg) WriteContent(w io.Writer) (uint64, error) {
	var n int
	var sn int
	var err error
	sn, err = w.Write(prm.header)
	n += sn
	if err != nil {
		return uint64(n), err
	}
	sn, err = w.Write(prm.body)
	n += sn
	return uint64(n), err
}

func (prm *pullReplicationMsg) Done() {
	prm.vs.pullReplicationState.outMsgChan <- prm
}