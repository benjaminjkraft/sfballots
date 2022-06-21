package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

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

	if len(ids) > 1 {
		results := AnalyzeManyContests(b, ids...)
		fmt.Print(formatResults(results))
		err := os.WriteFile(
			filepath.Join(os.Args[1], strings.Join(os.Args[2:], "_")+".csv"),
			[]byte(formatCSV(results)),
			0o644)
		if err != nil {
			panic(err)
		}
	}
}
