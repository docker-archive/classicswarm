package zoo

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	log "github.com/golang/glog"
	"github.com/mesos/mesos-go/detector"
	mesos "github.com/mesos/mesos-go/mesosproto"
	util "github.com/mesos/mesos-go/mesosutil"
	"github.com/samuel/go-zookeeper/zk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const (
	zkurl        = "zk://127.0.0.1:2181/mesos"
	zkurl_bad    = "zk://127.0.0.1:2181"
	test_zk_path = "/test"
)

func TestParseZk_single(t *testing.T) {
	hosts, path, err := parseZk(zkurl)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(hosts))
	assert.Equal(t, "/mesos", path)
}

func TestParseZk_multi(t *testing.T) {
	hosts, path, err := parseZk("zk://abc:1,def:2/foo")
	assert.NoError(t, err)
	assert.Equal(t, []string{"abc:1", "def:2"}, hosts)
	assert.Equal(t, "/foo", path)
}

func TestParseZk_multiIP(t *testing.T) {
	hosts, path, err := parseZk("zk://10.186.175.156:2181,10.47.50.94:2181,10.0.92.171:2181/mesos")
	assert.NoError(t, err)
	assert.Equal(t, []string{"10.186.175.156:2181", "10.47.50.94:2181", "10.0.92.171:2181"}, hosts)
	assert.Equal(t, "/mesos", path)
}

type mockZkClient struct {
	mock.Mock
}

func (m *mockZkClient) stopped() (a <-chan struct{}) {
	args := m.Called()
	if x := args.Get(0); x != nil {
		a = x.(<-chan struct{})
	}
	return
}

func (m *mockZkClient) stop() {
	m.Called()
}

func (m *mockZkClient) data(path string) (a []byte, b error) {
	args := m.Called(path)
	if x := args.Get(0); x != nil {
		a = x.([]byte)
	}
	b = args.Error(1)
	return
}

func (m *mockZkClient) watchChildren(path string) (a string, b <-chan []string, c <-chan error) {
	args := m.Called(path)
	a = args.String(0)
	if x := args.Get(1); x != nil {
		b = x.(<-chan []string)
	}
	if x := args.Get(2); x != nil {
		c = x.(<-chan error)
	}
	return
}

// newMockZkClient returns a mocked implementation of zkInterface that implements expectations
// for stop() and stopped(); multiple calls to stop() are safe.
func newMockZkClient(initialChildren ...string) (mocked *mockZkClient, snaps chan []string, errs chan error) {
	var doneOnce sync.Once
	done := make(chan struct{})

	mocked = &mockZkClient{}
	mocked.On("stop").Return().Run(func(_ mock.Arguments) { doneOnce.Do(func() { close(done) }) })
	mocked.On("stopped").Return((<-chan struct{})(done))

	if initialChildren != nil {
		errs = make(chan error) // this is purposefully unbuffered (some tests depend on this)
		snaps = make(chan []string, 1)
		snaps <- initialChildren[:]
		mocked.On("watchChildren", currentPath).Return(
			test_zk_path, (<-chan []string)(snaps), (<-chan error)(errs)).Run(
			func(_ mock.Arguments) { log.V(1).Infoln("watchChildren invoked") })
	}
	return
}

func newTestMasterInfo(id int) []byte {
	miPb := util.NewMasterInfo(fmt.Sprintf("master(%d)@localhost:5050", id), 123456789, 400)
	data, err := proto.Marshal(miPb)
	if err != nil {
		panic(err)
	}
	return data
}

func TestMasterDetectorChildrenChanged(t *testing.T) {
	md, err := NewMasterDetector(zkurl)
	defer md.Cancel()
	assert.NoError(t, err)

	path := test_zk_path
	snapDetected := make(chan struct{})
	md.bootstrapFunc = func() error {
		if md.client != nil {
			return nil
		}
		log.V(1).Infoln("bootstrapping detector")
		defer log.V(1).Infoln("bootstrapping detector ..finished")

		mocked, _, errs := newMockZkClient("info_0", "info_5", "info_10")
		md.client = mocked
		md.minDetectorCyclePeriod = 10 * time.Millisecond // we don't have all day!

		mocked.On("data", fmt.Sprintf("%s/info_0", path)).Return(newTestMasterInfo(0), nil)

		// wait for the first child snapshot to be processed before signaling end-of-watch
		// (which is signalled by closing errs).
		go func() {
			defer close(errs)
			select {
			case <-snapDetected:
			case <-md.Done():
				t.Errorf("detector died before child snapshot")
			}
		}()
		return nil
	}

	called := 0
	lostMaster := make(chan struct{})
	const expectedLeader = "master(0)@localhost:5050"
	err = md.Detect(detector.OnMasterChanged(func(master *mesos.MasterInfo) {
		//expect 2 calls in sequence: the first setting a master
		//and the second clearing it
		switch called++; called {
		case 1:
			defer close(snapDetected)
			assert.NotNil(t, master)
			assert.Equal(t, expectedLeader, master.GetId())
		case 2:
			md.Cancel()
			defer close(lostMaster)
			assert.Nil(t, master)
		default:
			t.Errorf("unexpected notification call attempt %d", called)
		}
	}))
	assert.NoError(t, err)

	fatalOn(t, 10*time.Second, lostMaster, "Waited too long for lost master")

	select {
	case <-md.Done():
		assert.Equal(t, 2, called, "expected 2 detection callbacks instead of %d", called)
	case <-time.After(time.Second * 10):
		panic("Waited too long for detector shutdown...")
	}
}

