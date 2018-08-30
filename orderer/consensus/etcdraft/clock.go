/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package etcdraft

import (
	"time"

	"code.cloudfoundry.org/clock"
)

// timer hides away clock management.
// this is necessary mostly because of channel draining
// on stopped timer, see time.Timer for more details.

// this is not thread safe
type timer struct {
	// this is used to distinguish two conditions under which timer.Stop() returns false:
	// - timer already expired (ticking == true)
	// - timer was already stopped (ticking == false)
	ticking bool

	timer clock.Timer
}

// newTimer returns a stopped timer.
// we cannot use nil timer because main loop will be
// select-waiting on timer.C(), nil timer would panic.
func newTimer(c clock.Clock) *timer {
	t := c.NewTimer(0)
	if !t.Stop() {
		<-t.C()
	}

	return &timer{
		ticking: false,
		timer:   t,
	}
}

// if timer is already started, this is no-op
func (t *timer) start(d time.Duration) {
	if !t.ticking {
		t.ticking = true
		t.timer.Reset(d)
	}
}

func (t *timer) stop() {
	if !t.timer.Stop() && t.ticking {
		// we only need to drain channel if timer expired (not explicitly stopped)
		<-t.timer.C()
	}
	t.ticking = false
}

// this must be called after a signal has been received on c()
func (t *timer) reset() {
	t.ticking = false
}

// caller MUST call reset() after receiving a signal
// on this channel.
func (t *timer) c() <-chan time.Time {
	return t.timer.C()
}
