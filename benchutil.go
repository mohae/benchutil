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

type length struct {
	Group int // the length of the longest Bench.Group in the set.
	Name  int // the length of the longest Bench.Name in the set.
	Desc  int // the length of the longest Bench.Desc in the set.
	Note  int // the length of the longest Bench.Len in the set.
}
type Benchmarker interface {
	Add(Bench)
	Out() error
	SectionPerGroup(bool)
	SectionHeaders(bool)
}

// Benches is a collection of benchmark informtion and their results.
type Benches struct {
	Name            string // Name of the set; optional.
	Desc            string // Description of the collection of benchmarks; optional.
	Note            string // Additional notes about the set; optional.
	Benchmarks      []Bench
	sectionPerGroup bool // make a section for each group
	sectionHeaders  bool // if each section should have it's own col headers, when applicable
	length
}

// Add adds a Bench to the slice of Benchmarks
func (b *Benches) Add(bench Bench) {
	b.Benchmarks = append(b.Benchmarks, bench)
}

// Sets the sectionPerGroup bool
func (b *Benches) SectionPerGroup(v bool) {
	b.sectionPerGroup = v
}

// Sets the sectionHeaders bool.  Txt output ignores this.
func (b *Benches) SectionHeaders(v bool) {
	b.sectionHeaders = v
}