// single connector instance, it's internal connection to zk is flappy
func TestMasterDetectorFlappyConnectionState(t *testing.T) {
	md, err := NewMasterDetector(zkurl)
	defer md.Cancel()
	assert.NoError(t, err)

	const ITERATIONS = 3
	var wg sync.WaitGroup
	wg.Add(1 + ITERATIONS) // +1 for the initial snapshot that's sent for the first watch
	path := test_zk_path

	md.bootstrapFunc = func() error {
		if md.client != nil {
			return nil
		}
		log.V(1).Infoln("bootstrapping detector")
		defer log.V(1).Infoln("bootstrapping detector ..finished")

		children := []string{"info_0", "info_5", "info_10"}
		mocked, snaps, errs := newMockZkClient(children...)
		md.client = mocked
		md.minDetectorCyclePeriod = 10 * time.Millisecond // we don't have all day!

		mocked.On("data", fmt.Sprintf("%s/info_0", path)).Return(newTestMasterInfo(0), nil)

		// the first snapshot will be sent immediately and the detector will be awaiting en event.
		// cycle through some connected/disconnected events but maintain the same snapshot
		go func() {
			defer close(errs)
			for attempt := 0; attempt < ITERATIONS; attempt++ {
				// send an error, should cause the detector to re-issue a watch
				errs <- zk.ErrSessionExpired
				// the detection loop issues another watch, so send it a snapshot..
				// send another snapshot
				snaps <- children
			}
		}()
		return nil
	}

	called := 0
	lostMaster := make(chan struct{})
	const EXPECTED_CALLS = (ITERATIONS * 2) + 2 // +1 for initial snapshot, +1 for final lost-leader (close(errs))
	err = md.Detect(detector.OnMasterChanged(func(master *mesos.MasterInfo) {
		called++
		log.V(3).Infof("detector invoked: called %d", called)
		switch {
		case called < EXPECTED_CALLS:
			if master != nil {
				wg.Done()
				assert.Equal(t, master.GetId(), "master(0)@localhost:5050")
			}
		case called == EXPECTED_CALLS:
			md.Cancel()
			defer close(lostMaster)
			assert.Nil(t, master)
		default:
			t.Errorf("unexpected notification call attempt %d", called)
		}
	}))
	assert.NoError(t, err)

	fatalAfter(t, 10*time.Second, wg.Wait, "Waited too long for new-master alerts")
	fatalOn(t, 3*time.Second, lostMaster, "Waited too long for lost master")

	select {
	case <-md.Done():
		assert.Equal(t, EXPECTED_CALLS, called, "expected %d detection callbacks instead of %d", EXPECTED_CALLS, called)
	case <-time.After(time.Second * 10):
		panic("Waited too long for detector shutdown...")
	}
}

