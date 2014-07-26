package sentinels

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"
)

var (
	sd      *SentinelsData
	sdBytes = []byte(sdJson)
)

func init() {
	rand.Seed(time.Now().Unix())
	parseSentinelsData()
}

type CardType int

const (
	Hero CardType = iota
	Villain
	Environment
)

type ExpansionType int

const (
	BaseSet ExpansionType = iota
	MiniExpansion
	RookCity
	InfernalRelics
	ShatteredTimelines
	Vengeance
	Promos
)

// Card represents a SotM card.
type Card struct {
	Name      string // unique name
	Type      CardType
	Expansion ExpansionType
	Points    int
	Advanced  int
	AdvCount  int
	Base      string // Name of original card (for some promo versions)
}

// Cards is the master map of all cards.
var Cards map[string]*Card

// CardSet is a set of cards matching the user's selection criteria.
type CardSet struct {
	Heroes       []*Card
	Villains     []*Card
	Environments []*Card
}

// SentinelsData holds all the data unmarshaled from JSON.
type SentinelsData struct {
	Difficulty DifficultyData
	Scale      []ScaleData
}

// DifficultyData contains the contents of the "difficulty" field.
type DifficultyData struct {
	Hero    []Difficulty
	Villain []Difficulty
	Env     []Difficulty
	Nump    []Difficulty
}

// Difficulty gives the difficulty for a given hero/villain/environment
type Difficulty struct {
	Name     string
	Base     string
	Points   int
	Advanced int
	AdvCount int
	Promo    bool
}

// ScaleData is the expected loss percentage for a given difficulty.
type ScaleData struct {
	Total   int
	LossPct int
}

func parseSentinelsData() {
	sd = &SentinelsData{}
	if err := json.Unmarshal(sdBytes, sd); err != nil {
		log.Fatal(err)
	}
	makeCards(sd)
}

func makeCards(sd *SentinelsData) {
	makeCard := func(d Difficulty) *Card {
		c := &Card{Name: d.Name, Base: d.Base, Points: d.Points, Advanced: d.Advanced, AdvCount: d.AdvCount}
		if c.Base == "" {
			c.Base = c.Name
		}
		return c
	}
	Cards = make(map[string]*Card)
	for _, d := range sd.Difficulty.Hero {
		c := makeCard(d)
		c.Type = Hero
		Cards[d.Name] = c
	}
	for _, d := range sd.Difficulty.Villain {
		c := makeCard(d)
		c.Type = Villain
		Cards[d.Name] = c
	}
	for _, d := range sd.Difficulty.Env {
		c := makeCard(d)
		c.Type = Environment
		Cards[d.Name] = c
	}
	for exp, names := range ExpansionCards {
		for _, name := range names {
			if c, ok := Cards[name]; ok {
				c.Expansion = exp
			} else {
				log.Printf("Couldn't find card %s while setting expansions.", name)
			}
		}
	}
}

// GetCardSet builds a CardSet containing all cards in the selected expansions.
func GetCardSet(exp []ExpansionType) *CardSet {
	cs := new(CardSet)
	for _, c := range Cards {
		found := false
		for _, e := range exp {
			if c.Expansion == e {
				found = true
				break
			}
		}
		if !found {
			continue
		}
		switch c.Type {
		case Hero:
			cs.Heroes = append(cs.Heroes, c)
		case Villain:
			cs.Villains = append(cs.Villains, c)
		case Environment:
			cs.Environments = append(cs.Environments, c)
		}
	}
	return cs
}

// String formats a CardSet for output.
func (cs *CardSet) String() string {
	var b bytes.Buffer
	fmt.Fprint(&b, "Heroes:\n")
	for _, c := range cs.Heroes {
		fmt.Fprintf(&b, "   %s\n", c.Name)
	}
	fmt.Fprint(&b, "Villains:\n")
	for _, c := range cs.Villains {
		fmt.Fprintf(&b, "   %s\n", c.Name)
	}
	fmt.Fprint(&b, "Environments:\n")
	for _, c := range cs.Environments {
		fmt.Fprintf(&b, "   %s\n", c.Name)
	}
	return b.String()
}

// Setup is a specific game setup.
type Setup struct {
	Heroes      []*Card
	Villain     *Card
	Environment *Card
	PcPoints    int
	Difficulty  int
}

