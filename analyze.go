package main

import (
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
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
	abstain = "Abstain"
	invalid = "Invalid"
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
		// me reaping:
		case "COLLINS":
			return fmt.Sprintf("Collins, %s.", name[:1])
		case "JACOBS":
			return "Underwood Jacobs"
		case "ROUX":
			return "Le Roux"
		case "KRIEG":
			return "Von Krieg"
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

func AnalyzeManyContests(b *BallotData, coalesceInvalid bool, contestIDs ...int) map[string]int {
	var err error
	candss := make([]map[int]string, len(contestIDs))
	for i, contestID := range contestIDs {
		candss[i], err = candidates(b, contestID)
		if err != nil {
			panic(err)
		}
	}

	// Pad names consistently
	ws := make([]int, len(candss))
	for i, cands := range candss {
		if !coalesceInvalid {
			ws[i] = len(abstain)
		}
		for _, name := range cands {
			if ws[i] < len(name) {
				ws[i] = len(name)
			}
		}
		for id, name := range cands {
			cands[id] = fmt.Sprintf("%-"+strconv.Itoa(ws[i])+"v", name)
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
				if coalesceInvalid {
					incomplete++
					nVotes = 0
					break
				} else {
					vote = fmt.Sprintf("%-"+strconv.Itoa(ws[i])+"v", vote)
				}
			}

			votes[i] = vote
			nVotes++
		}
		if nVotes == 0 {
			continue
		}

		voteString := strings.Join(votes, "|")
		results[voteString]++
	}
	if coalesceInvalid {
		results["Incomplete"] = incomplete
	}

	return results
}

func GridChart(b *BallotData, coalesceInvalid bool, contestIDs ...int) [][]any {
	ns := make([]int, len(contestIDs))
	candss := make([][]string, len(contestIDs))
	for i, contestID := range contestIDs {
		cands, err := candidates(b, contestID)
		if err != nil {
			panic(err)
		}
		candss[i] = maps.Values(cands)
		if !coalesceInvalid {
			candss[i] = append(candss[i], abstain, invalid)
		}
		slices.SortFunc(candss[i], less)
		ns[i] = len(candss[i])
	}

	// hack hack hack pad names to match AnalyzeManyContests
	for _, cands := range candss {
		w := 0
		for _, name := range cands {
			if w < len(name) {
				w = len(name)
			}
		}
		for i, name := range cands {
			cands[i] = fmt.Sprintf("%-"+strconv.Itoa(w)+"v", name)
		}
	}

	h, w := sum(ns[1:]), sum(ns[:len(ns)-1])

	showContestLabels := len(contestIDs) > 2
	labels := 1
	if showContestLabels {
		labels = 2
	}
	ret := make([][]any, h+labels)
	if showContestLabels {
		c := labels
		ret[0] = make([]any, w+labels)
		for j := 0; j < len(ns)-1; j++ {
			for m := 0; m < ns[j]; m++ {
				ret[0][c+m] = b.Contests[contestIDs[j]].Description
			}
			c += ns[j]
		}
	}
	ret[labels-1] = make([]any, w+labels)
	c := labels
	for j := 0; j < len(ns)-1; j++ {
		for m := 0; m < ns[j]; m++ {
			ret[labels-1][c+m] = strings.TrimSpace(candss[j][m])
		}
		c += ns[j]
	}

	r := labels
	for i := 1; i < len(contestIDs); i++ {
		for k := 0; k < ns[i]; k++ {
			ret[r+k] = make([]any, w+labels)
			if showContestLabels {
				ret[r+k][0] = b.Contests[contestIDs[i]].Description
			}
			ret[r+k][labels-1] = strings.TrimSpace(candss[i][k])
		}
		c := labels
		for j := 0; j < i; j++ {
			results := AnalyzeManyContests(b, coalesceInvalid, contestIDs[i], contestIDs[j])
			total := sum(maps.Values(results))

			for k := 0; k < ns[i]; k++ {
				cand1 := candss[i][k]
				for m := 0; m < ns[j]; m++ {
					cand2 := candss[j][m]
					votes := results[cand1+"|"+cand2]
					ret[r+k][c+m] = float64(votes) / float64(total)
				}
			}

			c += ns[j]
		}
		r += ns[i]
	}
	return ret
}

func ShowPrecinctPortions(b *BallotData) {
	counts := make(map[int]int)
	for _, cvr := range b.Raw.CVRs {
		for _, sess := range cvr.Sessions {
			counts[sess.Original.PrecinctPortionID]++
		}
	}

	min := 0x7fffffff
	for _, ct := range counts {
		if ct < min {
			min = ct
		}
	}

	for id, ct := range counts {
		fmt.Printf("%v - %v votes", b.PrecinctPortionNames[id], ct)
		if ct <= min {
			fmt.Printf(" **********")
		} else if ct <= 100 {
			fmt.Printf(" ***")
		}
		fmt.Println()
	}
}
