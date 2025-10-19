package main

import (
	"log"
	"sync/atomic"
)

type attrStatus string

const (
	// the stored xattr is missing
	attrMiss attrStatus = "missing"
	// the stored xattr is different than the calculated value
	attrDiff attrStatus = "different"
	// the stored xattr is equal to the calculated value
	attrSame attrStatus = "same"
)

// decisionWithCount is used to hold both the name of
// the decision and the count of how often it was hit.
type decisionWithCount struct {
	atomic.Uint64
	name string
}

func (d *decisionWithCount) String() string {
	return d.name
}

type decisions struct {
	ok         decisionWithCount
	corrupt    decisionWithCount
	timechange decisionWithCount
	outdated   decisionWithCount
	new        decisionWithCount
}

var stats = struct {
	total              atomic.Uint64
	errorsNotRegular   atomic.Uint64
	errorsOpening      atomic.Uint64
	errorsWritingXattr atomic.Uint64
	errorsOther        atomic.Uint64
	inprogress         atomic.Uint64
	removed            atomic.Uint64
	decisions          decisions
}{
	decisions: decisions{
		ok:         decisionWithCount{name: "ok"},
		corrupt:    decisionWithCount{name: "corrupt"},
		timechange: decisionWithCount{name: "timechange"},
		outdated:   decisionWithCount{name: "outdated"},
		new:        decisionWithCount{name: "new"},
	},
}

var decisionTable = []struct {
	tsStatus     attrStatus
	sha256Status attrStatus
	decision     *decisionWithCount
}{
	// ts      sha256    decision
	{attrSame, attrSame, &stats.decisions.ok},
	{attrSame, attrDiff, &stats.decisions.corrupt},
	{attrSame, attrMiss, &stats.decisions.new}, // or outdated?
	{attrDiff, attrSame, &stats.decisions.timechange},
	{attrDiff, attrDiff, &stats.decisions.outdated},
	{attrDiff, attrMiss, &stats.decisions.outdated},
	{attrMiss, attrSame, &stats.decisions.timechange},
	{attrMiss, attrDiff, &stats.decisions.outdated},
	{attrMiss, attrMiss, &stats.decisions.new},
}

func lookupDecision(tsStatus, sha256Status attrStatus) *decisionWithCount {
	for _, v := range decisionTable {
		if v.tsStatus == tsStatus && v.sha256Status == sha256Status {
			return v.decision
		}
	}
	log.Panicf("No decision for tsStatus=%v sha256Status=%v", tsStatus, sha256Status)
	return nil
}
