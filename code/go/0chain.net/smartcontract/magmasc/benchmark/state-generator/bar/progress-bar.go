package bar

import (
	"fmt"
	"time"

	"github.com/cheggaaa/pb/v3"
)

type (
	ProgressBar struct {
		*pb.ProgressBar
		startTime time.Time

		separate  bool
		milestone int64
	}
)

// StartNew configures and starts ProgressBar.
//
// separate bool: prints each 1% new progress bar line and elapsed time for period if true
func StartNew(num int, separate bool) *ProgressBar {
	progressBar := pb.New(num)
	progressBar.SetRefreshRate(time.Microsecond)
	progressBar.SetWidth(100)
	progressBar.Start()
	return &ProgressBar{
		ProgressBar: progressBar,
		startTime:   time.Now(),
		separate:    separate,
	}
}

func (p *ProgressBar) Increment() {
	percent := p.Current() * 100 / p.Total()
	if p.separate && percent > p.milestone {
		fmt.Printf("; %f s\n", time.Now().Sub(p.startTime).Seconds())
		p.milestone = percent
	}
	p.ProgressBar.Increment()
}

func (p *ProgressBar) Finish() {
	p.ProgressBar.Finish()
	fmt.Printf("Elapsed time: %f s\n", time.Now().Sub(p.startTime).Seconds())
}
