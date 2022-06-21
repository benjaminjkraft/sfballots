package main

import (
	"fmt"
	"os"
	"strconv"
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
		ShowManyContests(b, ids...)
	}
}
