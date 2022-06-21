package main

import (
	"fmt"
	"strconv"
	"strings"

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

func cmpOne(x, y string) int {
	switch {
	case x == y:
		return 0

	case y == "Incomplete":
		return -1
	case x == "Incomplete":
		return 1
	case y == "Inv" || y == "Invalid":
		return -1
	case x == "Inv" || x == "Invalid":
		return 1
	case y == "Abs" || y == "Abstain":
		return -1
	case x == "Abs" || x == "Abstain":
		return 1
	case y == "Write-in":
		return -1
	case x == "Write-in":
		return 1

	case x == "Yes":
		return -1
	case y == "Yes":
		return 1
	case x == "No":
		return -1
	case y == "No":
		return 1

	case x < y:
		return -1
	default:
		return 1
	}
}

func nonempty[T comparable](xs []T) []T {
	var ret []T
	var zero T
	for _, x := range xs {
		if x != zero {
			ret = append(ret, x)
		}
	}
	return ret
}

func less(x, y string) bool {
	if x == y {
		return false
	}
	xw := strings.Split(x, " ")
	yw := strings.Split(y, " ")
	xw = nonempty(xw)
	yw = nonempty(yw)
	return slices.CompareFunc(xw, yw, cmpOne) == -1
}

func formatResults(results map[string]int) string {
	keys := make([]string, 0, len(results))
	total := 0
	w := len("Total")
	for k, v := range results {
		keys = append(keys, k)
		total += v
		if w < len(k) {
			w = len(k)
		}
	}
	slices.SortFunc(keys, less)

	f := "%" + strconv.Itoa(w) + "v"
	lines := make([]string, len(results)+1)
	for i, k := range keys {
		lines[i] = fmt.Sprintf(f+": %7v (%4.1f%%)", k, results[k], float64(100*results[k])/float64(total))
	}
	lines[len(results)] = fmt.Sprintf(f+": %7v", "Total", total)
	return strings.Join(lines, "\n") + "\n"
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

func ShowManyContests(b *BallotData, contestIDs ...int) {
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

	fmt.Print(formatResults(results))
}
