package eventsearch

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/codingsince1985/geo-golang"
	"github.com/codingsince1985/geo-golang/opencage"
	"github.com/hbollon/go-edlib"
)

type ApiSearch struct {
	Cities             string
	Genres             string
	DateFrom           string
	DateTo             string
	Ticketmaster       bool
	Skiddle            bool
	foundEventsChannel chan []FoundEvent
	dateFromSkiddle    string
	dateToSkiddle      string
	longitude          string
	latitude           string
	ticketmasterGenre  string
	skiddleGenreID     string
}

/*
Searches for events from the ticketmaster and skiddle API's using parameter provided in APISearch{}
Returns:
- []FoundEvent: A slice of FoundEvent{} that contain all relevant information of events returned from the API's.
*/
func (s *ApiSearch) Search() []FoundEvent {
	// check dates are in valid format and time
	s.validateDates()
	// find best match from user input genres to avaible genre params for skiddle and ticketmaster API's
	s.matchGenres()
	// find lng + lat of cities for use in indivual requests to skiddle API
	citiesLngLat := s.skiddleLongLat()
	wgValue := len(citiesLngLat) + 1
	// Create a channel for receiving the results from api's
	s.foundEventsChannel = make(chan []FoundEvent, wgValue)
	// create wait group
	wg := &sync.WaitGroup{}
	// set waitgroup limit to number of skiddle requests + ticketmaster request
	wg.Add(wgValue)

	// set ticketmaster url and unmarshalling function, then make request in goroutine (API handles list of cities)
	ticketmasterUrl, ticketmasterUnmarshallFunc := s.setApi(s.Ticketmaster, false)
	go s.makeRequest(ticketmasterUrl, ticketmasterUnmarshallFunc, wg)
	// set skiddle url and unmarshalling function, then make request in goroutine for each city
	for _, location := range citiesLngLat {
		s.longitude = fmt.Sprintf("%f", location.Lng)
		s.latitude = fmt.Sprintf("%f", location.Lat)
		skiddleUrl, skiddleUnmarshallFunc := s.setApi(false, s.Skiddle)
		go s.makeRequest(skiddleUrl, skiddleUnmarshallFunc, wg)
	}

	// Wait for goroutines to complete.
	wg.Wait()
	// Close the FoundEventsChannel after all goroutines are done.
	close(s.foundEventsChannel)

	// Collect results from the channel.
	var foundEvents []FoundEvent
	for events := range s.foundEventsChannel {
		foundEvents = append(foundEvents, events...)
	}

	//slices.SortFunc(foundEvents, func(a, b T) int { return a.Date.Compare(B.Date) })
	sort.Slice(foundEvents, func(i, j int) bool {
		return foundEvents[i].Date.Before(foundEvents[j].Date)
	})

	return foundEvents

}

/*
Validates the user provided dateFrom and DateTo strings are in a valid dates and formats them in the required way to be used as query params in the API requests.
*/
func (s *ApiSearch) validateDates() {
	// Check date in correct format
	dateFrom, err := time.Parse(time.DateOnly, s.DateFrom)
	if err != nil {
		fmt.Printf("Error with date-from format: %s err: %s\n", s.DateFrom, err)
		return
	}
	dateTo, err := time.Parse(time.DateOnly, s.DateTo)
	if err != nil {
		fmt.Printf("Error with date-from format: %s err: %s\n", s.DateTo, err)
		return
	}

	// Check if dateFrom is later than the current date
	current := time.Now().Truncate(24 * time.Hour)
	if dateFrom.Before(current) {
		fmt.Println("dateFrom must be later than the current date")
		return
	}
	// Check if dateFrom is earlier than dateTo
	if dateFrom.After(dateTo) {
		fmt.Println("dateFrom must be later than the dateTo")
		return
	}

	// Format the date as a string in "YYYY-MM-DDT00:00:00Z" format
	s.DateFrom = dateFrom.Format(time.RFC3339)
	s.DateTo = dateTo.Format(time.RFC3339)
	// Format the date as a string in "YYYY-MM-DD" format for skiddle
	s.dateFromSkiddle = dateFrom.Format(time.DateOnly)
	s.dateToSkiddle = dateTo.Format(time.DateOnly)
}

