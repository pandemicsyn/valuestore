package valuestore

import (
	"bytes"
	"encoding/binary"
	"io"
	"testing"

	"github.com/gholt/ring"
)

type testBulkSetMsgRing struct {
	ring         ring.Ring
	msgToNodeIDs []uint64
}

func (m *testBulkSetMsgRing) Ring() ring.Ring {
	return m.ring
}

func (m *testBulkSetMsgRing) MaxMsgLength() uint64 {
	return 65536
}

func (m *testBulkSetMsgRing) SetMsgHandler(msgType uint64, handler ring.MsgUnmarshaller) {
}

func (m *testBulkSetMsgRing) MsgToNode(nodeID uint64, msg ring.Msg) {
	m.msgToNodeIDs = append(m.msgToNodeIDs, nodeID)
}

func (m *testBulkSetMsgRing) MsgToOtherReplicas(ringVersion int64, partition uint32, msg ring.Msg) {
}

type testErrorWriter struct {
	goodBytes int
}

func (w *testErrorWriter) Write(p []byte) (int, error) {
	if w.goodBytes >= len(p) {
		w.goodBytes -= len(p)
		return len(p), nil
	}
	if w.goodBytes > 0 {
		n := w.goodBytes
		w.goodBytes = 0
		return n, io.EOF
	}
	return 0, io.EOF
}

func TestBulkSetInTimeout(t *testing.T) {
	vs := New(&Config{
		MsgRing:             &testBulkSetMsgRing{},
		InBulkSetMsgTimeout: 1,
	})
	// This means that the subsystem can never get a free bulkSetMsg since we
	// never feed this replacement channel.
	vs.bulkSetState.inFreeMsgChan = make(chan *bulkSetMsg, 1)
	n, err := vs.newInBulkSetMsg(bytes.NewBuffer(make([]byte, 100)), 100)
	// Validates we got no error and read all the bytes; meaning the message
	// was read and tossed after the timeout in getting a free bulkSetMsg.
	if err != nil {
		t.Fatal(err)
	}
	if n != 100 {
		t.Fatal(n)
	}
	// Try again to make sure it can handle Reader errors.
	n, err = vs.newInBulkSetMsg(bytes.NewBuffer(make([]byte, 10)), 100)
	if err != io.EOF {
		t.Fatal(err)
	}
	if n != 10 {
		t.Fatal(n)
	}
}

func TestBulkSetReadObviouslyTooShort(t *testing.T) {
	vs := New(&Config{MsgRing: &testBulkSetMsgRing{}})
	// Just in case the test fails, we don't want the subsystem to try to
	// interpret our junk bytes.
	vs.bulkSetState.inMsgChan = make(chan *bulkSetMsg, 1)
	n, err := vs.newInBulkSetMsg(bytes.NewBuffer(make([]byte, 1)), 1)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatal(n)
	}
	select {
	case <-vs.bulkSetState.inMsgChan:
		t.Fatal("")
	default:
	}
	// Once again, way too short but with an error too.
	_, err = vs.newInBulkSetMsg(bytes.NewBuffer(make([]byte, 1)), 2)
	if err != io.EOF {
		t.Fatal(err)
	}
	select {
	case <-vs.bulkSetState.inMsgChan:
		t.Fatal("")
	default:
	}
}

