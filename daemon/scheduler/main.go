package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/opensvc/om3/core/collector"
	"github.com/opensvc/om3/core/naming"
	"github.com/opensvc/om3/core/node"
	"github.com/opensvc/om3/core/object"
	"github.com/opensvc/om3/core/provisioned"
	"github.com/opensvc/om3/core/schedule"
	"github.com/opensvc/om3/daemon/daemondata"
	"github.com/opensvc/om3/daemon/msgbus"
	"github.com/opensvc/om3/util/funcopt"
	"github.com/opensvc/om3/util/hostname"
	"github.com/opensvc/om3/util/plog"
	"github.com/opensvc/om3/util/pubsub"
)

type (
	T struct {
		ctx       context.Context
		cancel    context.CancelFunc
		log       *plog.Logger
		localhost string
		databus   *daemondata.T
		pubsub    *pubsub.Bus

		events      chan any
		jobs        Jobs
		enabled     bool
		provisioned map[naming.Path]bool

		wg sync.WaitGroup
	}

	Jobs map[string]Job
	Job  struct {
		Queued   time.Time
		schedule schedule.Entry
		cancel   func()
	}
	eventJobDone struct {
		schedule schedule.Entry
		begin    time.Time
		end      time.Time
		err      error
	}
)

var (
	incompatibleNodeMonitorStatus = map[node.MonitorState]any{
		node.MonitorStateZero:        nil,
		node.MonitorStateUpgrade:     nil,
		node.MonitorStateShutting:    nil,
		node.MonitorStateMaintenance: nil,
	}

	// SubscriptionQueueSize is size of "scheduler" subscription
	SubscriptionQueueSize = 16000
)

func New(opts ...funcopt.O) *T {
	t := &T{
		log:         plog.NewDefaultLogger().Attr("pkg", "daemon/scheduler").WithPrefix("daemon: scheduler: "),
		localhost:   hostname.Hostname(),
		events:      make(chan any),
		jobs:        make(Jobs),
		provisioned: make(map[naming.Path]bool),
	}
	if err := funcopt.Apply(t, opts...); err != nil {
		t.log.Errorf("init: %s", err)
		return nil
	}
	return t
}

func entryKey(e schedule.Entry) string {
	return fmt.Sprintf("%s:%s", e.Path, e.Key)
}

func (t Jobs) Add(e schedule.Entry, cancel func()) {
	k := entryKey(e)
	t[k] = Job{
		Queued:   time.Now(),
		schedule: e,
		cancel:   cancel,
	}
}

func (t Jobs) Del(e schedule.Entry) {
	k := entryKey(e)
	job, ok := t[k]
	if !ok {
		return
	}
	job.cancel()
	delete(t, k)
}

func (t Jobs) DelPath(p naming.Path) {
	for _, e := range t {
		if e.schedule.Path != p {
			continue
		}
		t.Del(e.schedule)
	}
}

func (t Jobs) Purge() {
	for k, e := range t {
		e.cancel()
		delete(t, k)
	}
}

func (t *T) createJob(e schedule.Entry) {
	// clean up the existing job
	t.jobs.Del(e)

	if !t.enabled {
		return
	}

	logger := naming.LogWithPath(t.log, e.Path).Attr("action", e.Action).Attr("key", e.Key)
	now := time.Now() // keep before GetNext call
	next, _, err := e.GetNext()
	if err != nil {
		logger.Attr("definition", e.Schedule).Errorf("get next %s %s %s schedule: %s", e.Path, e.Key, e.Action, err)
		t.jobs.Del(e)
		return
	}
	if next.Before(now) {
		t.jobs.Del(e)
		return
	}
	e.NextRunAt = next
	delay := next.Sub(now)
	var obj string
	if e.Path.IsZero() {
		obj = "node"
	} else {
		obj = "object " + e.Path.String()
	}
	logger.Infof("next %s %s at %s (in %s)", obj, e.Key, next, delay)
	tmr := time.AfterFunc(delay, func() {
		begin := time.Now()
		if begin.Sub(next) < 500*time.Millisecond {
			// prevent drift if the gap is small
			begin = next
		}
		if e.RequireCollector && !collector.Alive.Load() {
			logger.Debugf("The collector is not alive")
		} else if err := t.action(e); err != nil {
			logger.Errorf("scheduler: on exec %s %s: %s", obj, e.Key, err)
		}

		// remember last run, to not run the job too soon after a daemon restart
		if err := e.SetLastRun(begin); err != nil {
			logger.Errorf("on update last run %s %s: %s", obj, e.Key, err)
		}

		// remember last success, for users benefit
		if err == nil {
			if err := e.SetLastSuccess(begin); err != nil {
				logger.Errorf("on update last success %s %s: %s", obj, e.Key, err)
			}
		}

		// store end time, for duration sampling
		end := time.Now()

		t.events <- eventJobDone{
			schedule: e,
			begin:    begin,
			end:      end,
			err:      err,
		}
	})
	cancel := func() {
		if tmr == nil {
			return
		}
		tmr.Stop()
	}
	t.jobs.Add(e, cancel)
	return
}

func (t *T) Start(ctx context.Context) error {
	errC := make(chan error)
	t.ctx, t.cancel = context.WithCancel(ctx)

	t.wg.Add(1)
	go func(errC chan<- error) {
		defer t.wg.Done()
		errC <- nil
		t.loop()
	}(errC)

	return <-errC
}

func (t *T) Stop() error {
	t.log.Infof("stopping")
	defer t.log.Infof("stopped")
	t.cancel()
	t.wg.Wait()
	return nil
}

