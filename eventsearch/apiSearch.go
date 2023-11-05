package eventsearch

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
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
	FoundEventsChannel chan []FoundEvent
	FoundEvents        []FoundEvent
	dateFromSkiddle    string
	dateToSkiddle      string
	longitude          string
	latitude           string
	ticketmasterGenre  string
	skiddleGenreID     string
}

func (s *ApiSearch) Search() []FoundEvent {

	s.validateDates()
	s.matchGenres()
	// find lng + lat of cities for use in indivual requests to skiddle API
	citiesLngLat := s.skiddleLongLat()
	wgValue := len(citiesLngLat) + 1
	// Create a channel for receiving the results from api's
	s.FoundEventsChannel = make(chan []FoundEvent, wgValue)
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
	close(s.FoundEventsChannel)

	// Collect results from the channel.
	var foundEvents []FoundEvent
	for events := range s.FoundEventsChannel {
		foundEvents = append(foundEvents, events...)
	}

	return foundEvents

}

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
		requestUrl += fmt.Sprintf("&g=%s", url.QueryEscape(s.skiddleGenreID))
		requestUrl += fmt.Sprintf("&description=%s", url.QueryEscape("1"))
		//requestUrl += fmt.Sprintf("&limit=%s", url.QueryEscape("100"))

		// set the function used for unmarshalling the json response
		unmarshalFunction := UnmarshalSkiddleJSON

		return requestUrl, unmarshalFunction
	}
	return "", nil
}

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
	s.FoundEventsChannel <- events
	// signal done to waitgroup
	wg.Done()
}

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

func (s *ApiSearch) matchGenres() {
	// Read the JSON data from the file
	genresJSON, err := os.ReadFile("eventsearch/genres.json")
	if err != nil {
		panic(err)
	}

	// Unmarshal the JSON data into the Segments structure
	var genres GenreJSON
	if err := json.Unmarshal(genresJSON, &genres); err != nil {
		panic(err)
	}

	// find the ticketmaster genre that matches user input closest set to ticketmasterGenre attribute
	s.ticketmasterGenre, _ = edlib.FuzzySearch(s.Genres, genres.Ticketmaster.Genres, edlib.Levenshtein)

	var stringSimilarity float32
	stringSimilarity = 0.0
	// find the skiddle genre that matches user input closest set its ID to skiddleGenreID sttribute
	for _, skiddleGenre := range genres.Skiddle.Genres {
		res, _ := edlib.StringsSimilarity(s.Genres, skiddleGenre.Name, edlib.Levenshtein)
		if res == 1 {
			s.skiddleGenreID = skiddleGenre.ID
			break
		}
		if res > stringSimilarity {
			stringSimilarity = res
			s.skiddleGenreID = skiddleGenre.ID
		}
	}

	fmt.Println(stringSimilarity)
}