func TestBulkSetRead(t *testing.T) {
	vs := New(&Config{MsgRing: &testBulkSetMsgRing{}})
	// We don't want the subsystem to try to interpret our junk bytes.
	vs.bulkSetState.inMsgChan = make(chan *bulkSetMsg, 1)
	n, err := vs.newInBulkSetMsg(bytes.NewBuffer(make([]byte, 100)), 100)
	if err != nil {
		t.Fatal(err)
	}
	if n != 100 {
		t.Fatal(n)
	}
	select {
	case <-vs.bulkSetState.inMsgChan:
	default:
		t.Fatal("")
	}
	// Again, but with an error in the header.
	n, err = vs.newInBulkSetMsg(bytes.NewBuffer(make([]byte, _BULK_SET_MSG_HEADER_LENGTH-1)), 100)
	if err != io.EOF {
		t.Fatal(err)
	}
	if n != _BULK_SET_MSG_HEADER_LENGTH-1 {
		t.Fatal(n)
	}
	select {
	case <-vs.bulkSetState.inMsgChan:
		t.Fatal("")
	default:
	}
	// Once again, but with an error in the body.
	n, err = vs.newInBulkSetMsg(bytes.NewBuffer(make([]byte, 10)), 100)
	if err != io.EOF {
		t.Fatal(err)
	}
	if n != 10 {
		t.Fatal(n)
	}
	select {
	case <-vs.bulkSetState.inMsgChan:
		t.Fatal("")
	default:
	}
}

func TestBulkSetReadLowSendCap(t *testing.T) {
	vs := New(&Config{MsgRing: &testBulkSetMsgRing{}, OutBulkSetMsgCap: _BULK_SET_MSG_HEADER_LENGTH + 1})
	// We don't want the subsystem to try to interpret our junk bytes.
	vs.bulkSetState.inMsgChan = make(chan *bulkSetMsg, 1)
	n, err := vs.newInBulkSetMsg(bytes.NewBuffer(make([]byte, 100)), 100)
	if err != nil {
		t.Fatal(err)
	}
	if n != 100 {
		t.Fatal(n)
	}
	select {
	case <-vs.bulkSetState.inMsgChan:
	default:
		t.Fatal("")
	}
}

func TestBulkSetMsgWithoutAck(t *testing.T) {
	b := ring.NewBuilder()
	n := b.AddNode(true, 1, nil, nil, "", nil)
	r := b.Ring()
	r.SetLocalNode(n.ID())
	m := &testBulkSetMsgRing{ring: r}
	vs := New(&Config{
		MsgRing:          m,
		InBulkSetWorkers: 1,
		InBulkSetMsgs:    1,
	})
	vs.EnableAll()
	bsm := <-vs.bulkSetState.inFreeMsgChan
	bsm.body = bsm.body[:0]
	if !bsm.add(1, 2, 0x300, []byte("testing")) {
		t.Fatal("")
	}
	vs.bulkSetState.inMsgChan <- bsm
	// only one of these, so if we get it back we know the previous data was
	// processed
	<-vs.bulkSetState.inFreeMsgChan
	ts, v, err := vs.Read(1, 2, nil)
	if err != nil {
		t.Fatal(err)
	}
	if ts != 3 { // the bottom 8 bits are discarded for the public Read
		t.Fatal(ts)
	}
	if string(v) != "testing" {
		t.Fatal(string(v))
	}
	if len(m.msgToNodeIDs) != 0 {
		t.Fatal(len(m.msgToNodeIDs))
	}
}

func TestBulkSetMsgWithAck(t *testing.T) {
	b := ring.NewBuilder()
	n := b.AddNode(true, 1, nil, nil, "", nil)
	r := b.Ring()
	r.SetLocalNode(n.ID())
	m := &testBulkSetMsgRing{ring: r}
	vs := New(&Config{
		MsgRing:          m,
		InBulkSetWorkers: 1,
		InBulkSetMsgs:    1,
	})
	vs.EnableAll()
	bsm := <-vs.bulkSetState.inFreeMsgChan
	binary.BigEndian.PutUint64(bsm.header, 123)
	bsm.body = bsm.body[:0]
	if !bsm.add(1, 2, 0x300, []byte("testing")) {
		t.Fatal("")
	}
	vs.bulkSetState.inMsgChan <- bsm
	// only one of these, so if we get it back we know the previous data was
	// processed
	<-vs.bulkSetState.inFreeMsgChan
	ts, v, err := vs.Read(1, 2, nil)
	if err != nil {
		t.Fatal(err)
	}
	if ts != 3 { // the bottom 8 bits are discarded for the public Read
		t.Fatal(ts)
	}
	if string(v) != "testing" {
		t.Fatal(string(v))
	}
	if len(m.msgToNodeIDs) != 1 {
		t.Fatal(len(m.msgToNodeIDs))
	}
	if m.msgToNodeIDs[0] != 123 {
		t.Fatal(m.msgToNodeIDs[0])
	}
}

