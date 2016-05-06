// Copyright (c) 2016 Joel Scoble: https://github.com/mohae.  All rights
// reserved.  Licensed under the MIT License. See the LICENSE file in the
// project root for license information.

// Package benchutil contains utilities used for benchmarking and
package benchutil

import (
	"bytes"
	crand "crypto/rand"
	"encoding/csv"
	"fmt"
	"io"
	"math/big"
	"math/rand"
	"testing"

	"github.com/mohae/csv2md"
)

// Result holds information about a benchmark's results.
type Result struct {
	Ops      int64 // the number of operations performed
	NsOp     int64 // The amount of time, in Nanoseconds, per Op.
	BytesOp  int64 // The number of bytes allocated per Op.
	AllocsOp int64 // The number of Allocations per Op.
}

// ResultFromBenchmarkResult creates a Result{} from a testing.BenchmarkResult.
func ResultFromBenchmarkResult(br testing.BenchmarkResult) Result {
	var r Result
	r.Ops = int64(br.N)
	r.NsOp = br.T.Nanoseconds() / r.Ops
	r.BytesOp = int64(br.MemBytes) / r.Ops
	r.AllocsOp = int64(br.MemAllocs) / r.Ops
	return r
}

// OpsString returns the operations performed by the benchmark as a formatted
// string.
func (r Result) OpsString() string {
	return fmt.Sprintf("%d ops", r.Ops)
}

// NsOpString returns the nanoseconds each operation took as a formatted
// string.
func (r Result) NsOpString() string {
	return fmt.Sprintf("%d ns/Op", r.NsOp)
}

// BytesOpString returns the bytes allocated for each operation as a formatted
// string.
func (r Result) BytesOpString() string {
	return fmt.Sprintf("%d bytes/Op", r.BytesOp)
}

// AllocsOpString returns the allocations per operation as a formatted string.
func (r Result) AllocsOpString() string {
	return fmt.Sprintf("%d allocs/Op", r.AllocsOp)
}

func (r Result) String() string {
	return fmt.Sprintf("%s%s%s%s", column(15, r.OpsString()), column(15, r.NsOpString()), column(18, r.BytesOpString()), column(16, r.AllocsOpString()))
}

// CSV returns the benchmark results as []string.
func (r Result) CSV() []string {
	return []string{fmt.Sprintf("%d", r.Ops), fmt.Sprintf("%d", r.NsOp), fmt.Sprintf("%d", r.BytesOp), fmt.Sprintf("%d", r.AllocsOp)}
}

// Bench holds information about a serialization protocol's benchmark.
type Bench struct {
	Name      string            // the name of the bench
	Keys      []string          // a slice of keys, allows for consistent ordering of output
	MaxKeyLen int               // the length of the longest key
	Results   map[string]Result // A map of Result keyed by something.
}

// TXTOutput returns the benchmark information as a slice of strings.
func (b Bench) TXTOutput() []string {
	var out []string
	for _, v := range b.Keys {
		if len(v) > b.MaxKeyLen {
			b.MaxKeyLen = len(v)
		}
		r, ok := b.Results[v]
		if !ok {
			continue
		}
		out = append(out, b.formatOutput(v, r))
	}
	return out
}

func (b Bench) formatOutput(s string, r Result) string {
	return fmt.Sprintf("%s%s%s", column(len(b.Name)+4, b.Name), column(b.MaxKeyLen, s), r.String())
}

// CSVOutput returns the benchmark info as [][]string.
func (b Bench) CSVOutput() [][]string {
	var out [][]string
	for _, v := range b.Keys {
		r, ok := b.Results[v]
		if !ok {
			continue
		}
		tmp := []string{v}
		tmp = append(tmp, r.CSV()...)
		out = append(out, tmp)
	}
	return out
}

const alphanum = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// SeedVal gets a random int64 to use for a seed value
func SeedVal() int64 {
	bi := big.NewInt(1<<63 - 1)
	r, err := crand.Int(crand.Reader, bi)
	if err != nil {
		panic(fmt.Sprintf("entropy read error: %s\n", err))
	}
	return (r.Int64())
}

// RandString returns a randomly generated string of length l.
func RandString(l int) string {
	return string(RandBytes(l))
}

// RandBytes returns a randomly generated []byte of length l.  The values of
// these bytes are restricted to the ASCII alphanum range; that doesn't matter
// for the purposes of these benchmarks.
func RandBytes(l int) []byte {
	b := make([]byte, l)
	for i := 0; i < l; i++ {
		b[i] = alphanum[rand.Intn(len(alphanum))]
	}
	return b
}

// RandBool returns a pseudo-random bool value.
func RandBool() bool {
	if rand.Int31()%2 == 0 {
		return false
	}
	return true
}

// column returns a right justified string of width w.
func column(w int, s string) string {
	pad := w - len(s)
	if pad < 0 {
		pad = 2
	}
	padding := make([]byte, pad)
	for i := 0; i < pad; i++ {
		padding[i] = 0x20
	}
	return fmt.Sprintf("%s%s", string(padding), s)
}

// TXTOut writes the benchmark results to the writer as strings.
func TXTOut(w io.Writer, benchResults []Bench) {
	for _, v := range benchResults {
		lines := v.TXTOutput()
		for _, line := range lines {
			fmt.Fprintln(w, line)
		}
	}
}

// CSVOut writes the benchmark results to the writer as CSV.
func CSVOut(w io.Writer, benchResults []Bench) error {
	wr := csv.NewWriter(w)
	defer wr.Flush()
	// first write out the header
	err := wr.Write([]string{"Protocol", "Operation", "Data Type", "Operations", "Ns/Op", "Bytes/Op", "Allocs/Op"})
	if err != nil {
		return err
	}
	for _, bench := range benchResults {
		lines := bench.CSVOutput()
		for _, line := range lines {
			err := wr.Write(line)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// MDOut writes the benchmark results to the writer as a Markdown Table.
func MDOut(w io.Writer, benchResults []Bench) error {
	var buff bytes.Buffer
	// first generate the csv
	err := CSVOut(&buff, benchResults)
	if err != nil {
		return fmt.Errorf("error while creating intermediate CSV: %s", err)
	}
	// then transmogrify to MD
	t := csv2md.NewTransmogrifier(&buff, w)
	t.SetFieldAlignment([]string{"l", "l", "l", "r", "r", "r", "r"})
	return t.MDTable()
}