// String formats a setup for logging.
func (s *Setup) String() string {
	heroes := make([]string, len(s.Heroes))
	for i, h := range s.Heroes {
		heroes[i] = fmt.Sprintf("%s[%d]", h.Name, h.Points)
	}
	return fmt.Sprintf(
		"%s; %s[%d]; %s[%d]; %d heroes[%d]; difficulty=%d",
		strings.Join(heroes, ", "),
		s.Villain.Name,
		s.Villain.Points,
		s.Environment.Name,
		s.Environment.Points,
		len(heroes),
		s.PcPoints,
		s.Difficulty)
}

// makeSetup generates a random setup for the given card set and scores its difficulty.
func makeSetup(cs *CardSet, pc, pcpts int) (*Setup, error) {
	if pc > len(cs.Heroes) {
		return nil, errors.New("Too many players for the selected heroes.")
	}
	s := &Setup{PcPoints: pcpts, Difficulty: pcpts}
	for {
		bases := make(map[string]bool)
		for _, i := range pick(len(cs.Heroes), pc) {
			c := cs.Heroes[i]
			// if we have two heroes with the same base, try again.
			if bases[c.Base] {
				s.Heroes = nil
				break
			}
			bases[c.Base] = true
			s.Heroes = append(s.Heroes, c)
			s.Difficulty += c.Points
		}
		// keep trying until we get a list with no duplicate bases.
		if s.Heroes != nil {
			break
		}
	}
	s.Villain = cs.Villains[rand.Intn(len(cs.Villains))]
	s.Difficulty += s.Villain.Points
	s.Environment = cs.Environments[rand.Intn(len(cs.Environments))]
	s.Difficulty += s.Environment.Points
	return s, nil
}

// FindSetup finds a setup given a player count, loss pecrcentage, range,
// and set of expansions.
func FindSetup(pc, lp, rg int, exp []ExpansionType) (*Setup, int, error) {
	log.Printf("pc: %d, lp:%d, rg: %d, exp: %v", pc, lp, rg, exp)
	cs := GetCardSet(exp)
	min, max := sd.findDifficultyRange(lp)
	pcpts := sd.Difficulty.Nump[pc-3].Points
	for i := 0; ; i++ {
		if i >= 100000 {
			return nil, i + 1, errors.New("Couldn't find a setup with these parameters.")
		}
		s, err := makeSetup(cs, pc, pcpts)
		if err != nil {
			return nil, 0, err
		}
		if s.Difficulty >= min-rg && s.Difficulty <= max+rg {
			log.Printf("iterations: %d, setup: %s", i+1, s)
			return s, i + 1, nil
		}
	}
}

// findDifficultyRange finds the minimum and maximum difficulty scores for a given loss percentage.
func (sd *SentinelsData) findDifficultyRange(l int) (min, max int) {
	for _, v := range sd.Scale {
		if v.LossPct <= l-1 {
			break
		}
		if v.LossPct <= l {
			if max == 0 {
				max = v.Total
			}
			min = v.Total
		}
	}
	return
}

// pick picks m different random numbers between 0 and n-1.
func pick(n, m int) []int {
	if n <= 0 || m <= 0 || m > n {
		log.Fatalf("can't pick %d numbers between 0 and %d", m, n-1)
	}
	vals := make([]int, n)
	for i := 0; i < n; i++ {
		vals[i] = i
	}
	for i := 0; i < n; i++ {
		j := rand.Intn(n - i)
		vals[i], vals[i+j] = vals[i+j], vals[i]
	}
	result := make([]int, m)
	for i := 0; i < m; i++ {
		result[i] = vals[i]
	}
	return result
}

