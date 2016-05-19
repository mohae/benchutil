// Copyright (c) 2016 Joel Scoble: https://github.com/mohae.  All rights
// reserved.  Licensed under the MIT License. See the LICENSE file in the
// project root for license information.

package benchutil

import "testing"

func TestSystemInfo(t *testing.T) {
	b := Benches{}
	s, err := b.SystemInfo()
	if err != nil {
		t.Errorf("unexpected error: %s", err)
		return
	}
	if s == "" {
		t.Error("got empty string; want a non-empty string")
	}
	t.Logf(s)
}

func TestAppend(t *testing.T) {
	b := Benches{}
	bench := Bench{}
	b.Append(bench)
	if len(b.Benchmarks) != 1 {
		t.Errorf("expected Benchmarks len to be 1; got %d", len(b.Benchmarks))
	}
	benches := []Bench{Bench{}, Bench{}}
	b.Append(benches...)
	if len(b.Benchmarks) != 3 {
		t.Errorf("expected Benchmarks len to be 3; got %d", len(b.Benchmarks))
	}
}
