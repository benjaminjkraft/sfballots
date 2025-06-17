package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func ShowContestsByCard(b *BallotData) {
	contestIDs := make([]int, len(b.Raw.Contests))
	contestIndexes := make(map[int]int, len(b.Raw.Contests))
	for i, contest := range b.Raw.Contests {
		contestIDs[i] = contest.ID
		contestIndexes[contest.ID] = i
	}

	counts := map[string]int{}
	sig := make([]byte, len(contestIndexes))
	for _, card := range b.Cards {
		for i := range sig {
			sig[i] = ' '
		}
		for _, contest := range card.Contests {
			sig[contestIndexes[contest.ID]] = 'X'
		}
		counts[string(sig)] += 1
	}

	digits := len(strconv.Itoa(max(contestIDs...)))
	for i := 0; i < digits; i++ {
		for _, id := range contestIDs {
			fmt.Printf(fmt.Sprintf("%0"+strconv.Itoa(digits)+"d", id)[i : i+1])
		}
		fmt.Println()
	}

	for k, v := range counts {
		fmt.Println(k, v)
	}
	fmt.Println()
}

const (
	abstain   = "Abstain"
	invalid   = "Invalid"
	exhausted = "Exhausted"
)

func shortName(name string) string {
	// TODO: fallacies programmers believe about names

	// Alaska does Last, First M. "Nick" Jr.; just take up to the comma.
	i := strings.IndexByte(name, ',')
	if i != -1 {
		return name[:i]
	}

	// SF does FIRST M. "NICK" LAST JR.; try to guess what is the "last" name.
	for {
		i := strings.LastIndexByte(name, ' ')
		if i == -1 {
			return name
		}
		last := name[i+1:]
		switch last {
		case "", "II", "III", "JR", "JR.", "SR", "SR.":
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

func scoreRCVContest(contest *RawCardContest, candidates map[int]string, ranks int) ([]int, string, error) {
	switch {
	// Undervote here seems to mean "no first choice", which may still be a
	// valid vote, so ignore it (and handle below).
	case contest.Overvotes > 0:
		return nil, invalid, nil
		// we ignore outstack condition IDs because UnusedRanking is fine.
	}

	votedRanks := make([]int, ranks)
	for i := range votedRanks {
		votedRanks[i] = -1
	}
	for _, mark := range contest.Marks {
		if mark.IsAmbiguous {
			continue
		}
		if votedRanks[mark.Rank-1] == -1 {
			votedRanks[mark.Rank-1] = mark.CandidateID
		} else {
			// voted for two candidates at the same rank, we will discard this
			// and lower ranks below
			votedRanks[mark.Rank-1] = -2
		}
	}

	// We assume that ranks are tabulated as follows: for each rank, take the
	// next rank the voter voted, skipping any that are empty, or where the
	// voter already voted for this candidate. If two candidates have the same
	// rank, bail, but don't discard higher ranks.
	// So ABCD → ABCD, ABAB → AB, A__B → AB, AB(CD)E → AB, etc.
	tabulatedRanks := make([]int, 0, ranks)
	seen := make(map[int]bool, ranks)
	for _, cand := range votedRanks {
		if cand == -2 {
			// two at the same rank, discard this and later ranks.
			if len(tabulatedRanks) == 0 {
				// if this was the only rank, it's an overvote.
				return nil, invalid, nil
			}
			break
		}
		if cand < 0 || seen[cand] {
			// no valid vote at this rank, but we continue on
			continue
		}
		tabulatedRanks = append(tabulatedRanks, cand)
		seen[cand] = true
	}

	if len(tabulatedRanks) == 0 {
		return nil, abstain, nil
	}

	names := make([]string, len(tabulatedRanks))
	for i, id := range tabulatedRanks {
		name, ok := candidates[id]
		if !ok {
			return nil, invalid, fmt.Errorf("unexpected candidate: %v", id)
		}
		names[i] = name
	}
	return tabulatedRanks, strings.Join(names, " > "), nil
}

func rcvBallotSummary(ranks [][]int, candidates map[int]string) string {
	counts := map[int]int{}  // number of candidates -> count
	singles := map[int]int{} // candidate ID -> count
	firsts := map[int]int{}  // candidate ID -> count

	for _, ranking := range ranks {
		counts[len(ranking)]++
		if len(ranking) == 0 {
			continue
		}
		firsts[ranking[0]]++
		if len(ranking) == 1 {
			singles[ranking[0]]++
		}
	}

	var buf strings.Builder
	for i := 0; i <= len(candidates); i++ {
		if counts[i] > 0 {
			fmt.Fprintf(&buf, "%v candidates: %v\n", i, counts[i])
		}
	}
	for id, name := range candidates {
		fmt.Fprintf(&buf, "%v: %v only, %v first\n", name, singles[id], firsts[id])
	}
	return buf.String()
}

type irvRoundResults struct {
	topChoices map[string]int
	eliminated string
}

func runIRV(ranks [][]int, candidates map[int]string) []irvRoundResults {
	var results []irvRoundResults
	eliminated := map[string]bool{}

	for {
		topChoices := map[string]int{}
		for _, ranking := range ranks {
			top := exhausted
			for _, rank := range ranking {
				if !eliminated[candidates[rank]] {
					top = candidates[rank]
					break
				}
			}
			topChoices[top]++
		}

		total := 0
		worst := -1
		worstName := ""
		for name, votes := range topChoices {
			total += votes
			if name != exhausted && (worst == -1 || votes < worst) {
				worst = votes
				worstName = name
			}
		}
		eliminated[worstName] = true
		results = append(results, irvRoundResults{topChoices, worstName})

		if len(topChoices) <= 2 { // winner + exhausted
			return results
		}
	}
}

func borda(numRanks int) func(int) int {
	return func(rank int) int { return numRanks - rank }
}

var dowdall = func(rank int) float64 { return 1 / (float64(rank) + 1) }

func runPositional[T numeric](ranks [][]int, candidates map[int]string, value func(int) T) map[string]T {
	totals := map[string]T{}
	for _, ranking := range ranks {
		for i, rank := range ranking {
			totals[candidates[rank]] += value(i)
		}
	}
	return totals
}

func convertCondorcetMap(m map[[2]int]int, candidates map[int]string) map[string]int {
	ret := make(map[string]int, len(m))
	for k, v := range m {
		ret[fmt.Sprintf("%v > %v", candidates[k[0]], candidates[k[1]])] = v
	}
	return ret
}

// returns paths iff winner is not condorcet
func runSchulze(ranks [][]int, candidates map[int]string) (winner string, prefs, paths map[string]int) {
	candsMap := map[int]bool{}
	for _, ranking := range ranks {
		for _, rank := range ranking {
			candsMap[rank] = true
		}
	}
	prefsMap := map[[2]int]int{}
	for _, ranking := range ranks {
		seen := map[int]bool{}
		for i, rank := range ranking {
			seen[rank] = true
			for _, otherRank := range ranking[i+1:] {
				prefsMap[[2]int{rank, otherRank}] += 1
			}
		}
		for cand := range candsMap {
			if !seen[cand] {
				// unranked candidates go after all ranked candidates
				for _, rank := range ranking {
					prefsMap[[2]int{rank, cand}] += 1
				}
			}
		}
	}
	cands := maps.Keys(candsMap)
	sort.Ints(cands)

	for _, c1 := range cands {
		winner := true
		for _, c2 := range cands {
			if c1 != c2 && prefsMap[[2]int{c1, c2}] <= prefsMap[[2]int{c2, c1}] {
				winner = false
			}
		}
		if winner {
			return candidates[c1], convertCondorcetMap(prefsMap, candidates), nil
		}
	}

	pathsMap := map[[2]int]int{}
	// https://en.wikipedia.org/wiki/Schulze_method#Implementation
	// d is prefsMap, p is pathsMap, 1..C is cands.
	for _, c1 := range cands {
		for _, c2 := range cands {
			if c1 == c2 {
				continue
			}

			if prefsMap[[2]int{c1, c2}] > prefsMap[[2]int{c2, c1}] {
				pathsMap[[2]int{c1, c2}] = prefsMap[[2]int{c1, c2}]
			} else {
				pathsMap[[2]int{c1, c2}] = 0
			}
		}
	}

	for _, c1 := range cands {
		for _, c2 := range cands {
			if c1 == c2 {
				continue
			}
			for _, c3 := range cands {
				if c1 == c3 || c2 == c3 {
					continue
				}
				pathsMap[[2]int{c2, c3}] = max(
					pathsMap[[2]int{c2, c3}],
					min(pathsMap[[2]int{c2, c1}], pathsMap[[2]int{c1, c3}]))
			}
		}
	}

	for _, c1 := range cands {
		winner := true
		for _, c2 := range cands {
			if c1 != c2 && pathsMap[[2]int{c1, c2}] <= pathsMap[[2]int{c2, c1}] {
				winner = false
			}
		}
		if winner {
			return candidates[c1],
				convertCondorcetMap(prefsMap, candidates),
				convertCondorcetMap(pathsMap, candidates)
		}
	}
	panic(fmt.Sprintf("no schulze winner %v %v", prefsMap, pathsMap))
}

func ShowRCVContest(b *BallotData, contestID int) {
	// NOTE: results here differ slightly from published results; seemingly for
	// ballots that get manually audited that doesn't make it back into the
	// dataset.
	contestInfo := b.Contests[contestID]

	cands, err := candidates(b, contestID)
	if err != nil {
		panic(err)
	}

	stringResults := map[string]int{}
	var rankResults [][]int
	for _, card := range b.Cards {
		for _, contest := range card.Contests {
			if contest.ID != contestID {
				continue
			}

			ranks, voteStr, err := scoreRCVContest(contest, cands, contestInfo.NumOfRanks)
			if err != nil {
				panic(err)
			}
			stringResults[voteStr]++
			rankResults = append(rankResults, ranks)
		}
	}

	fmt.Printf("%v (RCV, rank up to %v)\n", contestInfo.Description, contestInfo.NumOfRanks)
	fmt.Print(formatResults(stringResults))
	fmt.Println()

	fmt.Println("Ballot summary")
	fmt.Print(rcvBallotSummary(rankResults, cands))
	fmt.Println()

	irvResults := runIRV(rankResults, cands)
	for i, round := range irvResults {
		fmt.Printf("IRV Round %v\n", i+1)
		fmt.Print(formatResults(round.topChoices))
		if i == len(irvResults)-1 {
			fmt.Println(round.eliminated, "wins")
		} else {
			fmt.Println(round.eliminated, "is eliminated")
		}
		fmt.Println()
	}

	fmt.Println("Borda count")
	fmt.Print(formatResults(runPositional(rankResults, cands, borda(contestInfo.NumOfRanks))))
	fmt.Println()

	fmt.Println("Nauru/Dowdall method")
	fmt.Print(formatResults(runPositional(rankResults, cands, dowdall)))
	fmt.Println()

	fmt.Println("Schulze method")
	winner, prefs, paths := runSchulze(rankResults, cands)
	fmt.Println(winner, ternary(paths == nil, "(condorcet winner)", ""))
	fmt.Println("Preferences:")
	fmt.Print(formatResults(prefs))
	if paths != nil {
		fmt.Println("Strongest paths:")
		fmt.Print(formatResults(paths))
	}
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

	ret := make([][]any, h+2)
	c := 2
	ret[0] = make([]any, w+2)
	for j := 0; j < len(ns)-1; j++ {
		for m := 0; m < ns[j]; m++ {
			ret[0][c+m] = b.Contests[contestIDs[j]].Description
		}
		c += ns[j]
	}
	ret[1] = make([]any, w+2)
	c = 2
	for j := 0; j < len(ns)-1; j++ {
		for m := 0; m < ns[j]; m++ {
			ret[1][c+m] = strings.TrimSpace(candss[j][m])
		}
		c += ns[j]
	}

	r := 2
	for i := 1; i < len(contestIDs); i++ {
		for k := 0; k < ns[i]; k++ {
			ret[r+k] = make([]any, w+2)
			ret[r+k][0] = b.Contests[contestIDs[i]].Description
			ret[r+k][1] = strings.TrimSpace(candss[i][k])
		}
		c := 2
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
