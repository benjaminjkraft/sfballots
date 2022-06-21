package main

import (
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
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

const (
	abstain = "Abs"
	invalid = "Inv"
)

func shortName(name string) string {
	// TODO: fallacies programmers believe about names
	for {
		i := strings.LastIndexByte(name, ' ')
		if i == -1 {
			return name
		}
		last := name[i+1:]
		switch last {
		case "II", "III", "JR", "JR.", "SR", "SR.":
			name = name[:i]
		default:
			return cases.Title(language.English).String(strings.TrimSuffix(last, ","))
		}
	}
}

func candidates(b *BallotData, contestID int) (map[int]string, error) {
	// NOTE: results here differ slightly from published results; seemingly for
	// ballots that get manually audited that doesn't make it back into the
	// dataset.
	cands := b.CandidatesByContest[contestID]
	ret := map[int]string{}
	for _, cand := range cands {
		ret[cand.ID] = shortName(cand.Description)
	}
	if len(ret) != len(cands) {
		return nil, fmt.Errorf("dupe candidates, got %v", cands)
	}
	return ret, nil
}

func scoreContest(contest *RawCardContest, candidates map[int]string) (string, error) {
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

func ShowContest(b *BallotData, contestID int) {
	// NOTE: results here differ slightly from published results; seemingly for
	// ballots that get manually audited that doesn't make it back into the
	// dataset.
	cands, err := candidates(b, contestID)
	if err != nil {
		panic(err)
	}

	results := map[string]int{}
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
	fmt.Print(formatResults(results))
	fmt.Println()
}

func AnalyzeManyContests(b *BallotData, contestIDs ...int) map[string]int {
	var err error
	candss := make([]map[int]string, len(contestIDs))
	for i, contestID := range contestIDs {
		candss[i], err = candidates(b, contestID)
		if err != nil {
			panic(err)
		}
	}

	// Pad names consistently
	for _, cands := range candss {
		w := 0
		for _, name := range cands {
			if w < len(name) {
				w = len(name)
			}
		}
		for id, name := range cands {
			cands[id] = fmt.Sprintf("%-"+strconv.Itoa(w)+"v", name)
		}
	}

	contestToIndex := make(map[int]int, len(contestIDs))
	for i, contestID := range contestIDs {
		contestToIndex[contestID] = i
	}

	results := map[string]int{}
	incomplete := 0

	for _, card := range b.Cards {
		votes := make([]string, len(contestIDs))
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
				nVotes = 0
				break
			}

			votes[i] = vote
			nVotes++
		}
		if nVotes == 0 {
			continue
		}

		voteString := strings.Join(votes, " ")
		results[voteString]++
	}
	results["Incomplete"] = incomplete

	return results
}
