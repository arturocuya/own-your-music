package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
	"sync/atomic"

	"github.com/Rhymond/go-money"
	"github.com/labstack/echo/v4"
	"github.com/zmb3/spotify/v2"

	"ownyourmusic/providers"
	"ownyourmusic/templates"
	"ownyourmusic/types"
)

var loadSpotifySongsChan = make(chan []types.InputTrack)
var foundSongChan = make(chan types.PurchaseableTrack)
var flushCompleteChan = make(chan struct{})

var totalInvestment = make(map[string]*money.Money)

var totalSongs = 0
var processedSongs int64

func updateSpotifyCredentials(c echo.Context) error {
	clientId := c.FormValue("clientId")
	clientSecret := c.FormValue("clientSecret")

	err := SetKeyValue(KEY_SPOTIFY_CLIENT_ID, clientId)

	if err != nil {
		return err
	}

	err = SetKeyValue(KEY_SPOTIFY_CLIENT_SECRET, clientSecret)

	if err != nil {
		return err
	}

	return c.NoContent(http.StatusOK)
}

func spotifyAuthCallback(c echo.Context) error {
	state, err := GetKeyValue(KEY_SPOTIFY_AUTH_STATE)

	if err != nil {
		return c.String(http.StatusForbidden, "could not get state")
	}

	if receivedState := c.FormValue("state"); receivedState != state {
		return c.String(http.StatusOK, "state mismatch")
	}

	auth, err := getSpotifyAuth()
	if err != nil {
		return c.String(http.StatusInternalServerError, "could not get spotify auth object")
	}

	token, err := auth.Token(c.Request().Context(), state, c.Request())

	if err != nil {
		return c.String(http.StatusForbidden, "could not get token")
	}

	err = SetSpotifyToken(token)

	if err != nil {
		return c.String(http.StatusForbidden, "could not save token")
	}

	return c.NoContent(http.StatusOK)
}

func serverSentEvents(c echo.Context) error {
	c.Response().Header().Set(echo.HeaderContentType, "text/event-stream")
	c.Response().Header().Set(echo.HeaderCacheControl, "no-cache")
	c.Response().Header().Set(echo.HeaderConnection, "keep-alive")

	c.Response().Header().Set("Access-Control-Allow-Origin", "*")
	for {
		select {
		case <-c.Request().Context().Done():
			return nil
		case tracks := <-loadSpotifySongsChan:
			log.Println("sse tracks")
			var tracksAndMatches []types.TrackAndMatch

			for _, track := range tracks {
				tracksAndMatches = append(tracksAndMatches, types.TrackAndMatch{
					Track: track,
					// TODO: insert cached match
					Match: types.PurchaseableTrack{},
				})
			}

			var buf bytes.Buffer

			tmpl := templates.ManyTracks(tracksAndMatches)
			tmpl.Render(context.Background(), &buf)

			content := strings.ReplaceAll(buf.String(), "\n", "")
			content = strings.ReplaceAll(content, "\t", "")

			data := fmt.Sprintf("data: %v\n\n", content)

			if _, err := c.Response().Write([]byte(data)); err != nil {
				return err
			}
			c.Response().Flush()
		case foundMatch := <-foundSongChan:
			log.Println("sse found match")
			type investmentEntry struct {
				currency string
				value    *money.Money
			}

			var investments []investmentEntry
			for currency, money := range totalInvestment {
				investments = append(investments, investmentEntry{
					currency: currency,
					value:    money,
				})
			}

			sort.Slice(investments, func(i, j int) bool {
				return investments[i].value.Amount() > investments[j].value.Amount()
			})

			var investmentTexts []string
			for _, entry := range investments {
				investmentTexts = append(investmentTexts, fmt.Sprintf("%s %s", entry.currency, entry.value.Display()))
			}

			component := templates.ResultFound(foundMatch, strings.Join(investmentTexts, " + "))

			var buf bytes.Buffer

			component.Render(context.Background(), &buf)

			content := strings.ReplaceAll(buf.String(), "\n", "")
			content = strings.ReplaceAll(content, "\t", "")

			content += fmt.Sprintf("<div id=\"progress\" hx-swap-oob=\"true\">%v / %v</div>", processedSongs, totalSongs)

			data := fmt.Sprintf("data: %v\n\n", content)

			if _, err := c.Response().Write([]byte(data)); err != nil {
				return err
			}
			c.Response().Flush()
			flushCompleteChan <- struct{}{}
		}
	}
}

