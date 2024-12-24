package main

import (
	"fmt"
	"testing"
)

func TestHappyPath(t *testing.T) {
	song := SpotifySong{
		Index:  1,
		Name:   "The Bay",
		Artist: "Metronomy",
		Album:  "The English Riviera",
	}

	match := findSongInBandcamp(&song)

	if match == nil {
		t.Fatalf("match not found for %+v\n", song)
	}

	fmt.Printf("match found: %+v\n", match)
}

// func TestSpecialCharacters(t *testing.T) {
// 	song := SpotifySong{
// 		Index:  1,
// 		Name:   "still feel.",
// 		Artist: "half•alive",
// 		Album:  "Now, Not Yet",
// 	}
// }
