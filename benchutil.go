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
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	pcg "github.com/dgryski/go-pcgr"
	human "github.com/dustin/go-humanize"
	"github.com/mohae/csv2md"
	"github.com/mohae/joefriday/cpu/facts"
	"github.com/mohae/joefriday/mem"
	"github.com/mohae/joefriday/platform/kernel"
	"github.com/mohae/joefriday/platform/release"
	//"github.com/mohae/joefriday/sysinfo/mem"
)

const defaultPadding = 2

var prng pcg.Rand

func init() {
	prng.Seed(NewSeed())
}

// Benchmarker defines common behavior for a Benchmark output harness; format
// specific methods may be
type Benchmarker interface {
	Append(...Bench)
	Out() error
	IncludeOpsColumnDesc(bool)
	IncludeSystemInfo(bool)
	IncludeDetailedSystemInfo(bool)
	SystemInfo() (string, error)
	DetailedSystemInfo() (string, error)
	SetGroupColumnHeader(s string)
	SetSubGroupColumnHeader(s string)
	SetNameColumnHeader(s string)
	SetDescColumnHeader(s string)
	SetOpsColumnHeader(s string)
	SetNsOpColumnHeader(s string)
	SetBytesOpColumnHeader(s string)
	SetAllocsOpColumnHeader(s string)
	SetNoteColumnHeader(s string)
	SetColumnPadding(i int)
	SectionPerGroup(bool)
	SectionHeaders(bool)
	NameSections(bool)
}

type header struct {
	Group    string
	SubGroup string
	Name     string
	Desc     string
	Ops      string
	NsOp     string
	BytesOp  string
	AllocsOp string
	Note     string
}

func newHeader() header {
	return header{
		Group:    "Group",
		SubGroup: "Sub-Group",
		Name:     "Name",
		Desc:     "Desc",
		Ops:      "Ops",
		NsOp:     "ns/Op",
		BytesOp:  "B/Op",
		AllocsOp: "Allocs/Op",
		Note:     "Note",
	}
}

// SetGroupColumnHeader sets the Group column header; default is 'Group'.
// This only applies when Group is part of the output.
func (h *header) SetGroupColumnHeader(s string) {
	h.Group = s
}

// SetSubGroupColumnHeader sets the SubGroup column header; default is
// 'Sub-Group'.  This only applies when SubGroup is part of the output.
func (h *header) SetSubGroupColumnHeader(s string) {
	h.SubGroup = s
}

// SetNameColumnHeader sets the Name column header; default is 'Name'.
func (h *header) SetNameColumnHeader(s string) {
	h.Name = s
}

// SetDescColumnHeader sets the Desc column header; default is 'Desc'.  This
// only applies when Desc is part of the output.
func (h *header) SetDescColumnHeader(s string) {
	h.Desc = s
}

// SetOpsColumnHeader sets the Ops column header; default is 'Ops'.  This only
// applies when Ops is part of the output.
func (h *header) SetOpsColumnHeader(s string) {
	h.Ops = s
}

// SetNsOpColumnHeader sets the NsOp column header; default is 'ns/Op'.  This
// only applies when NsOp is part of the output.
func (h *header) SetNsOpColumnHeader(s string) {
	h.NsOp = s
}

// SetBytesOpColumnHeader sets the BytesOp column header; default is 'B/Op'.
// This only applies when BytesOp is part of the output.
func (h *header) SetBytesOpColumnHeader(s string) {
	h.BytesOp = s
}

// SetAllocsOpColumnHeader sets the AllocsOp column header; default is
// 'Allocs/Op'.  This only applies when AllocsOp is part of the output.
func (h *header) SetAllocsOpColumnHeader(s string) {
	h.AllocsOp = s
}

// SetNoteColumnHeader sets the Note column header; default is 'Note'.  This
// only applies when Note is part of the output.
func (h *header) SetNoteColumnHeader(s string) {
	h.Note = s
}

// Benches is a collection of benchmark informtion and their results.
type Benches struct {
	Name       string  // Name of the set; optional.
	Desc       string  // Description of the collection of benchmarks; optional.
	Note       string  // Additional notes about the set; optional.
	Benchmarks []Bench // The benchmark results
	header
	columnPadding             int  // The number of spaces between columns.
	includeOpsColumnDesc      bool // Include the description of the ops info in each column's result output.
	includeSystemInfo         bool // Add basic system info to the output
	includeDetailedSystemInfo bool // SystemInfo output uses DetailedSystemInfo.
	sectionPerGroup           bool // make a section for each group
	sectionHeaders            bool // if each section should have it's own col headers, when applicable
	nameSections              bool // Use the group name as the section name when there are sections.
	length
}