func loadSpotifySongs(c echo.Context) error {
	auth, err := getSpotifyAuth()

	if err != nil {
		return err
	}

	token, err := GetSpotifyToken()

	if err != nil {
		return err
	}

	go func() {
		client := spotify.New(auth.Client(c.Request().Context(), token))

		userTracks, err := client.CurrentUsersTracks(context.Background())

		if err != nil {
			log.Fatal("error getting current user tracks at offset 0: ", err)
		}

		var tracks []types.InputTrack

		for i := range len(userTracks.Tracks) {
			track := userTracks.Tracks[i]
			tracks = append(tracks, types.InputTrack{
				ProviderName: "spotify", // TODO: generalize
				ProviderId: string(track.ID),
				AddedAt: track.AddedAt,
				Name:   track.Name,
				Artist: track.Artists[0].Name,
				Album:  track.Album.Name,
			})
			log.Printf("Retrieved track #%d \"%s\" by %s \n", i+1, track.Name, track.Artists[0].Name)
		}

		loadSpotifySongsChan <- tracks

		// TODO: need to clear on client too
		ClearSpotifySongs()

		SaveSpotifySongs(tracks)

		offset := len(userTracks.Tracks)

		for userTracks.Next != "" {
			userTracks, err = client.CurrentUsersTracks(context.Background(), spotify.Offset(offset))

			if err != nil {
				log.Fatalf("error getting current user tracks at offset %d: %s", offset, err)
			}

			tracks = tracks[:0]
			for i := range len(userTracks.Tracks) {
				track := userTracks.Tracks[i]
				tracks = append(tracks, types.InputTrack{
					ProviderName: "spotify", // TODO: generalize
					ProviderId: string(track.ID),
					AddedAt: track.AddedAt,
					Name:   track.Name,
					Artist: track.Artists[0].Name,
					Album:  track.Album.Name,
				})
				log.Printf("Retrieved track #%d \"%s\" by %s \n", i+offset+1, track.Name, track.Artists[0].Name)
			}

			loadSpotifySongsChan <- tracks

			SaveSpotifySongs(tracks)

			offset += len(userTracks.Tracks)
		}
	}()

	return c.NoContent(http.StatusOK)
}

func findSongs(c echo.Context) error {
	db, err := OpenDatabase()

	if err != nil {
		log.Fatal("error opening database: ", err)
	}

	defer db.Close()

	var tracks []types.InputTrack

	err = db.Select(&tracks, "select * from spotify_songs order by \"idx\" asc")

	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("error fetching existing spotify songs: %v", err))
	}

	totalSongs = len(tracks)
	processedSongs = 0

	go func(mainProvider providers.Provider) {
		// tried parallelizing this at num procs workers, but it would reach
		// too many requests quickly, and with the proper request spacing
		// it wasn't that different than searching sequentially
		for _, track := range tracks {
			result := mainProvider.FindSong(&track)

			if result != nil {
				if result.Price != nil {
					currencyCode := result.Price.Currency().Code
					if _, exists := totalInvestment[currencyCode]; !exists {
						totalInvestment[currencyCode] = result.Price
					} else {
						existingPrice := totalInvestment[currencyCode]
						moMoney, _ := existingPrice.Add(result.Price)
						totalInvestment[currencyCode] = moMoney
					}

					log.Println("updated prices")
					for key, value := range totalInvestment {
						log.Printf("%v: %v\n", key, value.Display())
					}
				}
				foundSongChan <- *result
			} else {
				log.Println("no match found for idx", track.ComposedId())
				foundSongChan <- types.PurchaseableTrack{
					SongUrl: "",
				}
			}

			atomic.AddInt64(&processedSongs, 1)

			// wait for SSE to flush message to client before attempting to fetch another value
			// otherwise multiple writes can happen to the same response before flushing it, which will corrupt it
			<-flushCompleteChan
		}
	}(providers.BandcampProvider{})

	return c.NoContent(http.StatusOK)
}