// original data at http://x.gray.org/sentinels.json
// some names normalized (e.g. "Silver Gulch, 1889")
var sdJson = `{
	"difficulty": {
		"hero": [
			{"name": "NightMist", "points": -10 },
			{"name": "Dark Watch NightMist", "points": 62, "base": "NightMist" },
			{"name": "Expatriette", "points": 28 },
			{"name": "Dark Watch Expatriette", "points": 42, "base": "Expatriette" },
			{"name": "Absolute Zero", "points": 25 },
			{"name": "Absolute Zero Elemental Wrath", "points": 31, "base": "Absolute Zero" },
			{"name": "Bunker", "points": 26 },
			{"name": "Bunker Engine of War", "points": 20, "base": "Bunker" },
			{"name": "GI Bunker", "points": -4, "base": "Bunker" },
			{"name": "Mr. Fixer", "points": 27 },
			{"name": "Dark Watch Fixer", "points": 10, "base": "Mr. Fixer" },
			{"name": "Setback", "points": 41 },
			{"name": "Dark Watch Setback", "points": 10, "base": "Setback" },
			{"name": "Haka", "points": -5 },
			{"name": "The Eternal Haka", "points": -7, "base": "Haka" },
			{"name": "Ra", "points": -7 },
			{"name": "Ra: Horus of Two Horizons", "points": -20, "base": "Ra" },
			{"name": "Wraith", "points": -6 },
			{"name": "Wraith: Price of Freedom", "points": -19, "base": "Wraith" },
			{"name": "Rook City Wraith", "points": 12, "base": "Wraith" },
			{"name": "Tempest", "points": -17 },
			{"name": "Tempest; Freedom", "points": 1, "base": "Tempest" },
			{"name": "Fanatic", "points": -1 },
			{"name": "Redeemer Fanatic", "points": -31, "base": "Fanatic" },
			{"name": "Tachyon", "points": -9 },
			{"name": "Team Leader Tachyon", "points": -71, "base": "Tachyon" },
			{"name": "Legacy", "points": -45 },
			{"name": "Young Legacy", "points": -24, "base": "Legacy" },
			{"name": "The Greatest Legacy", "points": -89, "base": "Legacy" },
			{"name": "The Visionary", "points": -14 },
			{"name": "Dark Visionary", "points": -37, "base": "The Visionary" },
			{"name": "Unity", "points": 5 },
			{"name": "Golem Unity", "points": -14, "base": "Unity" },
			{"name": "Parse", "points": 30 },
			{"name": "The Sentinels", "points": 30 },
			{"name": "The Argent Adept", "points": 11 },
			{"name": "The Naturalist", "points": 9 },
			{"name": "Chrono-Ranger", "points": -11 },
			{"name": "The Scholar", "points": -18 },
			{"name": "K.N.Y.F.E.", "points": -32 },
			{"name": "Omnitron-X", "points": -42 } ],
		"villain": [
			{"name": "Baron Blade", "points": -63, "advanced": 4, "advcount": 170 },
			{"name": "Mad Bomber Blade", "points": -37, "advanced": 12, "advcount": 61, "base": "Baron Blade" },
			{"name": "Gloomweaver", "points": -113, "advanced": -71, "advcount": 107 },
			{"name": "Skinwalker Gloomweaver", "points": 6, "advanced": -4, "advcount": 2, "base": "Gloomweaver" },
			{"name": "Spite", "points": -21, "advanced": -25, "advcount": 42 },
			{"name": "Agent of Gloom Spite", "points": 5, "advanced": 0, "advcount": 0, "base": "Spite" },
			{"name": "Omnitron", "points": 7, "advanced": 39, "advcount": 93 },
			{"name": "Cosmic Omnitron", "points": 63, "advanced": 82, "advcount": 51, "base": "Omnitron" },
			{"name": "The Chairman", "points": 76, "advanced": 46, "advcount": 66 },
			{"name": "Iron Legacy", "points": 70, "advanced": 105, "advcount": 62 },
			{"name": "The Matriarch", "points": 57, "advanced": 40, "advcount": 66 },
			{"name": "The Dreamer", "points": 39, "advanced": 52, "advcount": 55 },
			{"name": "Vengeful Five", "points": 36, "advanced": 0, "advcount": 0 },
			{"name": "Citizen Dawn", "points": 11, "advanced": 56, "advcount": 85 },
			{"name": "La Capitan", "points": 8, "advanced": 9, "advcount": 66 },
			{"name": "Grand Warlord Voss", "points": -21, "advanced": 71, "advcount": 115 },
			{"name": "Plague Rat", "points": -25, "advanced": 80, "advcount": 86 },
			{"name": "Apostate", "points": -37, "advanced": -46, "advcount": 102 },
			{"name": "Kismet", "points": -52, "advanced": -31, "advcount": 89 },
			{"name": "Miss Information", "points": -57, "advanced": 100, "advcount": 64 },
			{"name": "Akash'bhuta", "points": -60, "advanced": 20, "advcount": 94 },
			{"name": "The Ennead", "points": -80, "advanced": 66, "advcount": 97 },
			{"name": "Ambuscade", "points": -128, "advanced": -89, "advcount": 87 }		],
		"env": [
			{"name": "Rook City", "points": 74 },
			{"name": "Ruins of Atlantis", "points": 36 },
			{"name": "Insula Primalis", "points": 3 },
			{"name": "Pike Industrial Complex", "points": 3 },
			{"name": "Time Cataclysm", "points": 0 },
			{"name": "Tomb of Anubis", "points": -2 },
			{"name": "Wagner Mars Base", "points": -3 },
			{"name": "Silver Gulch, 1883", "points": -4 },
			{"name": "Mobile Defense Platform", "points": -4 },
			{"name": "Realm of Discord", "points": -8 },
			{"name": "Megalopolis", "points": -9 },
			{"name": "Freedom Tower", "points": -32 },
			{"name": "The Block", "points": -61 },
			{"name": "The Final Wasteland", "points": -74 }		],
		"nump": [
			{"name": "Three", "points": 42 },
			{"name": "Four", "points": -38 },
			{"name": "Five", "points": -42 }		]	},
	"scale": [
		{"total": 500, "losspct": 99 },
		{"total": 495, "losspct": 99 },
		{"total": 490, "losspct": 98 },
		{"total": 484, "losspct": 98 },
		{"total": 480, "losspct": 98 },
		{"total": 475, "losspct": 98 },
		{"total": 470, "losspct": 98 },
		{"total": 465, "losspct": 98 },
		{"total": 459, "losspct": 98 },
		{"total": 455, "losspct": 98 },
		{"total": 450, "losspct": 97 },
		{"total": 445, "losspct": 97 },
		{"total": 440, "losspct": 97 },
		{"total": 434, "losspct": 97 },
		{"total": 430, "losspct": 97 },
		{"total": 425, "losspct": 97 },
		{"total": 420, "losspct": 96 },
		{"total": 415, "losspct": 96 },
		{"total": 409, "losspct": 96 },
		{"total": 405, "losspct": 96 },
		{"total": 400, "losspct": 95 },
		{"total": 395, "losspct": 95 },
		{"total": 390, "losspct": 95 },
		{"total": 385, "losspct": 95 },
		{"total": 380, "losspct": 94 },
		{"total": 375, "losspct": 94 },
		{"total": 370, "losspct": 94 },
		{"total": 365, "losspct": 94 },
		{"total": 360, "losspct": 93 },
		{"total": 355, "losspct": 93 },
		{"total": 350, "losspct": 92 },
		{"total": 345, "losspct": 92 },
		{"total": 340, "losspct": 92 },
		{"total": 335, "losspct": 91 },
		{"total": 330, "losspct": 91 },
		{"total": 325, "losspct": 90 },
		{"total": 320, "losspct": 90 },
		{"total": 315, "losspct": 90 },
		{"total": 310, "losspct": 89 },
		{"total": 305, "losspct": 88 },
		{"total": 300, "losspct": 88 },
		{"total": 295, "losspct": 87 },
		{"total": 290, "losspct": 87 },
		{"total": 285, "losspct": 86 },
		{"total": 280, "losspct": 86 },
		{"total": 275, "losspct": 85 },
		{"total": 270, "losspct": 84 },
		{"total": 265, "losspct": 84 },
		{"total": 260, "losspct": 83 },
		{"total": 254, "losspct": 82 },
		{"total": 250, "losspct": 81 },
		{"total": 245, "losspct": 81 },
		{"total": 240, "losspct": 80 },
		{"total": 235, "losspct": 79 },
		{"total": 229, "losspct": 78 },
		{"total": 225, "losspct": 77 },
		{"total": 220, "losspct": 76 },
		{"total": 215, "losspct": 75 },
		{"total": 210, "losspct": 74 },
		{"total": 204, "losspct": 73 },
		{"total": 200, "losspct": 72 },
		{"total": 195, "losspct": 71 },
		{"total": 190, "losspct": 70 },
		{"total": 185, "losspct": 69 },
		{"total": 180, "losspct": 68 },
		{"total": 175, "losspct": 67 },
		{"total": 170, "losspct": 66 },
		{"total": 165, "losspct": 65 },
		{"total": 160, "losspct": 64 },
		{"total": 155, "losspct": 63 },
		{"total": 150, "losspct": 61 },
		{"total": 145, "losspct": 60 },
		{"total": 140, "losspct": 59 },
		{"total": 135, "losspct": 58 },
		{"total": 130, "losspct": 57 },
		{"total": 125, "losspct": 55 },
		{"total": 120, "losspct": 54 },
		{"total": 114, "losspct": 53 },
		{"total": 110, "losspct": 52 },
		{"total": 105, "losspct": 50 },
		{"total": 100, "losspct": 49 },
		{"total": 95, "losspct": 48 },
		{"total": 90, "losspct": 47 },
		{"total": 85, "losspct": 45 },
		{"total": 80, "losspct": 44 },
		{"total": 75, "losspct": 43 },
		{"total": 70, "losspct": 42 },
		{"total": 65, "losspct": 40 },
		{"total": 60, "losspct": 39 },
		{"total": 55, "losspct": 38 },
		{"total": 50, "losspct": 37 },
		{"total": 45, "losspct": 36 },
		{"total": 40, "losspct": 35 },
		{"total": 35, "losspct": 34 },
		{"total": 30, "losspct": 32 },
		{"total": 25, "losspct": 31 },
		{"total": 20, "losspct": 30 },
		{"total": 15, "losspct": 29 },
		{"total": 10, "losspct": 28 },
		{"total": 5, "losspct": 27 },
		{"total": 0, "losspct": 26 },
		{"total": -5, "losspct": 25 },
		{"total": -10, "losspct": 24 },
		{"total": -15, "losspct": 24 },
		{"total": -20, "losspct": 23 },
		{"total": -25, "losspct": 22 },
		{"total": -30, "losspct": 21 },
		{"total": -35, "losspct": 20 },
		{"total": -40, "losspct": 19 },
		{"total": -45, "losspct": 19 },
		{"total": -50, "losspct": 18 },
		{"total": -55, "losspct": 17 },
		{"total": -60, "losspct": 17 },
		{"total": -65, "losspct": 16 },
		{"total": -70, "losspct": 15 },
		{"total": -75, "losspct": 15 },
		{"total": -80, "losspct": 14 },
		{"total": -85, "losspct": 13 },
		{"total": -90, "losspct": 13 },
		{"total": -95, "losspct": 12 },
		{"total": -100, "losspct": 12 },
		{"total": -105, "losspct": 11 },
		{"total": -110, "losspct": 11 },
		{"total": -114, "losspct": 10 },
		{"total": -120, "losspct": 10 },
		{"total": -125, "losspct": 10 },
		{"total": -130, "losspct": 9 },
		{"total": -135, "losspct": 9 },
		{"total": -140, "losspct": 8 },
		{"total": -145, "losspct": 8 },
		{"total": -150, "losspct": 8 },
		{"total": -155, "losspct": 7 },
		{"total": -160, "losspct": 7 },
		{"total": -165, "losspct": 7 },
		{"total": -170, "losspct": 6 },
		{"total": -175, "losspct": 6 },
		{"total": -180, "losspct": 6 },
		{"total": -185, "losspct": 6 },
		{"total": -190, "losspct": 5 },
		{"total": -195, "losspct": 5 },
		{"total": -200, "losspct": 5 },
		{"total": -204, "losspct": 5 },
		{"total": -210, "losspct": 5 },
		{"total": -215, "losspct": 4 },
		{"total": -220, "losspct": 4 },
		{"total": -225, "losspct": 4 },
		{"total": -229, "losspct": 4 },
		{"total": -235, "losspct": 4 },
		{"total": -240, "losspct": 4 },
		{"total": -245, "losspct": 3 },
		{"total": -250, "losspct": 3 },
		{"total": -254, "losspct": 3 },
		{"total": -260, "losspct": 3 },
		{"total": -265, "losspct": 3 },
		{"total": -270, "losspct": 3 },
		{"total": -275, "losspct": 3 },
		{"total": -280, "losspct": 3 },
		{"total": -285, "losspct": 2 },
		{"total": -290, "losspct": 2 },
		{"total": -295, "losspct": 2 },
		{"total": -300, "losspct": 2 },
		{"total": -305, "losspct": 2 },
		{"total": -310, "losspct": 2 },
		{"total": -315, "losspct": 2 },
		{"total": -320, "losspct": 2 },
		{"total": -325, "losspct": 2 },
		{"total": -330, "losspct": 2 },
		{"total": -335, "losspct": 2 },
		{"total": -340, "losspct": 2 },
		{"total": -345, "losspct": 2 },
		{"total": -350, "losspct": 2 },
		{"total": -355, "losspct": 1 },
		{"total": -360, "losspct": 1 },
		{"total": -365, "losspct": 1 },
		{"total": -370, "losspct": 1 },
		{"total": -375, "losspct": 1 },
		{"total": -380, "losspct": 1 },
		{"total": -385, "losspct": 1 },
		{"total": -390, "losspct": 1 },
		{"total": -395, "losspct": 1 },
		{"total": -400, "losspct": 1 },
		{"total": -405, "losspct": 1 },
		{"total": -409, "losspct": 1 },
		{"total": -415, "losspct": 1 },
		{"total": -420, "losspct": 1 },
		{"total": -425, "losspct": 1 },
		{"total": -430, "losspct": 1 },
		{"total": -434, "losspct": 1 },
		{"total": -440, "losspct": 1 },
		{"total": -445, "losspct": 1 },
		{"total": -450, "losspct": 1 },
		{"total": -455, "losspct": 1 },
		{"total": -459, "losspct": 1 },
		{"total": -465, "losspct": 1 },
		{"total": -470, "losspct": 1 },
		{"total": -475, "losspct": 1 },
		{"total": -480, "losspct": 1 },
		{"total": -484, "losspct": 1 },
		{"total": -490, "losspct": 1 },
		{"total": -495, "losspct": 1 }	]
}`

