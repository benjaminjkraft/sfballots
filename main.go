package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/exp/maps"
)

func doMany(b *BallotData, dir string, show bool, ids ...int) {
	results := AnalyzeManyContests(b, len(ids) > 2, ids...)
	if show {
		fmt.Print(formatResults(results))
	}
	filename := filepath.Join(dir,
		"results_"+strings.Join(map1(strconv.Itoa, ids), "_")+".csv")
	err := os.WriteFile(filename, []byte(formatCSV(results)), 0o644)
	if err != nil {
		panic(err)
	}
	fmt.Println("wrote", filename)
}

func main() {
	d, err := LoadAll(os.Args[1])
	if err != nil {
		panic(err)
	}

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

	fmt.Println(b)

	ShowContestsByCard(b)

	if len(ids) == 0 {
		contestIDs := maps.Keys(b.Contests)
		sort.Ints(contestIDs)
		for _, id := range contestIDs {
			fmt.Println(id, b.Contests[id].Description)
		}
	}

	for _, id := range ids {
		if b.Contests[id].NumOfRanks > 0 {
			ShowRCVContest(b, id)
		} else {
			ShowContest(b, id)
		}
	}

	for _, is := range powerset(ids) {
		if len(is) > 1 {
			doMany(b, os.Args[1], len(is) == len(ids), is...)
		}
	}

	if len(ids) > 1 {
		grid := GridChart(b, len(ids) > 2, ids...)
		basename := "results_grid_" + strings.Join(map1(strconv.Itoa, ids), "_")

		csv := filepath.Join(os.Args[1], basename+".csv")
		err = os.WriteFile(csv, []byte(formatGrid(grid)), 0o644)
		if err != nil {
			panic(err)
		}
		fmt.Println("wrote", csv)

		html := filepath.Join(os.Args[1], basename+".html")
		err = os.WriteFile(html, []byte(formatGridHTML(grid)), 0o644)
		if err != nil {
			panic(err)
		}
		fmt.Println("wrote", html)
	}
}
