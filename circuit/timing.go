//
// Copyright (c) 2020-2026 Markku Rossi
//
// All rights reserved.
//

package circuit

import (
	"fmt"
	"os"
	"time"

	"github.com/markkurossi/mpc/p2p"
	"github.com/markkurossi/tabulate"
)

// Timing records timing samples and renders a profiling report.
type Timing struct {
	Start   time.Time
	Samples []*Sample
}

// NewTiming creates a new Timing instance.
func NewTiming() *Timing {
	return &Timing{
		Start: time.Now(),
	}
}

// Sample adds a timing sample with label and data columns.
func (t *Timing) Sample(label string, cols []string) *Sample {
	start := t.Start
	if len(t.Samples) > 0 {
		start = t.Samples[len(t.Samples)-1].End
	}
	sample := &Sample{
		Label: label,
		Start: start,
		End:   time.Now(),
		Cols:  cols,
	}
	t.Samples = append(t.Samples, sample)
	return sample
}

// AbsSample adds an absolute sample with label and data columns.
func (t *Timing) AbsSample(label string, duration time.Duration,
	cols []string) *Sample {
	sample := &Sample{
		Label: label,
		End:   time.Now(),
		Cols:  cols,
	}
	sample.Start = sample.End.Add(-duration)
	t.Samples = append(t.Samples, sample)
	return sample
}

var ioHeaders = []string{"Onl", "Offl", "Xfer"}

// Print prints profiling report to standard output.
func (t *Timing) Print(stats p2p.IOStats, iostats ...p2p.IOStats) {
	if len(t.Samples) == 0 {
		return
	}

	sent := stats.Sent.Load()
	rcvd := stats.Recvd.Load()
	flushed := stats.Flushed.Load()

	tab := tabulate.New(tabulate.UnicodeLight)
	tab.Header("Op").SetAlign(tabulate.ML)
	tab.Header("Time").SetAlign(tabulate.MR)
	tab.Header("%").SetAlign(tabulate.MR)

	if len(iostats) == 0 {
		tab.Header("Xfer").SetAlign(tabulate.MR)
	} else {
		for i := 0; i <= len(iostats); i++ {
			var hdr string
			if i < len(ioHeaders) {
				hdr = ioHeaders[i]
			} else {
				hdr = fmt.Sprintf("Stat %v", i)
			}
			tab.Header(hdr).SetAlign(tabulate.MR)
		}
	}

	total := t.Samples[len(t.Samples)-1].End.Sub(t.Start)
	for _, sample := range t.Samples {
		row := tab.Row()
		row.Column(sample.Label)

		duration := sample.End.Sub(sample.Start)
		if duration > 0 {
			row.Column(duration.String())
			row.Column(fmt.Sprintf("%.2f%%",
				float64(duration)/float64(total)*100))
		} else {
			row.Column("")
			row.Column("")
		}

		for _, col := range sample.Cols {
			row.Column(col)
		}

		for idx, sub := range sample.Samples {
			row := tab.Row()

			var prefix string
			if idx+1 >= len(sample.Samples) {
				prefix = "\u2570\u2574"
			} else {
				prefix = "\u251C\u2574"
			}

			row.Column(prefix + sub.Label).SetFormat(tabulate.FmtItalic)

			var d time.Duration
			if sub.Abs > 0 {
				d = sub.Abs
			} else {
				d = sub.End.Sub(sub.Start)
			}
			row.Column(d.String()).SetFormat(tabulate.FmtItalic)

			row.Column(
				fmt.Sprintf("%.2f%%", float64(d)/float64(duration)*100)).
				SetFormat(tabulate.FmtItalic)
		}
	}
	row := tab.Row()
	row.Column("Total").SetFormat(tabulate.FmtBold)
	row.Column(t.Samples[len(t.Samples)-1].End.Sub(t.Start).String()).
		SetFormat(tabulate.FmtBold)

	row.Column("").SetFormat(tabulate.FmtBold)
	row.Column(FileSize(sent + rcvd).String()).SetFormat(tabulate.FmtBold)

	for _, stat := range iostats {
		s := stat.Sent.Load()
		r := stat.Recvd.Load()

		row.Column(FileSize(s + r).String()).SetFormat(tabulate.FmtBold)
	}

	row = tab.Row()
	row.Column("\u251C\u2574Sent").SetFormat(tabulate.FmtItalic)
	row.Column("")
	row.Column(
		fmt.Sprintf("%.2f%%", float64(sent)/float64(sent+rcvd)*100)).
		SetFormat(tabulate.FmtItalic)
	row.Column(FileSize(sent).String()).SetFormat(tabulate.FmtItalic)

	for _, stat := range iostats {
		s := stat.Sent.Load()
		row.Column(FileSize(s).String()).SetFormat(tabulate.FmtItalic)
	}

	row = tab.Row()
	row.Column("\u251C\u2574Rcvd").SetFormat(tabulate.FmtItalic)
	row.Column("")
	row.Column(
		fmt.Sprintf("%.2f%%", float64(rcvd)/float64(sent+rcvd)*100)).
		SetFormat(tabulate.FmtItalic)
	row.Column(FileSize(rcvd).String()).SetFormat(tabulate.FmtItalic)

	for _, stat := range iostats {
		r := stat.Sent.Load()
		row.Column(FileSize(r).String()).SetFormat(tabulate.FmtItalic)
	}

	row = tab.Row()
	row.Column("\u2570\u2574Flcd").SetFormat(tabulate.FmtItalic)
	row.Column("")
	row.Column("")
	row.Column(fmt.Sprintf("%v", flushed)).SetFormat(tabulate.FmtItalic)

	for _, stat := range iostats {
		f := stat.Sent.Load()
		row.Column(fmt.Sprintf("%v", f)).SetFormat(tabulate.FmtItalic)
	}

	tab.Print(os.Stdout)
}

// Get gets the named sample.
func (t *Timing) Get(label string) *Sample {
	for _, sample := range t.Samples {
		if sample.Label == label {
			return sample
		}
	}
	return nil
}

// Sample contains information about one timing sample.
type Sample struct {
	Label   string
	Start   time.Time
	End     time.Time
	Abs     time.Duration
	Cols    []string
	Samples []*Sample
}

// Duration returns the sample duration.
func (s Sample) Duration() time.Duration {
	if s.Abs > 0 {
		return s.Abs
	}
	return s.End.Sub(s.Start)
}

// SubSample adds a sub-sample for a timing sample.
func (s *Sample) SubSample(label string, end time.Time) {
	start := s.Start
	if len(s.Samples) > 0 {
		start = s.Samples[len(s.Samples)-1].End
	}
	s.Samples = append(s.Samples, &Sample{
		Label: label,
		Start: start,
		End:   end,
	})
}

// AbsSubSample adds an absolute sub-sample for a timing sample.
func (s *Sample) AbsSubSample(label string, duration time.Duration) {
	s.Samples = append(s.Samples, &Sample{
		Label: label,
		Abs:   duration,
	})
}

// Get gets the named sub-sample.
func (s *Sample) Get(label string) *Sample {
	for _, sample := range s.Samples {
		if sample.Label == label {
			return sample
		}
	}
	return nil
}