var ExpansionCards = map[ExpansionType][]string{
	MiniExpansion: {
		"The Scholar",
		"Unity",
		"Ambuscade",
		"Miss Information",
		"The Final Wasteland",
		"Silver Gulch, 1883",
	},
	RookCity: {
		"Expatriette",
		"Mr. Fixer",
		"Pike Industrial Complex",
		"Rook City",
		"The Chairman",
		"The Matriarch",
		"Plague Rat",
		"Spite",
	},
	InfernalRelics: {
		"The Argent Adept",
		"NightMist",
		"Akash'bhuta",
		"Apostate",
		"Gloomweaver",
		"The Ennead",
		"Realm of Discord",
		"Tomb of Anubis",
	},
	ShatteredTimelines: {
		"Omnitron-X",
		"Chrono-Ranger",
		"Iron Legacy",
		"The Dreamer",
		"La Capitan",
		"Kismet",
		"Time Cataclysm",
		"The Block",
	},
	Vengeance: {
		"K.N.Y.F.E.",
		"The Sentinels",
		"The Naturalist",
		"Setback",
		"Parse",
		"Vengeful Five", // treated as one villain; here they are:
		//"Fright Train",
		//"Ermine",
		//"Proletariat",
		//"Friction",
		//"Baron Blade Vengeance",
		"Mobile Defense Platform",
		"Freedom Tower",
	},
	Promos: {
		"Dark Watch NightMist",
		"Dark Watch Expatriette",
		"Absolute Zero Elemental Wrath",
		"Bunker Engine of War",
		"GI Bunker",
		"Dark Watch Fixer",
		"Dark Watch Setback",
		"The Eternal Haka",
		"Ra: Horus of Two Horizons",
		"Wraith: Price of Freedom",
		"Rook City Wraith",
		"Tempest; Freedom",
		"Redeemer Fanatic",
		"Team Leader Tachyon",
		"Young Legacy",
		"The Greatest Legacy",
		"Dark Visionary",
		"Golem Unity",
		"Mad Bomber Blade",
		"Skinwalker Gloomweaver",
		"Agent of Gloom Spite",
		"Cosmic Omnitron",
	},
}
