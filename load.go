package main

import (
	"archive/zip"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"

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
	RecordID        json.RawMessage // int for older CVR, string for newer
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

type loader interface {
	load(name string) (fs.File, error)
	files() ([]fs.FileInfo, error)
	Close() error
}

func decode(loader loader, filename string, v any) error {
	f, err := loader.load(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	return json.NewDecoder(f).Decode(v)
}

type dirLoader struct{ dir string }

func (dl *dirLoader) load(name string) (fs.File, error) {
	return os.Open(filepath.Join(dl.dir, name))
}

func (dl *dirLoader) files() ([]fs.FileInfo, error) {
	f, err := os.Open(dl.dir)
	if err != nil {
		return nil, err
	}
	return f.Readdir(0)
}

func (dl *dirLoader) Close() error { return nil }

type zipLoader struct {
	*zip.ReadCloser
}

func newZipLoader(path string) (*zipLoader, error) {
	r, err := zip.OpenReader(path)
	return &zipLoader{r}, err
}

func (zl *zipLoader) load(name string) (fs.File, error) {
	return zl.Open(name)
}

func (zl *zipLoader) files() ([]fs.FileInfo, error) {
	ret := make([]fs.FileInfo, len(zl.File))
	for i, f := range zl.File {
		ret[i] = f.FileInfo()
	}
	return ret, nil
}

func LoadAll(dirOrZip string) (*RawBallotData, error) {
	stat, err := os.Stat(dirOrZip)
	if err != nil {
		return nil, err
	}

	var loader loader
	if stat.IsDir() {
		loader = &dirLoader{dirOrZip}
	} else {
		loader, err = newZipLoader(dirOrZip)
		if err != nil {
			return nil, err
		}
	}

	var out RawBallotData
	rv := reflect.ValueOf(&out).Elem()
	typ := rv.Type()
	for i := 0; i < typ.NumField(); i++ {
		name := typ.Field(i).Tag.Get("file")
		if name == "-" {
			continue
		}

		x := file{List: rv.Field(i).Addr().Interface()}
		err := decode(loader, name+".json", &x)
		if err != nil {
			return nil, err
		}
	}

	fileInfos, err := loader.files()
	if err != nil {
		return nil, err
	}
	var cvrFilenames []string
	for _, fileInfo := range fileInfos {
		if strings.HasPrefix(fileInfo.Name(), "CvrExport") {
			cvrFilenames = append(cvrFilenames, fileInfo.Name())
		}
	}

	out.CVRs = make([]*RawCVR, len(cvrFilenames))
	var g errgroup.Group
	g.SetLimit(512) // avoid ulimit problems
	for i, filename := range cvrFilenames {
		i, filename := i, filename
		g.Go(func() error {
			err := decode(loader, filename, &out.CVRs[i])
			return err
		})
	}
	return &out, g.Wait()
}