/*
Uses the levenshtien algorithm to find the best match of the genres provided in the Genres attribute, and the accepted format of that genre to be sent as a query param to the relevant API. Accepted formats are stored in genres.json. Skiddle API requires genreID and ticketmaster API requires spelling + wording to be the same as exspected. Sets the ticketmasterGenres and skiddleGenreID attributes as string comma seperated lists.
*/
func (s *ApiSearch) matchGenres() {
	// Read the JSON data from the genres.json file
	genresJSON, err := os.ReadFile("eventsearch/genres.json")
	if err != nil {
		panic(err)
	}

	// Unmarshal the JSON data into the genres structure
	var genres GenreJSON
	if err := json.Unmarshal(genresJSON, &genres); err != nil {
		panic(err)
	}

	// Split the user input string by commas
	userGenresList := strings.Split(s.Genres, ",")

	var stringSimilarity float32
	var bestMatchSkiddleID string
	stringSimilarity = 0.0
	// iterate over genre inputs
	for _, userGenre := range userGenresList {
		// find the ticketmaster genre that matches user input closest add to ticketmasterGenre attribute as comma seperated list
		bestMatchTicketmasterGenre, _ := edlib.FuzzySearch(userGenre, genres.Ticketmaster.Genres, edlib.Levenshtein)
		if s.ticketmasterGenre != "" {
			s.ticketmasterGenre = s.ticketmasterGenre + "," + bestMatchTicketmasterGenre
		} else {
			s.ticketmasterGenre = bestMatchTicketmasterGenre
		}

		// find the skiddle genre that matches user input closest set its ID to skiddleGenreID sttribute
		for _, skiddleGenre := range genres.Skiddle.Genres {
			similarityRes, _ := edlib.StringsSimilarity(userGenre, skiddleGenre.Name, edlib.Levenshtein)
			// if strings match exactly set best match break loop
			if similarityRes == 1 {
				bestMatchSkiddleID = skiddleGenre.ID
				break
			}
			// current best match
			if similarityRes > stringSimilarity {
				stringSimilarity = similarityRes
				bestMatchSkiddleID = skiddleGenre.ID
			}
		}
		// add found genre id as part of comma seperated list to skiddleGenreID attribute and reset stringSimilarity and best match vars
		if s.skiddleGenreID != "" {
			s.skiddleGenreID = s.skiddleGenreID + "," + bestMatchSkiddleID
		} else {
			s.skiddleGenreID = bestMatchSkiddleID
		}
		stringSimilarity = 0.0
		bestMatchSkiddleID = ""
	}
	fmt.Println(s.ticketmasterGenre)
}

/*
Uses the opencage API to find longitide and latitude of cities provided in Cities attribute.
Returns:
- []geo.Location: slice containing the geo data on the provided cities.
*/
func (s *ApiSearch) skiddleLongLat() []geo.Location {
	// geocoder SDK for finding lng and lat of city
	geocoder := opencage.Geocoder(os.Getenv("opencageAPIKey"))
	var citiesLngLat []geo.Location
	// Split the input string by commas
	citiesList := strings.Split(s.Cities, ",")
	// find long and lat of each city append to slice
	for _, city := range citiesList {
		location, _ := geocoder.Geocode(city)
		if location != nil {
			citiesLngLat = append(citiesLngLat, *location)
		}
	}
	return citiesLngLat
}