func TestMasterDetector_multipleLeadershipChanges(t *testing.T) {
	md, err := NewMasterDetector(zkurl)
	defer md.Cancel()
	assert.NoError(t, err)

	leadershipChanges := [][]string{
		{"info_014", "info_010", "info_005"},
		{"info_005", "info_004", "info_022"},
		{}, // indicates no master
		{"info_017", "info_099", "info_200"},
	}

	ITERATIONS := len(leadershipChanges)

	// +1 for initial snapshot, +1 for final lost-leader (close(errs))
	EXPECTED_CALLS := (ITERATIONS + 2)

	var wg sync.WaitGroup
	wg.Add(ITERATIONS) // +1 for the initial snapshot that's sent for the first watch, -1 because set 3 is empty
	path := test_zk_path

	md.bootstrapFunc = func() error {
		if md.client != nil {
			return nil
		}
		log.V(1).Infoln("bootstrapping detector")
		defer log.V(1).Infoln("bootstrapping detector ..finished")

		children := []string{"info_0", "info_5", "info_10"}
		mocked, snaps, errs := newMockZkClient(children...)
		md.client = mocked
		md.minDetectorCyclePeriod = 10 * time.Millisecond // we don't have all day!

		mocked.On("data", fmt.Sprintf("%s/info_0", path)).Return(newTestMasterInfo(0), nil)
		mocked.On("data", fmt.Sprintf("%s/info_005", path)).Return(newTestMasterInfo(5), nil)
		mocked.On("data", fmt.Sprintf("%s/info_004", path)).Return(newTestMasterInfo(4), nil)
		mocked.On("data", fmt.Sprintf("%s/info_017", path)).Return(newTestMasterInfo(17), nil)

		// the first snapshot will be sent immediately and the detector will be awaiting en event.
		// cycle through some connected/disconnected events but maintain the same snapshot
		go func() {
			defer close(errs)
			for attempt := 0; attempt < ITERATIONS; attempt++ {
				snaps <- leadershipChanges[attempt]
			}
		}()
		return nil
	}

	called := 0
	lostMaster := make(chan struct{})
	expectedLeaders := []int{0, 5, 4, 17}
	leaderIdx := 0
	err = md.Detect(detector.OnMasterChanged(func(master *mesos.MasterInfo) {
		called++
		log.V(3).Infof("detector invoked: called %d", called)
		switch {
		case called < EXPECTED_CALLS:
			if master != nil {
				expectedLeader := fmt.Sprintf("master(%d)@localhost:5050", expectedLeaders[leaderIdx])
				assert.Equal(t, expectedLeader, master.GetId())
				leaderIdx++
				wg.Done()
			}
		case called == EXPECTED_CALLS:
			md.Cancel()
			defer close(lostMaster)
			assert.Nil(t, master)
		default:
			t.Errorf("unexpected notification call attempt %d", called)
		}
	}))
	assert.NoError(t, err)

	fatalAfter(t, 10*time.Second, wg.Wait, "Waited too long for new-master alerts")
	fatalOn(t, 3*time.Second, lostMaster, "Waited too long for lost master")

	select {
	case <-md.Done():
		assert.Equal(t, EXPECTED_CALLS, called, "expected %d detection callbacks instead of %d", EXPECTED_CALLS, called)
	case <-time.After(time.Second * 10):
		panic("Waited too long for detector shutdown...")
	}
}

func TestMasterDetect_selectTopNode_none(t *testing.T) {
	assert := assert.New(t)
	nodeList := []string{}
	node := selectTopNodePrefix(nodeList, "foo")
	assert.Equal("", node)
}

func TestMasterDetect_selectTopNode_0000x(t *testing.T) {
	assert := assert.New(t)
	nodeList := []string{
		"info_0000000046",
		"info_0000000032",
		"info_0000000058",
		"info_0000000061",
		"info_0000000008",
	}
	node := selectTopNodePrefix(nodeList, nodePrefix)
	assert.Equal("info_0000000008", node)
}

func TestMasterDetect_selectTopNode_mixJson(t *testing.T) {
	assert := assert.New(t)
	nodeList := []string{
		nodePrefix + "0000000046",
		nodePrefix + "0000000032",
		nodeJSONPrefix + "0000000046",
		nodeJSONPrefix + "0000000032",
	}
	node := selectTopNodePrefix(nodeList, nodeJSONPrefix)
	assert.Equal(nodeJSONPrefix+"0000000032", node)

	node = selectTopNodePrefix(nodeList, nodePrefix)
	assert.Equal(nodePrefix+"0000000032", node)
}

func TestMasterDetect_selectTopNode_mixedEntries(t *testing.T) {
	assert := assert.New(t)
	nodeList := []string{
		"info_0000000046",
		"info_0000000032",
		"foo_lskdjfglsdkfsdfgdfg",
		"info_0000000061",
		"log_replicas_fdgwsdfgsdf",
		"bar",
	}
	node := selectTopNodePrefix(nodeList, nodePrefix)
	assert.Equal("info_0000000032", node)
}

// implements MasterChanged and AllMasters extension
type allMastersListener struct {
	mock.Mock
}

func (a *allMastersListener) OnMasterChanged(mi *mesos.MasterInfo) {
	a.Called(mi)
}

func (a *allMastersListener) UpdatedMasters(mi []*mesos.MasterInfo) {
	a.Called(mi)
}

func afterFunc(f func()) <-chan struct{} {
	ch := make(chan struct{})
	go func() {
		defer close(ch)
		f()
	}()
	return ch
}

func fatalAfter(t *testing.T, d time.Duration, f func(), msg string, args ...interface{}) {
	fatalOn(t, d, afterFunc(f), msg, args...)
}

func fatalOn(t *testing.T, d time.Duration, ch <-chan struct{}, msg string, args ...interface{}) {
	select {
	case <-ch:
		return
	case <-time.After(d):
		// check for a tie
		select {
		case <-ch:
			return
		default:
			t.Fatalf(msg, args...)
		}
	}
}

