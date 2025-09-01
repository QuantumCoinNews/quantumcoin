package consolefx

import (
	"fmt"
	"os"
	"time"
)

// Süre boyunca terminalde ufak bir spinner gösterir.
// Log akışını bozmamak için aynı satırı kullanır ve sonunda temizler.
func SpinFor(d time.Duration) {
	frames := []rune{'|', '/', '-', '\\'}
	end := time.Now().Add(d)
	i := 0
	for time.Now().Before(end) {
		fmt.Fprintf(os.Stdout, "\r   %c", frames[i%len(frames)])
		time.Sleep(90 * time.Millisecond)
		i++
	}
	// satırı temizle
	fmt.Fprint(os.Stdout, "\r      \r")
}

// Kısa spinner (yaklaşık 1.2s)
func SpinBrief() { SpinFor(1200 * time.Millisecond) }
