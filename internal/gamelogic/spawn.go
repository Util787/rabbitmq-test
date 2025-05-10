package gamelogic

import (
	"errors"
	"fmt"
)

var PossibleUnits = map[string]struct{}{
	"infantry":  {},
	"cavalry":   {},
	"artillery": {},
}

var PossibleLocations = map[string]struct{}{
	"americas":   {},
	"europe":     {},
	"africa":     {},
	"asia":       {},
	"antarctica": {},
	"australia":  {},
}

func (gs *GameState) CommandSpawn(words []string) error {
	if len(words) < 3 { //idk why author put it here if I need to check the possible words anyway
		return errors.New("usage: spawn <location> <rank>")
	}

	locationName := words[1]
	locations := getAllLocations()
	if _, ok := locations[Location(locationName)]; !ok {
		return fmt.Errorf("error: %s is not a valid location", locationName)
	}

	rank := words[2]
	units := getAllRanks()
	if _, ok := units[UnitRank(rank)]; !ok {
		return fmt.Errorf("error: %s is not a valid unit", rank)
	}

	id := len(gs.getUnitsSnap()) + 1
	gs.addUnit(Unit{
		ID:       id,
		Rank:     UnitRank(rank),
		Location: Location(locationName),
	})

	fmt.Printf("Spawned a(n) %s in %s with id %v\n", rank, locationName, id)
	return nil
}