/* TODO(jdef) refactor this to work with the new zkInterface
func TestNotifyAllMasters(t *testing.T) {
	c, err := newClient(test_zk_hosts, test_zk_path)
	assert.NoError(t, err)

	childEvents := make(chan zk.Event, 5)
	connector := NewMockConnector()

	c.setFactory(asFactory(func() (Connector, <-chan zk.Event, error) {
		sessionEvents := make(chan zk.Event, 1)
		sessionEvents <- zk.Event{
			Type:  zk.EventSession,
			State: zk.StateConnected,
		}
		return connector, sessionEvents, nil
	}))

	md, err := NewMasterDetector(zkurl)
	defer md.Cancel()

	assert.NoError(t, err)

	c.errorHandler = ErrorHandler(func(c *Client, e error) {
		t.Errorf("unexpected error: %v", e)
	})
	md.client = c

	listener := &allMastersListener{}

	//-- expect primer
	var primer sync.WaitGroup
	ignoreArgs := func(f func()) func(mock.Arguments) {
		primer.Add(1)
		return func(_ mock.Arguments) {
			f()
		}
	}
	connector.On("Children", test_zk_path).Return([]string{}, &zk.Stat{}, nil).Run(ignoreArgs(primer.Done)).Once()
	listener.On("UpdatedMasters", []*mesos.MasterInfo{}).Return().Run(ignoreArgs(primer.Done)).Once()
	connector.On("ChildrenW", test_zk_path).Return([]string{test_zk_path}, &zk.Stat{}, (<-chan zk.Event)(childEvents), nil).Run(ignoreArgs(primer.Done)).Once()
	md.Detect(listener)
	fatalAfter(t, 3*time.Second, primer.Wait, "timed out waiting for detection primer")

	listener.AssertExpectations(t)
	connector.AssertExpectations(t)

	//-- test membership changes
	type expectedGets struct {
		info []byte
		err  error
	}
	tt := []struct {
		zkEntry   []string
		gets      []expectedGets
		leaderIdx int
	}{
		{[]string{"info_004"}, []expectedGets{{newTestMasterInfo(1), nil}}, 0},
		{[]string{"info_007", "info_005", "info_006"}, []expectedGets{{newTestMasterInfo(2), nil}, {newTestMasterInfo(3), nil}, {newTestMasterInfo(4), nil}}, 1},
		{nil, nil, -1},
	}
	for j, tc := range tt {
		// expectations
		var tcwait sync.WaitGroup
		ignoreArgs = func(f func()) func(mock.Arguments) {
			tcwait.Add(1)
			return func(_ mock.Arguments) {
				f()
			}
		}

		expectedInfos := []*mesos.MasterInfo{}
		for i, zke := range tc.zkEntry {
			connector.On("Get", fmt.Sprintf("%s/%s", test_zk_path, zke)).Return(tc.gets[i].info, &zk.Stat{}, tc.gets[i].err).Run(ignoreArgs(tcwait.Done)).Once()
			masterInfo := &mesos.MasterInfo{}
			err = proto.Unmarshal(tc.gets[i].info, masterInfo)
			if err != nil {
				t.Fatalf("failed to unmarshall MasterInfo data: %v", err)
			}
			expectedInfos = append(expectedInfos, masterInfo)
		}
		if len(tc.zkEntry) > 0 {
			connector.On("Get", fmt.Sprintf("%s/%s", test_zk_path, tc.zkEntry[tc.leaderIdx])).Return(
				tc.gets[tc.leaderIdx].info, &zk.Stat{}, tc.gets[tc.leaderIdx].err).Run(ignoreArgs(tcwait.Done)).Once()
		}
		connector.On("Children", test_zk_path).Return(tc.zkEntry, &zk.Stat{}, nil).Run(ignoreArgs(tcwait.Done)).Once()
		listener.On("OnMasterChanged", mock.AnythingOfType("*mesosproto.MasterInfo")).Return().Run(ignoreArgs(tcwait.Done)).Once()
		listener.On("UpdatedMasters", expectedInfos).Return().Run(ignoreArgs(tcwait.Done)).Once()
		connector.On("ChildrenW", test_zk_path).Return([]string{test_zk_path}, &zk.Stat{}, (<-chan zk.Event)(childEvents), nil).Run(ignoreArgs(tcwait.Done)).Once()

		// fire the event that triggers the test case
		childEvents <- zk.Event{
			Type: zk.EventNodeChildrenChanged,
			Path: test_zk_path,
		}

		// allow plenty of time for all the async processing to happen
		fatalAfter(t, 5*time.Second, tcwait.Wait, "timed out waiting for all-masters test case %d", j+1)
		listener.AssertExpectations(t)
		connector.AssertExpectations(t)
	}

	connector.On("Close").Return(nil)
}
*/
