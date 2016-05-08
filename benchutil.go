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
	"os"
	"testing"
	"time"

	"github.com/mohae/csv2md"
)

type Benchmarker interface {
	Add(Bench)
	Out() error
}

// Benches is a collection of benchmark informtion and their results.
type Benches struct {
	Name       string // Name of the set; optional.
	Desc       string // Description of the collection of benchmarks; optional.
	Note       string // Additional notes about the set; optional.
	Benchmarks []Bench
	nameLen    int // the length of the longest Bench.Name in the set.
	descLen    int // the length of the longest Bench.Desc in the set.
	noteLen    int // the length of the longest Bench.Len in the set.
}

// Add adds a Bench to the slice of Benchmarks
func (b *Benches) Add(bench Bench) {
	b.Benchmarks = append(b.Benchmarks, bench)
}

func (b *Benches) setLens() {
	// Sets the max length of each Bench value.
	for _, v := range b.Benchmarks {
		if len(v.Name) > b.nameLen {
			b.nameLen = len(v.Name)
		}
		if len(v.Desc) > b.descLen {
			b.descLen = len(v.Desc)
		}
		if len(v.Note) > b.noteLen {
			b.noteLen = len(v.Note)
		}
	}
}

// StringBench generates string output from the benchmarks.
type StringBench struct {
	w io.Writer
	Benches
}

func NewStringBench(w io.Writer) *StringBench {
	return &StringBench{w: w}
}

// Out writes the benchmark results.
func (b *StringBench) Out() error {
	b.setLens()
	// If this has a name, output that first.
	if len(b.Name) > 0 {
		fmt.Fprintln(b.w, b.Name)
	}
	// If this has a desc, output that next.
	if len(b.Desc) > 0 {
		fmt.Fprintln(b.w, b.Name)
	}
	for _, v := range b.Benchmarks {
		fmt.Fprintln(b.w, v.txt(b.nameLen, b.descLen, b.noteLen))
	}
	// If this has a note, output that.
	if len(b.Desc) > 0 {
		fmt.Fprintln(b.w, b.Name)
	}
	return nil
}

// CSVBench Benches is a collection of benchmark informtion and their results.
// The output is written as CSV to the writer.  The Name, Desc, and Note
// fields are ignored
type CSVBench struct {
	Benches
	w *csv.Writer
}

func NewCSVBench(w io.Writer) *CSVBench {
	return &CSVBench{w: csv.NewWriter(w)}
}

// Out writes the benchmark results to the writer as strings.
func (b *CSVBench) Out() error {
	return csvOut(b.w, b.Benches)
}

// MDBench Benches is a collection of benchmark informtion and their results.
// The output is written as Markdown to the writer, with the benchmark results
// formatted as a table.
type MDBench struct {
	Benches
	w io.Writer
}

func NewMDBench(w io.Writer) *MDBench {
	return &MDBench{w: w}
}

// Out writes the benchmark results to the writer as a Markdown Table.
func (b *MDBench) Out() error {
	b.setLens()
	// TODO add MDBench Name, Desc, Note handling

	// Generate the CSV
	var buff bytes.Buffer // holds the generated CSV
	w := csv.NewWriter(&buff)
	err := csvOut(w, b.Benches)
	if err != nil {
		return fmt.Errorf("error while creating intermediate CSV: %s", err)
	}
	// then transmogrify to MD
	t := csv2md.NewTransmogrifier(&buff, b.w)
	t.SetFieldAlignment([]string{"l", "r", "r", "r", "r"})
	return t.MDTable()
}

// Bench holds information about a benchmark.
type Bench struct {
	Name   string // Name of the bench.
	Desc   string // Description of the bench; optional.
	Note   string // Additional note about the bench; optional.
	Result        // A map of Result keyed by something.
}

// TXTOutput returns the benchmark information as a slice of strings.
//
// The args exist to ensure consistency in the output layout as what is
// true for this bench may not be true for all benches in the set.
func (b Bench) txt(nameLen, descLen, noteLen int) string {
	var s string
	if nameLen > 0 {
		s = columnL(nameLen+2, b.Name)
	}
	if descLen > 0 {
		s += columnL(descLen+2, b.Desc)
	}
	s += b.Result.String()
	if noteLen > 0 {
		s += b.Note
	}
	return s
}

// CSVOutput returns the benchmark info as []string.
func (b Bench) csv(nameLen, descLen, noteLen int) []string {
	var s []string
	if nameLen > 0 {
		s = append(s, b.Name)
	}
	if descLen > 0 {
		s = append(s, b.Desc)
	}
	s = append(s, b.Result.CSV()...)
	if noteLen > 0 {
		s = append(s, b.Note)
	}
	return s
}

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
	return fmt.Sprintf("%s%s%s%s", columnR(15, r.OpsString()), columnR(15, r.NsOpString()), columnR(18, r.BytesOpString()), columnR(16, r.AllocsOpString()))
}

// CSV returns the benchmark results as []string.
func (r Result) CSV() []string {
	return []string{fmt.Sprintf("%d", r.Ops), fmt.Sprintf("%d", r.NsOp), fmt.Sprintf("%d", r.BytesOp), fmt.Sprintf("%d", r.AllocsOp)}
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

// columnR returns a right justified string of width w.
func columnR(w int, s string) string {
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

// columnL returns a right justified string of width w.
func columnL(w int, s string) string {
	pad := w - len(s)
	if pad < 0 {
		pad = 2
	}
	padding := make([]byte, pad)
	for i := 0; i < pad; i++ {
		padding[i] = 0x20
	}
	return fmt.Sprintf("%s%s", s, string(padding))
}

// Dot prints a . every second to os.StdOut.
func Dot(done chan struct{}) {
	var i int
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		select {
		case <-done:
			return
		case <-t.C:
			i++
			fmt.Fprint(os.Stderr, ".")
			if i%60 == 0 {
				fmt.Fprint(os.Stderr, "\n")
			}
		}
	}
}

// csvOut generates the CSV from a slice of Benches.
func csvOut(w *csv.Writer, benches Benches) error {
	defer w.Flush()
	benches.setLens()
	var line []string
	if benches.nameLen > 0 {
		line = append(line, "Name")
	}
	if benches.descLen > 0 {
		line = append(line, "Description")
	}
	line = append(line, []string{"Operations", "Ns/Op", "Bytes/Op", "Allocs/Op"}...)
	if benches.noteLen > 0 {
		line = append(line, "Note")
	}
	err := w.Write(line)
	if err != nil {
		return err
	}
	for _, v := range benches.Benchmarks {
		line = v.csv(benches.nameLen, benches.descLen, benches.noteLen)
		err := w.Write(line)
		if err != nil {
			return err
		}
	}
	return nil
}