// DetailedSystemInfo generates the System Information string, including
// information about every CPU core on the system.
func (b *Benches) DetailedSystemInfo() (string, error) {
	inf, err := facts.Get()
	if err != nil {
		return "", err
	}
	k, err := kernel.Get()
	if err != nil {
		return "", err
	}
	r, err := release.Get()
	if err != nil {
		return "", err
	}
	/* TODO value returned by sysinfo is > actual system mem, why?
	var m mem.Info
	err = m.Get()
	*/
	m, err := mem.Get()
	if err != nil {
		return "", err
	}
	var buff bytes.Buffer
	for _, v := range inf.CPU {
		buff.WriteString(fmt.Sprintf("Processor:  %d\n", v.Processor))
		buff.WriteString("Model:      ")
		buff.WriteString(v.ModelName)
		buff.WriteRune('\n')
		buff.WriteString(fmt.Sprintf("CPU MHz:    %7.2f\n", v.CPUMHz))
		buff.WriteString("Cache:      ")
		buff.WriteString(v.CacheSize)
		buff.WriteRune('\n')
	}
	buff.WriteString("Memory:     ")
	buff.WriteString(human.Bytes(m.MemTotal))
	buff.WriteRune('\n')
	// release info
	info := r.PrettyName
	if info == "" {
		info = r.Version
		if info == "" {
			info = r.VersionID
		}
	}
	buff.WriteString(fmt.Sprintf("OS:         %s %s\n", strings.Title(r.ID), info))
	// kernel info
	if k.Version != "" {
		buff.WriteString(fmt.Sprintf("Kernel:     %s\n", k.Version))
		buff.WriteRune('\n')
	}
	return buff.String(), nil
}

// SystemInfo generates a System Information string.
func (b *Benches) SystemInfo() (string, error) {
	inf, err := facts.Get()
	if err != nil {
		return "", err
	}
	k, err := kernel.Get()
	if err != nil {
		return "", err
	}
	r, err := release.Get()
	if err != nil {
		return "", err
	}

	m, err := mem.Get()
	if err != nil {
		return "", err
	}
	var buff bytes.Buffer

	buff.WriteString(fmt.Sprintf("Processors:  %d\n", len(inf.CPU)))
	buff.WriteString("Model:      ")
	buff.WriteString(inf.CPU[0].ModelName)
	buff.WriteRune('\n')
	buff.WriteString(fmt.Sprintf("CPU MHz:    %7.2f\n", inf.CPU[0].CPUMHz))
	buff.WriteString("Cache:      ")
	buff.WriteString(inf.CPU[0].CacheSize)
	buff.WriteRune('\n')
	buff.WriteString("Memory:     ")
	buff.WriteString(human.Bytes(m.MemTotal))
	buff.WriteRune('\n')
	// release info
	info := r.PrettyName
	if info == "" {
		info = r.Version
		if info == "" {
			info = r.VersionID
		}
	}
	buff.WriteString(fmt.Sprintf("OS:         %s %s\n", strings.Title(r.ID), info))
	// kernel info
	if k.Version != "" {
		buff.WriteString(fmt.Sprintf("Kernel:     %s\n", k.Version))
		buff.WriteRune('\n')
	}
	return buff.String(), nil
}

// Add adds a Bench to the slice of Benchmarks
func (b *Benches) Append(benches ...Bench) {
	b.Benchmarks = append(b.Benchmarks, benches...)
}

// IncludeOpsColumnDesc: if true, the ops information will be included in each
// ops column's result.
func (b *Benches) IncludeOpsColumnDesc(v bool) {
	b.includeOpsColumnDesc = v
}

// IncludeSystemInfo: if true, basic system info will be included in the
// benchmarker's output.  If both IncludeSystemInfo and
// IncludeDetailedSystemInfo are set to true, the detailed system info will
// be included.
func (b *Benches) IncludeSystemInfo(v bool) {
	b.includeSystemInfo = v
}

// DetailedSystemInfoOutput: if true, detailed system info will be included in
// the benchmarker's output.  If both IncludeSystemInfo and
// IncludeDetailedSystemInfo are set to true, the detailed system info will
// be included.
func (b *Benches) IncludeDetailedSystemInfo(v bool) {
	b.includeDetailedSystemInfo = v
}

