package eventsearch

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"
)

type ApiSearch struct {
	Cities             string
	Genres             string
	DateFrom           string
	DateTo             string
	Ticketmaster       bool
	Skiddle            bool
	responseStruct     Response
	FoundEventsChannel chan []FoundEvent
	FoundEvents        []FoundEvent
}

func (s *ApiSearch) Search() {

	s.validateDates()

	// Create a channel for receiving the results.
	s.FoundEventsChannel = make(chan []FoundEvent)

	ticketmasterUrl, ticketmasterUnmarshallFunc := s.setApi(s.Ticketmaster, false)

	// Launch the method in a goroutine.
	go s.makeRequest(ticketmasterUrl, ticketmasterUnmarshallFunc)

	s.FoundEvents = <-s.FoundEventsChannel

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
}

func (s *ApiSearch) setApi(ticketmaster bool, skiddle bool) (requestUrl string, unmarshallFunction UnmarshalFunction) {
	s.responseStruct = Response{}
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
		unmarshalFunction := s.responseStruct.UnmarshalTicketmasterJSON

		return requestUrl, unmarshalFunction
	}
	return "", nil
}

func (s *ApiSearch) makeRequest(requestUrl string, unmarshalFunction UnmarshalFunction) {
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
	// unmarshall the response into Response struct
	err = unmarshalFunction(body)
	if err != nil {
		fmt.Println("error reading response body:", err)
		return
	}
	// send []FoundEvents to channel
	s.FoundEventsChannel <- s.responseStruct.Events
}
