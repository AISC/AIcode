// Copyright 2024 The Aisc Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package storer

import (
	"context"
	"fmt"
	"time"

	storage "github.com/aisc/pkg/storage"
	"github.com/aisc/pkg/storer/internal"
	pinstore "github.com/aisc/pkg/storer/internal/pinning"
	"github.com/aisc/pkg/ aisc"
)

// NewCollection is the implementation of the PinStore.NewCollection method.
func (db *DB) NewCollection(ctx context.Context) (PutterSession, error) {
	var (
		pinningPutter internal.PutterCloserWithReference
		err           error
	)
	err = db.Execute(ctx, func(txnRepo internal.Storage) error {
		pinningPutter, err = pinstore.NewCollection(txnRepo)
		if err != nil {
			return fmt.Errorf("pinstore.NewCollection: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &putterSession{
		Putter: putterWithMetrics{
			storage.PutterFunc(
				func(ctx context.Context, chunk  aisc.Chunk) error {
					return db.Execute(ctx, func(s internal.Storage) error {
						return pinningPutter.Put(ctx, s, s.IndexStore(), chunk)
					})
				},
			),
			db.metrics,
			"pinstore",
		},
		done: func(address  aisc.Address) error {
			return db.Execute(ctx, func(s internal.Storage) error {
				return pinningPutter.Close(s, s.IndexStore(), address)
			})
		},
		cleanup: func() error {
			return pinningPutter.Cleanup(db)
		},
	}, nil
}

// DeletePin is the implementation of the PinStore.DeletePin method.
func (db *DB) DeletePin(ctx context.Context, root  aisc.Address) (err error) {
	dur := captureDuration(time.Now())
	defer func() {
		db.metrics.MethodCallsDuration.WithLabelValues("pinstore", "DeletePin").Observe(dur())
		if err == nil {
			db.metrics.MethodCalls.WithLabelValues("pinstore", "DeletePin", "success").Inc()
		} else {
			db.metrics.MethodCalls.WithLabelValues("pinstore", "DeletePin", "failure").Inc()
		}
	}()

	return pinstore.DeletePin(ctx, db, root)
}

// Pins is the implementation of the PinStore.Pins method.
func (db *DB) Pins() (address [] aisc.Address, err error) {
	dur := captureDuration(time.Now())
	defer func() {
		db.metrics.MethodCallsDuration.WithLabelValues("pinstore", "Pins").Observe(dur())
		if err == nil {
			db.metrics.MethodCalls.WithLabelValues("pinstore", "Pins", "success").Inc()
		} else {
			db.metrics.MethodCalls.WithLabelValues("pinstore", "Pins", "failure").Inc()
		}
	}()

	return pinstore.Pins(db.repo.IndexStore())
}

// HasPin is the implementation of the PinStore.HasPin method.
func (db *DB) HasPin(root  aisc.Address) (has bool, err error) {
	dur := captureDuration(time.Now())
	defer func() {
		db.metrics.MethodCallsDuration.WithLabelValues("pinstore", "HasPin").Observe(dur())
		if err == nil {
			db.metrics.MethodCalls.WithLabelValues("pinstore", "HasPin", "success").Inc()
		} else {
			db.metrics.MethodCalls.WithLabelValues("pinstore", "HasPin", "failure").Inc()
		}
	}()

	return pinstore.HasPin(db.repo.IndexStore(), root)
}

func (db *DB) IteratePinCollection(root  aisc.Address, iterateFn func( aisc.Address) (bool, error)) error {
	return pinstore.IterateCollection(db.repo.IndexStore(), root, iterateFn)
}