// Sets the sectionPerGroup bool
func (b *Benches) SectionPerGroup(v bool) {
	b.sectionPerGroup = v
}

// Sets the sectionHeaders bool.  Txt output ignores this.
func (b *Benches) SectionHeaders(v bool) {
	b.sectionHeaders = v
}

// Sets the nameSections bool.  Txt output ignores this.
func (b *Benches) NameSections(v bool) {
	b.nameSections = v
}

// Sets the number of spaces between columns; default is 2.
func (b *Benches) SetColumnPadding(i int) {
	b.columnPadding = i
}

func (b *Benches) setLength() {
	// Sets the max length of each Bench value.
	var maxIters int64
	// find the longest value in all of the benchmarks
	for _, v := range b.Benchmarks {
		if len(v.Group) > b.length.Group {
			b.length.Group = len(v.Group)
		}
		if len(v.SubGroup) > b.length.SubGroup {
			b.length.SubGroup = len(v.SubGroup)
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
		// result
		if len(strconv.Itoa(int(v.Result.Ops)*v.Iterations)) > b.length.Ops {
			b.length.Ops = len(strconv.Itoa(int(v.Result.Ops) * v.Iterations))
		}
		// if each result represents more than 1 iteration; store the
		// benches value if it's greater than the current value.
		if (v.Result.Ops * int64(v.Iterations)) > maxIters {
			maxIters = v.Result.Ops * int64(v.Iterations)
		}
		if len(strconv.Itoa(int(v.Result.NsOp))) > b.length.NsOp {
			b.length.NsOp = len(strconv.Itoa(int(v.Result.NsOp)))
		}
		if len(strconv.Itoa(int(v.Result.BytesOp))) > b.length.BytesOp {
			b.length.BytesOp = len(strconv.Itoa(int(v.Result.BytesOp)))
		}
		if len(strconv.Itoa(int(v.Result.AllocsOp))) > b.length.AllocsOp {
			b.length.AllocsOp = len(strconv.Itoa(int(v.Result.AllocsOp)))
		}
	}
	// if the ops desc is going to be included in each ops row/column; add that length
	if b.includeOpsColumnDesc {
		b.length.NsOp += 6
		b.length.BytesOp += 9
		b.length.AllocsOp += 10
	}
	// see if the header column values are > than the contents they hold
	if b.length.Group > 0 && len(b.header.Group) > b.length.Group {
		b.length.Group = len(b.header.Group)
	}
	if b.length.SubGroup > 0 && len(b.header.SubGroup) > b.length.SubGroup {
		b.length.SubGroup = len(b.header.SubGroup)
	}
	if b.length.Name > 0 && len(b.header.Name) > b.length.Name {
		b.length.Name = len(b.header.Name)
	}
	if b.length.Desc > 0 && len(b.header.Desc) > b.length.Desc {
		b.length.Desc = len(b.header.Desc)
	}
	if b.length.Note > 0 && len(b.header.Note) > b.length.Note {
		b.length.Note = len(b.header.Note)
	}
	if len(b.header.Ops) > b.length.Ops {
		b.length.Ops = len(b.header.Ops)
	}
	if len(b.header.NsOp) > b.length.NsOp {
		b.length.NsOp = len(b.header.NsOp)
	}
	if len(b.header.BytesOp) > b.length.BytesOp {
		b.length.BytesOp = len(b.header.BytesOp)
	}
	if len(b.header.AllocsOp) > b.length.AllocsOp {
		b.length.AllocsOp = len(b.header.AllocsOp)
	}
}

// OpsString returns the operations performed by the benchmark as a formatted
// string.
func (b *Benches) OpsString(v Bench) string {
	if b.includeOpsColumnDesc {
		return fmt.Sprintf("%d ops", v.Ops*int64(v.Iterations))
	}
	return fmt.Sprintf("%d", v.Ops*int64(v.Iterations))
}

// NsOpString returns the nanoseconds each operation took as a formatted
// string.
func (b *Benches) NsOpString(v Bench) string {
	if b.includeOpsColumnDesc {
		return fmt.Sprintf("%s ns/op", b.perOpsString(v.NsOp, v.Iterations))
	}
	return b.perOpsString(v.NsOp, v.Iterations)
}

// BytesOpString returns the bytes allocated for each operation as a formatted
// string.
func (b *Benches) BytesOpString(v Bench) string {
	if b.includeOpsColumnDesc {
		return fmt.Sprintf("%s bytes/op", b.perOpsString(v.BytesOp, v.Iterations))
	}
	return b.perOpsString(v.BytesOp, v.Iterations)
}

// AllocsOpString returns the allocations per operation as a formatted string.
func (b *Benches) AllocsOpString(v Bench) string {
	if b.includeOpsColumnDesc {
		return fmt.Sprintf("%s allocs/op", b.perOpsString(v.AllocsOp, v.Iterations))
	}
	return b.perOpsString(v.AllocsOp, v.Iterations)
}

// perOpsString takes a value and uses it to calculate the per operation value,
// which is returned as a string.
func (b *Benches) perOpsString(v int64, it int) string {
	if v == 0 {
		return "0"
	}
	return fmt.Sprintf("%d", v/int64(it))
}

// columnR returns a right justified string of width w.
func (b *Benches) columnR(w int, s string) string {
	pad := w - len(s)
	if pad < 0 {
		pad = 0
	}
	rpadding := make([]byte, pad)
	for i := range rpadding {
		rpadding[i] = 0x20
	}
	lpadding := make([]byte, b.columnPadding)
	for i := range lpadding {
		lpadding[i] = 0x20
	}
	return fmt.Sprintf("%s%s%s", rpadding, s, lpadding)
}

// columnL returns a left justified string of width w.
func (b *Benches) columnL(w int, s string) string {
	pad := w + b.columnPadding - len(s)
	if pad < 0 {
		pad = b.columnPadding
	}
	padding := make([]byte, pad)
	for i := range padding {
		padding[i] = 0x20
	}
	return fmt.Sprintf("%s%s", s, padding)
}

// resultCSV returns the benchmark results as []string.
func (b *Benches) resultCSV(i int) []string {
	return []string{b.OpsString(b.Benchmarks[i]), b.NsOpString(b.Benchmarks[i]), b.BytesOpString(b.Benchmarks[i]), b.AllocsOpString(b.Benchmarks[i])}
}

// csv returns the info of the benchmark at index i as []string.
func (b Benches) csv(i int) []string {
	var s []string
	if b.length.Group > 0 {
		s = append(s, b.Benchmarks[i].Group)
	}
	if b.length.SubGroup > 0 {
		s = append(s, b.Benchmarks[i].SubGroup)
	}
	if b.length.Name > 0 {
		s = append(s, b.Benchmarks[i].Name)
	}
	if b.length.Desc > 0 {
		s = append(s, b.Benchmarks[i].Desc)
	}
	s = append(s, b.resultCSV(i)...)
	if b.length.Note > 0 {
		s = append(s, b.Benchmarks[i].Note)
	}
	return s
}

// StringBench generates string output from the benchmarks.
type StringBench struct {
	w io.Writer
	Benches
}

func NewStringBench(w io.Writer) *StringBench {
	return &StringBench{
		w: w,
		Benches: Benches{
			header:        newHeader(),
			columnPadding: defaultPadding,
		},
	}
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
	// If systeminfo is included, include it.
	if b.includeDetailedSystemInfo {
		inf, err := b.SystemInfo()
		if err != nil {
			return err
		}
		fmt.Fprintln(b.w, inf)
		goto writeTable
	}
	if b.includeSystemInfo {
		inf, err := b.SystemInfo()
		if err != nil {
			return err
		}
		fmt.Fprintln(b.w, inf)
	}
writeTable:

	// Write the headers
	b.WriteHeader()
	// Write the separator line
	b.WriteSeparatorLine()
	// set it so that the first section doesn't get an extraneous line break.
	b.WriteResults()
	// If this has a note, output that.
	if len(b.Desc) > 0 {
		fmt.Fprintln(b.w, b.Name)
	}
	return nil
}

// WriteHeader writes the table header to the writer.
func (b *StringBench) WriteHeader() {
	var buf bytes.Buffer
	if b.length.Group > 0 {
		buf.WriteString(b.columnL(b.length.Group, b.header.Group))
	}
	if b.length.SubGroup > 0 {
		buf.WriteString(b.columnL(b.length.SubGroup, b.header.SubGroup))
	}
	if b.length.Name > 0 {
		buf.WriteString(b.columnL(b.length.Name, b.header.Name))
	}
	if b.length.Desc > 0 {
		buf.WriteString(b.columnL(b.length.Desc, b.header.Desc))
	}
	buf.WriteString(b.columnL(b.length.Ops, b.header.Ops))
	buf.WriteString(b.columnL(b.length.NsOp, b.header.NsOp))
	buf.WriteString(b.columnL(b.length.BytesOp, b.header.BytesOp))
	buf.WriteString(b.columnL(b.length.AllocsOp, b.header.AllocsOp))
	if b.length.Note > 0 {
		buf.WriteString(b.header.Note)
	}
	fmt.Fprintln(b.w, buf.String())
}

// WriteSeparatorLine writes a line consisting of dashes to the writer.
func (b *StringBench) WriteSeparatorLine() {
	var buf bytes.Buffer
	var l int
	if b.length.Group > 0 {
		l = b.length.Group + b.columnPadding
	}
	if b.length.SubGroup > 0 {
		l += b.length.SubGroup + b.columnPadding
	}
	if b.length.Name > 0 {
		l += b.length.Name + b.columnPadding
	}
	if b.length.Desc > 0 {
		l += b.length.Desc + b.columnPadding
	}
	l += b.length.Ops + b.columnPadding
	l += b.length.NsOp + b.columnPadding
	l += b.length.BytesOp + b.columnPadding
	l += b.length.AllocsOp + b.columnPadding
	l += b.length.Note
	for i := 0; i < l; i++ {
		buf.WriteByte('-')
	}
	//buf.WriteRune('\n')
	fmt.Fprintln(b.w, buf.String())
}

// WriteResults writes the benchmark results to the writer.
func (b *StringBench) WriteResults() {
	var buf bytes.Buffer
	priorGroup := b.Benchmarks[0].Group
	for i, bench := range b.Benchmarks {
		buf.Reset()
		if b.sectionPerGroup && bench.Group != priorGroup {
			buf.WriteRune('\n')
		}
		priorGroup = bench.Group

		if b.length.Group > 0 {
			buf.WriteString(b.columnL(b.length.Group, bench.Group))
		}
		if b.length.SubGroup > 0 {
			buf.WriteString(b.columnL(b.length.SubGroup, bench.SubGroup))
		}
		if b.length.Name > 0 {
			buf.WriteString(b.columnL(b.length.Name, bench.Name))
		}
		if b.length.Desc > 0 {
			buf.WriteString(b.columnL(b.length.Desc, bench.Desc))
		}
		buf.WriteString(b.BenchString(i))
		if b.length.Note > 0 {
			buf.WriteString(b.Note)
		}
		fmt.Fprintln(b.w, buf.String())
	}
}

// BenchString generates the Ops, ns/Ops, B/Ops, and Allocs/Op string for a
// given benchmark result.
func (b *StringBench) BenchString(i int) string {
	return fmt.Sprintf("%s%s%s%s", b.columnR(b.length.Ops, b.OpsString(b.Benchmarks[i])), b.columnR(b.length.NsOp, b.NsOpString(b.Benchmarks[i])), b.columnR(b.length.BytesOp, b.BytesOpString(b.Benchmarks[i])), b.columnR(b.length.AllocsOp, b.AllocsOpString(b.Benchmarks[i])))
}

// CSVBench Benches is a collection of benchmark informtion and their results.
// The output is written as CSV to the writer.  The Name, Desc, and Note
// fields are ignored
type CSVBench struct {
	Benches
	w *csv.Writer
}

func NewCSVBench(w io.Writer) *CSVBench {
	return &CSVBench{
		w: csv.NewWriter(w),
		Benches: Benches{
			header:        newHeader(),
			columnPadding: defaultPadding,
		},
	}
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
	SectionHeaderHash  string // the markdown header hash for section names, when applicable
}

func NewMDBench(w io.Writer) *MDBench {
	return &MDBench{
		w: w,
		Benches: Benches{
			header:        newHeader(),
			columnPadding: defaultPadding,
		},
		SectionHeaderHash: "####",
	}
}

// Out writes the benchmark results to the writer as a Markdown Table.
func (b *MDBench) Out() error {
	// If systeminfo is included, include it.
	if b.includeDetailedSystemInfo {
		inf, err := b.SystemInfo()
		if err != nil {
			return err
		}
		fmt.Fprintln(b.w, inf)
		goto output
	}
	if b.includeSystemInfo {
		inf, err := b.SystemInfo()
		if err != nil {
			return err
		}
		fmt.Fprintln(b.w, inf)
	}

output:
	b.setLength()
	// Each section may end up as it's own table so we really have a slice
	// of csv, e.g. [][][]string
	// build the alignment & header row
	var hdr, align []string
	// Don't add a group column if groups aren't used or if the group is used as section name
	// and output is being split into sections.
	if b.length.Group > 0 && !b.sectionPerGroup && !b.sectionHeaders && !b.GroupAsSectionName {
		align = append(align, "l")
		hdr = append(hdr, b.header.Group)
	}
	if b.length.SubGroup > 0 {
		align = append(align, "l")
		hdr = append(hdr, b.header.SubGroup)
	}
	if b.length.Name > 0 {
		align = append(align, "l")
		hdr = append(hdr, b.header.Name)
	}
	if b.length.Desc > 0 {
		align = append(align, "l")
		hdr = append(hdr, b.header.Desc)
	}
	align = append(align, []string{"r", "r", "r", "r"}...)
	hdr = append(hdr, []string{b.header.Ops, b.header.NsOp, b.header.BytesOp, b.header.AllocsOp}...)
	if b.length.Note > 0 {
		align = append(align, "l")
		hdr = append(hdr, b.header.Note)
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
	for i, v := range b.Benchmarks {
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
		line := b.csv(i)
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

// SectionName writes out the name of a new section using the Group as the
// name.  This is only applicable when output is being split up into Groups
// and when the sections are to be named.  By default sections are not named.
func (b *MDBench) SectionName(s string) error {
	// see if SectionName is being used.
	if !b.GroupAsSectionName || !b.sectionPerGroup || !b.sectionHeaders {
		return nil
	}
	// If output is in sections and Group is being used as section name;
	// write out the current group
	_, err := b.w.Write([]byte(b.SectionHeaderHash + s + "  "))
	if err != nil {
		return err
	}
	_, err = b.w.Write([]byte{'\n'})
	return err
}

type length struct {
	Group    int // the length of the longest Bench.Group in the set
	SubGroup int // the length of the longest Bench.Subgroup in the set.
	Name     int // the length of the longest Bench.Name in the set.
	Desc     int // the length of the longest Bench.Desc in the set.
	Ops      int // width of highest ops count in the set.
	NsOp     int // width of the largest ns/op in the set.
	BytesOp  int // width of the largest bytes/op alloc in the set.
	AllocsOp int // width of the largest allocs/op in the set.
	Note     int // the length of the longest Bench.Len in the set.
}

// Bench holds information about a benchmark.  If there is a value for Group,
// the output will have a break between the groups.
type Bench struct {
	Group      string // the Grouping of benchmarks this bench belongs to.
	SubGroup   string // the Sub-Group this bench belongs to; mainly for additional sort options.
	Name       string // Name of the bench.
	Desc       string // Description of the bench; optional.
	Note       string // Additional note about the bench; optional.
	Iterations int    // number of test iterations; default 1
	Result            // A map of Result keyed by something.
}

func NewBench(s string) Bench {
	return Bench{Name: s, Iterations: 1}
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

const alphanum = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

var alen = uint32(len(alphanum))

// NewSeed gets a random int64 to use for a seed value.
func NewSeed() int64 {
	bi := big.NewInt(1<<63 - 1)
	r, err := crand.Int(crand.Reader, bi)
	if err != nil {
		panic(fmt.Sprintf("entropy read error: %s\n", err))
	}
	return (r.Int64())
}

// RandString returns a randomly generated string of length l.
func RandString(l uint32) string {
	return string(RandBytes(l))
}

// RandBytes returns a randomly generated []byte of length l.  The values of
// these bytes are restricted to the ASCII alphanum range; that doesn't matter
// for the purposes of these benchmarks.
func RandBytes(l uint32) []byte {
	b := make([]byte, l)
	for i := 0; i < int(l); i++ {
		b[i] = alphanum[int(prng.Bound(alen))]
	}
	return b
}

// RandBool returns a pseudo-random bool value.
func RandBool() bool {
	if prng.Int63()%2 == 0 {
		return false
	}
	return true
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
	if benches.length.SubGroup > 0 {
		hdr = append(hdr, "SubGroup")
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
	for i, v := range benches.Benchmarks {
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
		err := w.Write(benches.csv(i))
		if err != nil {
			return err
		}
		priorGroup = v.Group
	}
	return nil
}
