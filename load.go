package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"

	"golang.org/x/sync/errgroup"
)

type RawBallotData struct {
	BallotTypesAndContests       []*RawBallotTypeAndContest       `file:"BallotTypeContestManifest"`
	BallotTypes                  []*RawBallotType                 `file:"BallotTypeManifest"`
	Candidates                   []*RawCandidate                  `file:"CandidateManifest"`
	Contests                     []*RawContest                    `file:"ContestManifest"`
	CountingGroups               []*RawCountingGroup              `file:"CountingGroupManifest"`
	Districts                    []*RawDistrict                   `file:"DistrictManifest"`
	DistrictsAndPrecinctPortions []*RawDistrictAndPrecinctPortion `file:"DistrictPrecinctPortionManifest"`
	DistrictTypes                []*RawDistrictType               `file:"DistrictTypeManifest"`
	OutstackConditions           []*RawOutstackCondition          `file:"OutstackConditionManifest"`
	Parties                      []*RawParty                      `file:"PartyManifest"`
	Precincts                    []*RawPrecinct                   `file:"PrecinctManifest"`
	PrecinctPortions             []*RawPrecinctPortion            `file:"PrecinctPortionManifest"`
	Tabulators                   []*RawTabulator                  `file:"TabulatorManifest"`
	CVRs                         []*RawCVR                        `file:"-"`
}

type RawBallotTypeAndContest struct {
	BallotTypeID int
	ContestID    int
}

type RawBallotType struct {
	Description string
	ID          int
}

type RawCandidate struct {
	Description string
	ID          int
	ContestID   int
	Type        string
	Disabled    int
}

type RawContest struct {
	Description string
	ID          int
	DistrictID  int
	VoteFor     int
	NumOfRanks  int
	Disabled    int
}

type RawCountingGroup struct {
	Description string
	ID          int
}

type RawDistrict struct {
	Description    string
	ID             int
	DistrictTypeID string
}

type RawDistrictAndPrecinctPortion struct {
	DistrictID        int
	PrecinctPortionID int
}

type RawDistrictType struct {
	Description string
	ID          string
}

type RawOutstackCondition struct {
	Description string
	ID          int
}

type RawParty struct {
	Description string
	ID          int
}

type RawPrecinct struct {
	Description string
	ID          int
	ExternalID  string
}

type RawPrecinctPortion struct {
	Description string
	ID          int
	ExternalID  string
	PrecinctID  int
}

type RawTabulator struct {
	Description          string
	ID                   int
	VotingLocationNumber int
	VotingLocationName   string
	Type                 string
}

type RawCVR struct {
	Sessions []*RawSession
}

type RawSession struct {
	BatchID         int
	CountingGroupID int
	Original        RawSessionOriginal
	RecordID        int
	SessionType     string
	TabulatorID     int
}

type RawSessionOriginal struct {
	BallotTypeID      int
	Cards             []*RawCard
	IsCurrent         bool
	PrecinctPortionID int
}

type RawCard struct {
	ID                   int
	KeyInID              int
	PaperIndex           int
	Contests             []*RawCardContest
	OutstackConditionIDs []int
}

type RawCardContest struct {
	ID                   int
	ManifestationID      int
	Undervotes           int
	Overvotes            int
	OutstackConditionIDs []int
	Marks                []*RawMark
}

type RawMark struct {
	CandidateID          int
	ManifestationID      int
	PartyID              int
	Rank                 int
	MarkDensity          int
	IsAmbiguous          bool
	IsVote               bool
	OutstackConditionIDs []int
}

type file struct {
	Version string
	List    any
}

func decode(filename string, v any) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	return json.NewDecoder(f).Decode(v)
}

func LoadAll(dir string) (*RawBallotData, error) {
	var out RawBallotData
	rv := reflect.ValueOf(&out).Elem()
	typ := rv.Type()
	for i := 0; i < typ.NumField(); i++ {
		name := typ.Field(i).Tag.Get("file")
		if name == "-" {
			continue
		}

		x := file{List: rv.Field(i).Addr().Interface()}
		err := decode(filepath.Join(dir, name+".json"), &x)
		if err != nil {
			return nil, err
		}
	}

	cvrFilenames, err := filepath.Glob(filepath.Join(dir, "CvrExport*.json"))
	if err != nil {
		return nil, err
	}

	out.CVRs = make([]*RawCVR, len(cvrFilenames))
	var g errgroup.Group
	g.SetLimit(512) // avoid ulimit problems
	for i, filename := range cvrFilenames {
		i, filename := i, filename
		g.Go(func() error {
			err := decode(filename, &out.CVRs[i])
			return err
		})
	}
	return &out, g.Wait()
}
