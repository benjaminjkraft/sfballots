package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

func ShowContestsByCard(b *BallotData) {
	counts := map[string]int{}
	// TODO: don't assume contest IDs are sequential 1-indexed
	sig := make([]byte, len(b.Contests))
	for _, card := range b.Cards {
		for i := range sig {
			sig[i] = ' '
		}
		for _, contest := range card.Contests {
			sig[contest.ID-1] = 'X'
		}
		counts[string(sig)] += 1
	}
	for k, v := range counts {
		fmt.Println(k, v)
	}
}

type propVote int

const (
	yes propVote = iota
	no
	abstain
	invalid
)

func (v propVote) String() string {
	switch v {
	case no:
		return "No"
	case yes:
		return "Yes"
	case abstain:
		return "Abs"
	case invalid:
		return "Inv"
	default:
		panic(fmt.Sprintf("invalid propVote %d", v))
	}
}

func propCandidates(b *BallotData, contestID int) (map[int]propVote, error) {
	// NOTE: results here differ slightly from published results; seemingly for
	// ballots that get manually audited that doesn't make it back into the
	// dataset.
	cands := b.CandidatesByContest[contestID]
	if len(cands) != 2 {
		return nil, fmt.Errorf("expected 2 candidates, got %v", cands)
	}

	ret := map[int]propVote{}
	for _, cand := range cands {
		switch cand.Description {
		case "Yes", "YES":
			ret[cand.ID] = yes
		case "No", "NO":
			ret[cand.ID] = no
		default:
			return nil, fmt.Errorf("expected candidates yes and no, got %v", cand)
		}
	}
	if len(ret) != 2 {
		return nil, fmt.Errorf("expected one yes one no, got %v", cands)
	}
	return ret, nil
}

func scoreContest(contest *RawCardContest, candidates map[int]propVote) (propVote, error) {
	switch {
	case contest.Undervotes > 0:
		return abstain, nil
	case contest.Overvotes > 0 || len(contest.OutstackConditionIDs) > 0:
		return invalid, nil
	}

	ret := invalid
	marks := 0
	for _, mark := range contest.Marks {
		if !mark.IsVote {
			continue
		}
		marks++
		var ok bool
		ret, ok = candidates[mark.CandidateID]
		if !ok {
			return invalid, fmt.Errorf("unexpected candidate: %v", mark.CandidateID)
		}
	}
	if marks != 1 {
		return invalid, fmt.Errorf("undetected under/overvote: %v", contest.Marks)
	}
	return ret, nil
}

func ShowProposition(b *BallotData, contestID int) {
	// NOTE: results here differ slightly from published results; seemingly for
	// ballots that get manually audited that doesn't make it back into the
	// dataset.
	cands, err := propCandidates(b, contestID)
	if err != nil {
		panic(err)
	}

	results := map[propVote]int{}
	for _, card := range b.Cards {
		for _, contest := range card.Contests {
			if contest.ID != contestID {
				continue
			}

			vote, err := scoreContest(contest, cands)
			if err != nil {
				panic(err)
			}
			results[vote]++
		}
	}

	fmt.Println(b.Contests[contestID].Description)
	for i := yes; i <= invalid; i++ {
		fmt.Printf("%-3v: %d\n", i, results[i])
	}
	fmt.Println()
}

func ShowManyPropositions(b *BallotData, contestIDs ...int) {
	var err error
	candss := make([]map[int]propVote, len(contestIDs))
	for i, contestID := range contestIDs {
		candss[i], err = propCandidates(b, contestID)
		if err != nil {
			panic(err)
		}
	}

	contestToIndex := make(map[int]int, len(contestIDs))
	for i, contestID := range contestIDs {
		contestToIndex[contestID] = i
	}

	results := map[string]int{}
	incomplete := 0
	total := 0

	for _, card := range b.Cards {
		votes := make([]propVote, len(contestIDs))
		for i := range votes {
			votes[i] = abstain
		}
		nVotes := 0

		for _, contest := range card.Contests {
			i, ok := contestToIndex[contest.ID]
			if !ok {
				continue
			}

			vote, err := scoreContest(contest, candss[i])
			if err != nil {
				panic(err)
			}

			if vote == abstain || vote == invalid {
				incomplete++
				total++
				nVotes = 0
				break
			}

			votes[i] = vote
			nVotes++
		}
		if nVotes == 0 {
			continue
		}

		voteStrings := make([]string, len(votes))
		for i, vote := range votes {
			voteStrings[i] = fmt.Sprintf("%-3v", vote)
		}
		voteString := strings.Join(voteStrings, " ")
		results[voteString]++
		total++
	}

	keys := make([]string, 0, len(results))
	for k := range results {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	w := len(keys[0])
	if w < len("Incomplete") {
		w = len("Incomplete")
	}
	f := "%" + strconv.Itoa(w) + "v"

	for _, k := range keys {
		fmt.Printf(f+": %7v (%4.1f%%)\n", k, results[k], float64(100*results[k])/float64(total))
	}
	fmt.Printf(f+": %7v (%4.1f%%)\n", "Incomplete", incomplete, float64(100*incomplete)/float64(total))
	fmt.Printf(f+": %7v\n", "Total", total)
}
