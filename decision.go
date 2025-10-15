package main

import (
	"log"
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

type decision string

const (
	decisionOk         decision = "ok"
	decisionCorrupt    decision = "corrupt"
	decisionTimechange decision = "timechange"
	decisionOutdated   decision = "outdated"
	decisionNew        decision = "new"
)

var decisionTable = []struct {
	tsStatus     attrStatus
	sha256Status attrStatus
	decision     decision
}{
	// ts      sha256    decision
	{attrSame, attrSame, decisionOk},
	{attrSame, attrDiff, decisionCorrupt},
	{attrSame, attrMiss, decisionNew}, // or decisionOutdated?
	{attrDiff, attrSame, decisionTimechange},
	{attrDiff, attrDiff, decisionOutdated},
	{attrDiff, attrMiss, decisionOutdated},
	{attrMiss, attrSame, decisionTimechange},
	{attrMiss, attrDiff, decisionOutdated},
	{attrMiss, attrMiss, decisionNew},
}

func lookupDecision(tsStatus, sha256Status attrStatus) decision {
	for _, v := range decisionTable {
		if v.tsStatus == tsStatus && v.sha256Status == sha256Status {
			return v.decision
		}
	}
	log.Panicf("No decision for tsStatus=%v sha256Status=%v", tsStatus, sha256Status)
	return ""
}
