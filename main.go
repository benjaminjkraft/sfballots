package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func doMany(b *BallotData, dir string, show bool, ids ...int) {
	results := AnalyzeManyContests(b, ids...)
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

	for _, id := range ids {
		ShowContest(b, id)
	}

	for _, is := range powerset(ids) {
		if len(is) > 1 {
			doMany(b, os.Args[1], len(is) == len(ids), is...)
		}
	}

	if len(ids) > 2 {
		grid := GridChart(b, ids...)
		filename := filepath.Join(os.Args[1],
			"results_grid_"+strings.Join(map1(strconv.Itoa, ids), "_")+".csv")
		err = os.WriteFile(filename, []byte(formatGrid(grid)), 0o644)
		if err != nil {
			panic(err)
		}
		fmt.Println("wrote", filename)
	}
}
