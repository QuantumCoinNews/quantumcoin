// Package consolefx contains QuantumCoin project code for this module.
package consolefx

import (
	"fmt"
	"io"
	"os"
	"time"
)

// SpinFor waits while writing complete progress lines to the shared console.
func SpinFor(d time.Duration) {
	spinFor(os.Stdout, d)
}

func spinFor(out io.Writer, d time.Duration) {
	const frames = "|/-\\"
	end := time.Now().Add(d)
	for i := 0; time.Now().Before(end); i++ {
		fmt.Fprintf(out, "%c Waiting for the next mining cycle\n", frames[i%len(frames)])
		time.Sleep(min(time.Until(end), time.Second))
	}
}

// SpinBrief waits for a short interval with console progress.
func SpinBrief() { SpinFor(1200 * time.Millisecond) }