/*
Creates a url with the user provided parameters for either ticketmaster or skiddle API, sets the relevant function to be used for unmarshall the API response.
Parameters:
- ticketmaster: bool: if true set for ticketmaster API.
- skiddle: bool: if true set for skiddle API.const
Returns:
- requestUrl: string: url to be used in a API request with relevant query parameters.
- unmarshallFunction: a function used to unmarshall the response json for a particular API request.
*/
func (s *ApiSearch) setApi(ticketmaster bool, skiddle bool) (requestUrl string, unmarshallFunction UnmarshalFunction) {
	if ticketmaster {
		apiKey := os.Getenv("ticketmasterAPIKey")
		// set base ticketmaster url
		requestUrl := fmt.Sprintf("https://app.ticketmaster.com/discovery/v2/events.json?apikey=%s", apiKey)
		// set query params for api request
		requestUrl += fmt.Sprintf("&city=%s", url.QueryEscape(s.Cities))
		requestUrl += fmt.Sprintf("&classificationName=%s", url.QueryEscape(s.ticketmasterGenre))
		requestUrl += fmt.Sprintf("&startDateTime=%s", url.QueryEscape(s.DateFrom))
		requestUrl += fmt.Sprintf("&endDateTime=%s", url.QueryEscape(s.DateTo))
		requestUrl += fmt.Sprintf("&size=%s", url.QueryEscape("100"))

		// set the function used for unmarshalling the json response
		unmarshalFunction := UnmarshalTicketmasterJSON
		fmt.Println(requestUrl)
		return requestUrl, unmarshalFunction
	} else if skiddle {
		apiKey := os.Getenv("skiddleAPIKey")
		// set base skiddle url
		requestUrl := fmt.Sprintf("https://www.skiddle.com/api/v1/events/search/?api_key=%s", apiKey)
		// set query params for api request
		requestUrl += fmt.Sprintf("&longitude=%s", url.QueryEscape(s.longitude))
		requestUrl += fmt.Sprintf("&latitude=%s", url.QueryEscape(s.latitude))
		requestUrl += fmt.Sprintf("&radius=%s", url.QueryEscape("8"))
		requestUrl += fmt.Sprintf("&minDate=%s", url.QueryEscape(s.dateFromSkiddle))
		requestUrl += fmt.Sprintf("&maxDate=%s", url.QueryEscape(s.dateToSkiddle))
		requestUrl += fmt.Sprintf("&description=%s", url.QueryEscape("1"))
		if s.skiddleGenreID != "" {
			requestUrl += fmt.Sprintf("&g=%s", url.QueryEscape(s.skiddleGenreID))
		}
		//requestUrl += fmt.Sprintf("&limit=%s", url.QueryEscape("100"))
		fmt.Printf("\n\n%s\n\n", requestUrl)
		fmt.Println(s.skiddleGenreID)
		// set the function used for unmarshalling the json response
		unmarshalFunction := UnmarshalSkiddleJSON

		return requestUrl, unmarshalFunction
	}
	return "", nil
}

/*
Makes a request to a API, unmarshalls the response into []FoundEvent and sends the unmarshalled data back to foundEventsChannel.
Parameters:
- requestUrl: string: the url to make the request to.
- unmarshallFunction: the function used to unmarsshall the API json response.const
- wg: waitGroup: the wait group of the goroutine
*/
func (s *ApiSearch) makeRequest(requestUrl string, unmarshalFunction UnmarshalFunction, wg *sync.WaitGroup) {
	// Send an HTTP GET request
	response, err := http.Get(requestUrl)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer response.Body.Close()

	// Check the response status code
	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		fmt.Printf("Request failed with status: %d, body:%s", response.StatusCode, body)
		return
	}
	// read the response body
	body, err := io.ReadAll(response.Body)
	if err != nil {
		fmt.Println("error reading response body:", err)
		return
	}
	// unmarshall the response into []FoundEvents
	events, err := unmarshalFunction(body)
	if err != nil {
		fmt.Println("error reading response body:", err)
		return
	}
	// send []FoundEvents to channel
	s.foundEventsChannel <- events
	// signal done to waitgroup
	wg.Done()
}
