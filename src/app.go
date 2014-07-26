package main

import (
	"errors"
	"flag"
	"fmt"
	"sentinels"
)

var (
	pc    int
	lp    int
	rg    int
	promo bool
	sd    *sentinels.SentinelsData
)

func main() {

	flag.IntVar(&pc, "pc", 3, "player count (3-5)")
	flag.IntVar(&lp, "lp", 50, "target loss percent (1-99, default 50")
	flag.IntVar(&rg, "rg", 10, "allowable difficulty variance around target loss percent (0-100, default 10")

	var err error

	if err = validateFlags(); err != nil {
		fmt.Println(err)
		return
	}

	s, i, err := sentinels.FindSetup(pc, lp, rg, []sentinels.ExpansionType{sentinels.BaseSet, sentinels.MiniExpansion})
	if err != nil {
		fmt.Println(err)
		return
	}
	if s != nil {
		fmt.Printf("\nFound in %d iterations:\n\n", i)
		fmt.Printf("%s", s)
	} else {
		fmt.Printf("\nNo setup found in %d iterations.\n", i)
	}

}

func validateFlags() error {
	flag.Parse()
	if pc < 3 || pc > 5 {
		return errors.New("player count must be between 3 and 5.")
	}

	if lp < 1 || lp > 99 {
		return errors.New("loss percentage must be between 1 and 99.")
	}

	if rg < 0 || rg > 100 {
		return errors.New("range must be between 0 and 100.")
	}
	return nil
}
