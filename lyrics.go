package main

import (
	"fmt"

	"github.com/mamal72/golyrics"
)

func lyricsByAT(artist, track string) string {
	suggestions, err := golyrics.SearchTrackByArtistAndName(artist, track)

	if len(suggestions) == 0 {
		return "No tracks found, sorry."
	}

	if err != nil {
		return "Whoops, error happened"
	}

	var tc = suggestions[0]

	err = tc.FetchLyrics()

	if err != nil {
		return "Can't fetch lyrics, sorry."
	}

	return fmt.Sprintf("*%s - %s*:\n%s", tc.Artist, tc.Name, tc.Lyrics)
}
