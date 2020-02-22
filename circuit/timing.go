//
// Copyright (c) 2020 Markku Rossi
//
// All rights reserved.
//

package circuit

import (
	"fmt"
	"time"
)

type Timing struct {
	Start   time.Time
	Samples []*Sample
}

func NewTiming() *Timing {
	return &Timing{
		Start: time.Now(),
	}
}

func (t *Timing) Sample(label string) *Sample {
	start := t.Start
	if len(t.Samples) > 0 {
		start = t.Samples[len(t.Samples)-1].End
	}
	sample := &Sample{
		Label: label,
		Start: start,
		End:   time.Now(),
	}
	t.Samples = append(t.Samples, sample)
	return sample
}

func (t *Timing) Print() {
	if len(t.Samples) == 0 {
		return
	}
	total := t.Samples[len(t.Samples)-1].End.Sub(t.Start)
	for _, sample := range t.Samples {
		duration := sample.End.Sub(sample.Start)
		dstr := fmt.Sprintf("%s", duration)
		if len(dstr) < 8 {
			dstr += "\t"
		}

		fmt.Printf("%s:\t%s\t%.2f%%\n", sample.Label, dstr,
			float64(duration)/float64(total)*100)

		for _, sub := range sample.Samples {
			d := sub.End.Sub(sub.Start)
			fmt.Printf("\x1b[3m  %s:\t%s\t%.2f%%\x1b[m\n", sub.Label, d,
				float64(d)/float64(duration)*100)
		}
	}
	fmt.Printf("\x1b[1mTotal:\t%s\x1b[m\n",
		t.Samples[len(t.Samples)-1].End.Sub(t.Start))
}

type Sample struct {
	Label   string
	Start   time.Time
	End     time.Time
	Samples []*Sample
}

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
