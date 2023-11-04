package eventsearch

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"
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
}

func (s *ApiSearch) Search() []FoundEvent {

	s.validateDates()

	// Create a channel for receiving the results from api's
	s.FoundEventsChannel = make(chan []FoundEvent, 2)
	wg := &sync.WaitGroup{}
	wg.Add(2)

	// create a url and set relevant unmarshalling function for both API's
	ticketmasterUrl, ticketmasterUnmarshallFunc := s.setApi(s.Ticketmaster, false)
	skiddleUrl, skiddleUnmarshallFunc := s.setApi(false, s.Skiddle)

	// Launch the method in a goroutine.
	go s.makeRequest(ticketmasterUrl, ticketmasterUnmarshallFunc, wg)
	go s.makeRequest(skiddleUrl, skiddleUnmarshallFunc, wg)

	// Wait for both goroutines to complete.
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
		requestUrl += fmt.Sprintf("&classificationName=%s", url.QueryEscape(s.Genres))
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
		requestUrl += fmt.Sprintf("&longitude=%s", url.QueryEscape("-2.2446"))
		requestUrl += fmt.Sprintf("&latitude=%s", url.QueryEscape("53.4839"))
		requestUrl += fmt.Sprintf("&radius=%s", url.QueryEscape("8"))
		requestUrl += fmt.Sprintf("&minDate=%s", url.QueryEscape(s.dateFromSkiddle))
		requestUrl += fmt.Sprintf("&maxDate=%s", url.QueryEscape(s.dateToSkiddle))
		requestUrl += fmt.Sprintf("&description=%s", url.QueryEscape("1"))
		//requestUrl += fmt.Sprintf("&limit=%s", url.QueryEscape("100"))

		// set the function used for unmarshalling the json response
		unmarshalFunction := UnmarshalSkiddleJSON
		fmt.Println(requestUrl)
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
