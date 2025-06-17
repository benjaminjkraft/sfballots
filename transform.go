package main

import "fmt"

type BallotData struct {
	Raw                  *RawBallotData
	Candidates           map[int]*RawCandidate
	Contests             map[int]*RawContest
	CandidatesByContest  map[int][]*RawCandidate
	Cards                []*RawCard
	PrecinctPortionNames map[int]string
}

func BuildBallotData(in *RawBallotData) (*BallotData, error) {
	out := BallotData{
		Raw:                  in,
		Candidates:           map[int]*RawCandidate{},
		Contests:             map[int]*RawContest{},
		CandidatesByContest:  map[int][]*RawCandidate{},
		PrecinctPortionNames: map[int]string{},
	}
	for _, cand := range in.Candidates {
		out.Candidates[cand.ID] = cand
		out.CandidatesByContest[cand.ContestID] = append(
			out.CandidatesByContest[cand.ContestID], cand)
	}
	for _, cont := range in.Contests {
		out.Contests[cont.ID] = cont
	}
	for _, cvr := range in.CVRs {
		for _, session := range cvr.Sessions {
			if session.Modified.IsCurrent {
				out.Cards = append(out.Cards, session.Modified.Cards...)
			} else {
				out.Cards = append(out.Cards, session.Original.Cards...)
			}
		}
	}
	for _, pp := range in.PrecinctPortions {
		out.PrecinctPortionNames[pp.ID] = pp.Description
	}
	return &out, nil
}

func (b *BallotData) String() string {
	return fmt.Sprintf("<ballot data, %v candidates in %v contests, %v cards>",
		len(b.Candidates), len(b.Contests), len(b.Cards))
}
