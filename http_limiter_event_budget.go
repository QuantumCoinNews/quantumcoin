package main

import "time"

func httpLimiterEventLabelV1(label string) string {
	if len(label) > httpLimiterMaxKeyBytes {
		return label[:httpLimiterMaxKeyBytes]
	}
	return label
}

func (g *HTTPGuard) beginEventV1(now time.Time) bool {
	if g == nil {
		return false
	}
	g.eventMu.Lock()
	defer g.eventMu.Unlock()
	if g.eventInFlight || (!g.nextEventAt.IsZero() && now.Before(g.nextEventAt)) {
		return false
	}
	g.eventInFlight = true
	g.nextEventAt = now.Add(httpSecurityEventCooldown)
	return true
}

func (g *HTTPGuard) finishEventV1() {
	g.eventMu.Lock()
	g.eventInFlight = false
	g.eventMu.Unlock()
}

func (el *EndpointLimiter) beginEventV1(now time.Time) bool {
	if el == nil {
		return false
	}
	el.eventMu.Lock()
	defer el.eventMu.Unlock()
	if el.eventInFlight || (!el.nextEventAt.IsZero() && now.Before(el.nextEventAt)) {
		return false
	}
	el.eventInFlight = true
	el.nextEventAt = now.Add(httpSecurityEventCooldown)
	return true
}

func (el *EndpointLimiter) finishEventV1() {
	el.eventMu.Lock()
	el.eventInFlight = false
	el.eventMu.Unlock()
}
