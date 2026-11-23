package miner

import "time"

// MergeOptionsV1 applies the same startup option contract to every entry point.
// The last non-nil callback and last positive interval win. Non-positive
// intervals leave the previous value in place. Animate is a plain bool: the
// last packet owns it, including false in an otherwise empty packet.
// With no packets, the compatibility miner defaults remain 1200ms and animation.
func MergeOptionsV1(options ...Options) Options {
	merged := Options{Interval: 1200 * time.Millisecond, Animate: true}
	for _, option := range options {
		if option.OnBlock != nil {
			merged.OnBlock = option.OnBlock
		}
		if option.OnError != nil {
			merged.OnError = option.OnError
		}
		if option.OnTick != nil {
			merged.OnTick = option.OnTick
		}
		if option.Broadcaster != nil {
			merged.Broadcaster = option.Broadcaster
		}
		if option.Interval > 0 {
			merged.Interval = option.Interval
		}
		merged.Animate = option.Animate
	}
	return merged
}
