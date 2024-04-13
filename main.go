package main

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/exp/maps"
)

func doMany(b *BallotData, prefix string, show bool, ids ...int) {
	results := AnalyzeManyContests(b, len(ids) > 2, ids...)
	if show {
		fmt.Print(formatResults(results))
	}
	filename := prefix + "results_" + strings.Join(map1(strconv.Itoa, ids), "_") + ".csv"
	err := os.WriteFile(filename, []byte(formatCSV(results)), 0o644)
	if err != nil {
		panic(err)
	}
	fmt.Println("wrote", filename)
}

func main() {
	if len(os.Args) == 1 {
		fmt.Printf("usage: %s data/CVR_Export_YYYYMMDDHHMMSS.zip> [<contest IDs>]\n", os.Args[0])
		os.Exit(1)
	}

	d, err := LoadAll(os.Args[1])
	if err != nil {
		panic(err)
	}

	prefix, _, _ := strings.Cut(os.Args[1], ".")
	prefix += "_"

	b, err := BuildBallotData(d)
	if err != nil {
		panic(err)
	}

	ids := make([]int, len(os.Args)-2)
	for i, arg := range os.Args[2:] {
		ids[i], err = strconv.Atoi(arg)
		if err != nil {
			panic(err)
		}
	}

	if len(ids) == 0 {
		fmt.Println(b)

		contestIDs := maps.Keys(b.Contests)
		sort.Ints(contestIDs)
		for _, id := range contestIDs {
			fmt.Println(id, b.Contests[id].Description)
		}

		ShowContestsByCard(b)

		return
	}

	for _, id := range ids {
		if b.Contests[id].NumOfRanks > 0 {
			ShowRCVContest(b, id)
		} else if b.Contests[id].VoteFor > 1 {
			panic("TODO implement vote for N")
		} else {
			ShowContest(b, id)
		}
	}

	for _, is := range powerset(ids) {
		if len(is) > 1 {
			doMany(b, prefix, len(is) == len(ids), is...)
		}
	}

	if len(ids) > 1 {
		grid := GridChart(b, len(ids) > 2, ids...)
		basename := "results_grid_" + strings.Join(map1(strconv.Itoa, ids), "_")

		csv := prefix + basename + ".csv"
		err = os.WriteFile(csv, []byte(formatGrid(grid)), 0o644)
		if err != nil {
			panic(err)
		}
		fmt.Println("wrote", csv)

		html := prefix + basename + ".html"
		err = os.WriteFile(html, []byte(formatGridHTML(grid)), 0o644)
		if err != nil {
			panic(err)
		}
		fmt.Println("wrote", html)
	}
}
