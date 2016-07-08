# benchutil
[![GoDoc](https://godoc.org/github.com/mohae/benchutil?status.svg)](https://godoc.org/github.com/mohae/benchutil)[![Build Status](https://travis-ci.org/mohae/benchutil.png)](https://travis-ci.org/mohae/benchutil)
Helper library to generate formatted output from `testing.BenchmarkResult` data.

Supported formats:
* text (default)
* CSV
* Markdown; results are formatted as a table

Benchmark results can be labeled by providing a name.  Additional information for the benchmark can be added through the description and notes fields.  Related benchmarks can be labeled by providing a group (grouping of groups is not done, the output is in the same order as they were added.)

Groups can be separated out to their own sections.  For `markdown` output, these sections can be created as their own table, and, optionally, the table can use the group identifier as its label, which results in the group column being omitted from the table.