func (b *Benches) setLength() {
	// Sets the max length of each Bench value.
	for _, v := range b.Benchmarks {
		if len(v.Group) > b.length.Group {
			b.length.Group = len(v.Group)
		}
		if len(v.Name) > b.length.Name {
			b.length.Name = len(v.Name)
		}
		if len(v.Desc) > b.length.Desc {
			b.length.Desc = len(v.Desc)
		}
		if len(v.Note) > b.length.Note {
			b.length.Note = len(v.Note)
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
	b.setLength()
	if len(b.Name) > 0 {
		fmt.Fprintln(b.w, b.Name)
	}
	// If this has a desc, output that next.
	if len(b.Desc) > 0 {
		fmt.Fprintln(b.w, b.Name)
	}
	// set it so that the first section doesn't get an extraneous line break.
	priorGroup := b.Benchmarks[0].Group
	for _, v := range b.Benchmarks {
		if v.Group != priorGroup && b.sectionPerGroup {
			fmt.Fprintln(b.w, "")
		}
		fmt.Fprintln(b.w, v.txt(b.length))
		priorGroup = v.Group
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
	w                  io.Writer
	GroupAsSectionName bool
}

func NewMDBench(w io.Writer) *MDBench {
	return &MDBench{w: w}
}

// Out writes the benchmark results to the writer as a Markdown Table.
func (b *MDBench) Out() error {
	b.setLength()
	// Each section may end up as it's own table so we really have a slice
	// of csv, e.g. [][][]string
	// build the alignment & header row
	var hdr, align []string
	// Don't add a group column if groups aren't used or if the group is used as section name
	// and output is being split into sections.
	if b.length.Group > 0 && !b.sectionPerGroup && !b.sectionHeaders && !b.GroupAsSectionName {
		align = append(align, "l")
		hdr = append(hdr, "Group")
	}
	if b.length.Name > 0 {
		align = append(align, "l")
		hdr = append(hdr, "Name")
	}
	align = append(align, []string{"r", "r", "r", "r"}...)
	hdr = append(hdr, []string{"Ops", "ns/Op", "Bytes/Op", "Allocs/Op"}...)
	if b.length.Note > 0 {
		align = append(align, "l")
		hdr = append(hdr, "Note")
	}
	empty := make([]string, len(hdr))
	// get a csv writer
	var buff bytes.Buffer // holds the generated CSV
	w := csv.NewWriter(&buff)
	// Generate the CSV:
	t := csv2md.NewTransmogrifier(&buff, b.w)
	t.HasHeaderRecord = false
	t.SetFieldAlignment(align)
	t.SetFieldNames(hdr)
	priorGroup := b.Benchmarks[0].Group
	// SectionName figures out if it should be written
	err := b.SectionName(priorGroup)
	if err != nil {
		return err
	}
	for _, v := range b.Benchmarks {
		if priorGroup != v.Group && b.sectionPerGroup {
			// if each section doesn't get it's own header row, just add an
			// empty row instead of creating a new table
			if !b.sectionHeaders {
				if b.GroupAsSectionName {
					empty[0] = priorGroup
				}
				err = w.Write(empty)
				if err != nil {
					return err
				}
			}
			// Get a markdown table transmogrifier and configure
			w.Flush()
			err = t.MDTable()
			if err != nil {
				return err
			}
			_, err = b.w.Write([]byte{'\n'})
			if err != nil {
				return err
			}
			buff.Reset()
			// SectionName figures out if it should be written
			err = b.SectionName(v.Group)
			if err != nil {
				return err
			}
		}
		line := v.csv(b.length)
		if b.sectionPerGroup && b.sectionHeaders && b.GroupAsSectionName {
			line = line[1:]
		}
		err = w.Write(line)
		if err != nil {
			return err
		}
		priorGroup = v.Group
	}
	w.Flush()
	return t.MDTable()
}

func (b *MDBench) SectionName(s string) error {
	// see if SectionName is being used.
	if !b.GroupAsSectionName || !b.sectionPerGroup || !b.sectionHeaders {
		return nil
	}
	// If output is in sections and Group is being used as section name;
	// write out the current group
	_, err := b.w.Write([]byte(s + "  "))
	if err != nil {
		return err
	}
	_, err = b.w.Write([]byte{'\n'})
	return err
}

// Bench holds information about a benchmark.  If there is a value for Group,
// the output will have a break between the groups.
type Bench struct {
	Group  string // the Grouping of benchmarks this bench belongs to.
	Name   string // Name of the bench.
	Desc   string // Description of the bench; optional.
	Note   string // Additional note about the bench; optional.
	Result        // A map of Result keyed by something.
}

func NewBench(s string) Bench {
	return Bench{Name: s}
}

// TXTOutput returns the benchmark information as a slice of strings.
//
// The args exist to ensure consistency in the output layout as what is
// true for this bench may not be true for all benches in the set.
func (b Bench) txt(lens length) string {
	var s string
	if lens.Group > 0 {
		s = columnL(lens.Group+2, b.Group)
	}
	if lens.Name > 0 {
		s += columnL(lens.Name+2, b.Name)
	}
	if lens.Desc > 0 {
		s += columnL(lens.Desc+2, b.Desc)
	}
	s += b.Result.String()
	if lens.Note > 0 {
		s += b.Note
	}
	return s
}

// CSVOutput returns the benchmark info as []string.
func (b Bench) csv(lens length) []string {
	var s []string
	if lens.Group > 0 {
		s = append(s, b.Group)
	}
	if lens.Name > 0 {
		s = append(s, b.Name)
	}
	if lens.Desc > 0 {
		s = append(s, b.Desc)
	}
	s = append(s, b.Result.CSV()...)
	if lens.Note > 0 {
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
	benches.setLength()
	var hdr []string
	if benches.length.Group > 0 {
		hdr = append(hdr, "Group")
	}
	if benches.length.Name > 0 {
		hdr = append(hdr, "Name")
	}
	if benches.length.Desc > 0 {
		hdr = append(hdr, "Description")
	}
	hdr = append(hdr, []string{"Operations", "Ns/Op", "Bytes/Op", "Allocs/Op"}...)
	if benches.length.Note > 0 {
		hdr = append(hdr, "Note")
	}
	err := w.Write(hdr)
	if err != nil {
		return err
	}
	var empty []string
	// if there are sections, make a slice for the empty line between sections
	if benches.sectionPerGroup {
		empty = make([]string, len(hdr))
	}
	// set it so that the first section doesn't get an extraneous line break.
	priorGroup := benches.Benchmarks[0].Group
	for _, v := range benches.Benchmarks {
		if v.Group != priorGroup && benches.sectionPerGroup {
			err := w.Write(empty)
			if err != nil {
				return err
			}
			if benches.sectionHeaders {
				err := w.Write(hdr)
				if err != nil {
					return err
				}
			}
		}
		err := w.Write(v.csv(benches.length))
		if err != nil {
			return err
		}
		priorGroup = v.Group
	}
	return nil
}
