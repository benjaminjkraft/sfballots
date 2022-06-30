package main

import "fmt"

type BallotData struct {
	Raw                 *RawBallotData
	Candidates          map[int]*RawCandidate
	Contests            map[int]*RawContest
	CandidatesByContest map[int][]*RawCandidate
	Cards               []*RawCard
}

func BuildBallotData(in *RawBallotData) (*BallotData, error) {
	out := BallotData{
		Raw:                 in,
		Candidates:          map[int]*RawCandidate{},
		Contests:            map[int]*RawContest{},
		CandidatesByContest: map[int][]*RawCandidate{},
	}
	for _, cand := range in.Candidates {
		if cand.Type == "QualifiedWriteIn" {
			// Skip write-ins as we don't have that data anyway
			continue
		}
		out.Candidates[cand.ID] = cand
		out.CandidatesByContest[cand.ContestID] = append(
			out.CandidatesByContest[cand.ContestID], cand)
	}
	for _, cont := range in.Contests {
		out.Contests[cont.ID] = cont
	}
	for _, cvr := range in.CVRs {
		for _, session := range cvr.Sessions {
			out.Cards = append(out.Cards, session.Original.Cards...)
		}
	}
	return &out, nil
}

func (b *BallotData) String() string {
	return fmt.Sprintf("<ballot data, %v candidates in %v contests, %v cards>",
		len(b.Candidates), len(b.Contests), len(b.Cards))
}