func TestBulkSetMsgOut(t *testing.T) {
	vs := New(&Config{MsgRing: &testBulkSetMsgRing{}})
	bsm := vs.newOutBulkSetMsg()
	if bsm.MsgType() != _BULK_SET_MSG_TYPE {
		t.Fatal(bsm.MsgType())
	}
	if bsm.MsgLength() != _BULK_SET_MSG_HEADER_LENGTH {
		t.Fatal(bsm.MsgLength())
	}
	buf := bytes.NewBuffer(nil)
	n, err := bsm.WriteContent(buf)
	if err != nil {
		t.Fatal(err)
	}
	if n != _BULK_SET_MSG_HEADER_LENGTH {
		t.Fatal(n)
	}
	if !bytes.Equal(buf.Bytes(), []byte{0, 0, 0, 0, 0, 0, 0, 0}) {
		t.Fatal(buf.Bytes())
	}
	bsm.Done()
	bsm = vs.newOutBulkSetMsg()
	binary.BigEndian.PutUint64(bsm.header, 12345)
	bsm.add(1, 2, 0x300, nil)
	bsm.add(4, 5, 0x600, []byte("testing"))
	if bsm.MsgType() != _BULK_SET_MSG_TYPE {
		t.Fatal(bsm.MsgType())
	}
	if bsm.MsgLength() != _BULK_SET_MSG_HEADER_LENGTH+_BULK_SET_MSG_ENTRY_HEADER_LENGTH+0+_BULK_SET_MSG_ENTRY_HEADER_LENGTH+7 {
		t.Fatal(bsm.MsgLength())
	}
	buf = bytes.NewBuffer(nil)
	n, err = bsm.WriteContent(buf)
	if err != nil {
		t.Fatal(err)
	}
	if n != _BULK_SET_MSG_HEADER_LENGTH+_BULK_SET_MSG_ENTRY_HEADER_LENGTH+0+_BULK_SET_MSG_ENTRY_HEADER_LENGTH+7 {
		t.Fatal(n)
	}
	if !bytes.Equal(buf.Bytes(), []byte{
		0, 0, 0, 0, 0, 0, 48, 57, // header
		0, 0, 0, 0, 0, 0, 0, 1, // keyA
		0, 0, 0, 0, 0, 0, 0, 2, // keyB
		0, 0, 0, 0, 0, 0, 3, 0, // timestamp
		0, 0, 0, 0, // length
		0, 0, 0, 0, 0, 0, 0, 4, // keyA
		0, 0, 0, 0, 0, 0, 0, 5, // keyB
		0, 0, 0, 0, 0, 0, 6, 0, // timestamp
		0, 0, 0, 7, // length
		116, 101, 115, 116, 105, 110, 103, // "testing"
	}) {
		t.Fatal(buf.Bytes())
	}
	bsm.Done()
}

func TestBulkSetMsgOutWriteError(t *testing.T) {
	vs := New(&Config{MsgRing: &testBulkSetMsgRing{}})
	bsm := vs.newOutBulkSetMsg()
	_, err := bsm.WriteContent(&testErrorWriter{})
	if err == nil {
		t.Fatal(err)
	}
	bsm.Done()
}

func TestBulkSetMsgOutHitCap(t *testing.T) {
	vs := New(&Config{MsgRing: &testBulkSetMsgRing{}, OutBulkSetMsgCap: _BULK_SET_MSG_HEADER_LENGTH + _BULK_SET_MSG_ENTRY_HEADER_LENGTH + 3})
	bsm := vs.newOutBulkSetMsg()
	if !bsm.add(1, 2, 0x300, []byte("1")) {
		t.Fatal("")
	}
	if bsm.add(1, 2, 0x300, []byte("12345678901234567890")) {
		t.Fatal("")
	}
}