func (t *T) startSubscriptions() *pubsub.Subscription {
	t.pubsub = pubsub.BusFromContext(t.ctx)
	sub := t.pubsub.Sub("scheduler", pubsub.WithQueueSize(SubscriptionQueueSize))
	labelLocalhost := pubsub.Label{"node", t.localhost}
	sub.AddFilter(&msgbus.InstanceConfigUpdated{}, labelLocalhost)
	sub.AddFilter(&msgbus.InstanceStatusDeleted{}, labelLocalhost)
	sub.AddFilter(&msgbus.ObjectStatusDeleted{}, labelLocalhost)
	sub.AddFilter(&msgbus.ObjectStatusUpdated{}, labelLocalhost)
	sub.AddFilter(&msgbus.NodeConfigUpdated{}, labelLocalhost)
	sub.AddFilter(&msgbus.NodeMonitorUpdated{}, labelLocalhost)
	sub.Start()
	return sub
}

func (t *T) loop() {
	t.log.Debugf("loop started")
	t.databus = daemondata.FromContext(t.ctx)
	sub := t.startSubscriptions()
	defer func() {
		if err := sub.Stop(); err != nil {
			t.log.Errorf("subscription stop: %s", err)
		}
	}()

	for {
		select {
		case ev := <-sub.C:
			switch c := ev.(type) {
			case *msgbus.InstanceConfigUpdated:
				t.onInstConfigUpdated(c)
			case *msgbus.InstanceStatusDeleted:
				t.onInstStatusDeleted(c)
			case *msgbus.NodeMonitorUpdated:
				t.onNodeMonitorUpdated(c)
			case *msgbus.NodeConfigUpdated:
				t.onNodeConfigUpdated(c)
			case *msgbus.ObjectStatusUpdated:
				t.onMonObjectStatusUpdated(c)
			}
		case ev := <-t.events:
			switch c := ev.(type) {
			case eventJobDone:
				// remember last run
				c.schedule.LastRunAt = c.begin
				// reschedule
				t.createJob(c.schedule)
			default:
				t.log.Errorf("received an unsupported event: %#v", c)
			}
		case <-t.ctx.Done():
			t.jobs.Purge()
			return
		}
	}
}

func (t *T) onInstStatusDeleted(c *msgbus.InstanceStatusDeleted) {
	t.loggerWithPath(c.Path).Infof("unschedule %s jobs (instance deleted)", c.Path)
	t.unschedule(c.Path)
}

func (t *T) onMonObjectStatusUpdated(c *msgbus.ObjectStatusUpdated) {
	isProvisioned := c.Value.Provisioned.IsOneOf(provisioned.True, provisioned.NotApplicable)
	t.provisioned[c.Path] = isProvisioned
	hasAnyJob := t.hasAnyJob(c.Path)
	switch {
	case isProvisioned && !hasAnyJob:
		t.schedule(c.Path)
	case !isProvisioned && hasAnyJob:
		t.loggerWithPath(c.Path).Infof("unschedule %s jobs (instance no longer provisionned)", c.Path)
		t.unschedule(c.Path)
	}
}

func (t *T) loggerWithPath(p naming.Path) *plog.Logger {
	return naming.LogWithPath(t.log, p)
}

func (t *T) onInstConfigUpdated(c *msgbus.InstanceConfigUpdated) {
	switch {
	case t.enabled:
		t.loggerWithPath(c.Path).Infof("update %s schedules", c.Path)
		t.unschedule(c.Path)
		t.scheduleObject(c.Path)
	}
}

func (t *T) onNodeConfigUpdated(c *msgbus.NodeConfigUpdated) {
	switch {
	case t.enabled:
		t.log.Infof("update node schedules")
		t.unschedule(naming.Path{})
		t.scheduleNode()
	}
}

func (t *T) onNodeMonitorUpdated(c *msgbus.NodeMonitorUpdated) {
	_, incompatible := incompatibleNodeMonitorStatus[c.Value.State]
	switch {
	case incompatible && t.enabled:
		t.log.Infof("disable scheduling (node monitor status is now %s)", c.Value.State)
		t.jobs.Purge()
		t.enabled = false
	case !incompatible && !t.enabled:
		t.log.Infof("enable scheduling (node monitor status is now %s)", c.Value.State)
		t.enabled = true
		t.scheduleAll()
	}
}

func (t *T) hasAnyJob(p naming.Path) bool {
	for _, job := range t.jobs {
		if job.schedule.Path == p {
			return true
		}
	}
	return false
}

func (t *T) scheduleAll() {
	for _, p := range object.StatusData.GetPaths() {
		t.scheduleObject(p)
	}
	t.scheduleNode()
}

func (t *T) schedule(p naming.Path) {
	if !t.enabled {
		return
	}
	if p.IsZero() {
		t.scheduleNode()
	} else {
		t.scheduleObject(p)
	}
}

func (t *T) scheduleNode() {
	o, err := object.NewNode()
	if err != nil {
		t.log.Errorf("schedule node: %s", err)
		return
	}
	for _, e := range o.PrintSchedule() {
		t.createJob(e)
	}
}

func (t *T) scheduleObject(p naming.Path) {
	if isProvisioned, ok := t.provisioned[p]; !ok {
		t.log.Debugf("schedule object %s: provisioned state has not been discovered yet", p)
		return
	} else if !isProvisioned {
		t.log.Infof("schedule object %s: not provisioned", p)
		return
	}
	i, err := object.New(p, object.WithVolatile(true))
	if err != nil {
		t.log.Errorf("schedule object %s: %s", p, err)
		return
	}
	o, ok := i.(object.Actor)
	if !ok {
		// only actor objects have scheduled actions
		return
	}
	for _, e := range o.PrintSchedule() {
		t.createJob(e)
	}
}

func (t *T) unschedule(p naming.Path) {
	t.jobs.DelPath(p)
}
