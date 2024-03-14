// Copyright 2024 The Aisc Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package  aisc_test

import (
	"testing"

	"github.com/aisc/pkg/ aisc"
)

type distanceTest struct {
	x       aisc.Address
	y       aisc.Address
	result string
}

type distanceCmpTest struct {
	a       aisc.Address
	x       aisc.Address
	y       aisc.Address
	result int
}

var (
	distanceTests = []distanceTest{
		{
			x:       aisc.MustParseHexAddress("9100000000000000000000000000000000000000000000000000000000000000"),
			y:       aisc.MustParseHexAddress("8200000000000000000000000000000000000000000000000000000000000000"),
			result: "8593944123082061379093159043613555660984881674403010612303492563087302590464",
		},
	}

	distanceCmpTests = []distanceCmpTest{
		{
			a:       aisc.MustParseHexAddress("9100000000000000000000000000000000000000000000000000000000000000"), // 10010001
			x:       aisc.MustParseHexAddress("8200000000000000000000000000000000000000000000000000000000000000"), // 10000010 xor(0x91,0x82) =	19
			y:       aisc.MustParseHexAddress("1200000000000000000000000000000000000000000000000000000000000000"), // 00010010 xor(0x91,0x12) = 131
			result: 1,
		},
		{
			a:       aisc.MustParseHexAddress("9100000000000000000000000000000000000000000000000000000000000000"),
			x:       aisc.MustParseHexAddress("1200000000000000000000000000000000000000000000000000000000000000"),
			y:       aisc.MustParseHexAddress("8200000000000000000000000000000000000000000000000000000000000000"),
			result: -1,
		},
		{
			a:       aisc.MustParseHexAddress("9100000000000000000000000000000000000000000000000000000000000000"),
			x:       aisc.MustParseHexAddress("1200000000000000000000000000000000000000000000000000000000000000"),
			y:       aisc.MustParseHexAddress("1200000000000000000000000000000000000000000000000000000000000000"),
			result: 0,
		},
	}
)

// TestDistance tests the correctness of the distance calculation.
func TestDistance(t *testing.T) {
	t.Parallel()

	for _, dt := range distanceTests {
		distance, err :=  aisc.Distance(dt.x, dt.y)
		if err != nil {
			t.Fatal(err)
		}
		if distance.String() != dt.result {
			t.Fatalf("incorrect distance, expected %s, got %s (x: %x, y: %x)", dt.result, distance.String(), dt.x, dt.y)
		}
	}
}

// TestDistanceCmp tests the distance comparison method.
func TestDistanceCmp(t *testing.T) {
	t.Parallel()

	for _, dt := range distanceCmpTests {
		direction, err :=  aisc.DistanceCmp(dt.a, dt.x, dt.y)
		if err != nil {
			t.Fatal(err)
		}
		if direction != dt.result {
			t.Fatalf("incorrect distance compare, expected %d, got %d (a: %x, x: %x, y: %x)", dt.result, direction, dt.a, dt.x, dt.y)
		}
	}
}
