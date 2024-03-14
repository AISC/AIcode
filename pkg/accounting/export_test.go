// Copyright 2021 The Aisc Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package accounting

import (
	"time"

	"github.com/aisc/pkg/ aisc"
)

func (a *Accounting) SetTimeNow(f func() time.Time) {
	a.timeNow = f
}

func (a *Accounting) SetTime(k int64) {
	a.SetTimeNow(func() time.Time {
		return time.Unix(k, 0)
	})
}

func (a *Accounting) IsPaymentOngoing(peer  aisc.Address) bool {
	return a.getAccountingPeer(peer).paymentOngoing
}
