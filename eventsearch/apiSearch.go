package eventsearch

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

type ApiSearch struct {
	Cities     string
	Genres     string
	DateFrom   string
	DateTo     string
	requestUrl string
}

func (s *ApiSearch) Search() {

	s.validateDates()

	apiKey := os.Getenv("ticketmasterAPIKey")
	// URL of the API or web service you want to request
	s.requestUrl = fmt.Sprintf("https://app.ticketmaster.com/discovery/v2/events.json?apikey=%s", apiKey)
	// set query params
	s.requestUrl += fmt.Sprintf("&city=%s", url.QueryEscape(s.Cities))
	s.requestUrl += fmt.Sprintf("&classificationName=%s", url.QueryEscape(s.Genres))
	s.requestUrl += fmt.Sprintf("&startDateTime=%s", url.QueryEscape(s.DateFrom))
	s.requestUrl += fmt.Sprintf("&endDateTime=%s", url.QueryEscape(s.DateTo))
	s.requestUrl += fmt.Sprintf("&size=%s", url.QueryEscape("100"))

	s.makeRequest()

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

func (s *ApiSearch) makeRequest() {
	// Send an HTTP GET request
	response, err := http.Get(s.requestUrl)
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
	// unmarshall the response body json into events struct
	resStruct := Events{}
	if err := json.Unmarshal(body, &resStruct); err != nil {
		fmt.Println("Error unmarshaling JSON:", err)
		return
	}
	// Access and use the data
	fmt.Println("Page info:", resStruct.Page)
	for _, event := range resStruct.Embedded.Events {
		fmt.Println("Event: ", event.Name)
		fmt.Println("city", event.Embedded.Venues[0].City)
		fmt.Println("date", event.Dates.Start.LocalDate)
		fmt.Println("tickets", event.URL)
		fmt.Printf("genre: %s, subgenre: %s\n\n", event.Classifications[0].Segment.Name, event.Classifications[0].Genre.Name)
	}
	// check if there are multiple pages to the response
	if resStruct.Links.Next.Href != "" {
		// parse the request url
		u, err := url.Parse(s.requestUrl)
		if err != nil {
			fmt.Println("Error parsing URL for next page:", err)
			return
		}
		nextPageNum := resStruct.Page.Number + 1
		nextPageNumString := strconv.Itoa(nextPageNum)
		// Get the query parameters
		queryValues := u.Query()
		// Set a new value for the "page" parameter
		queryValues.Set("page", nextPageNumString)
		// Update the URL's RawQuery with the modified query parameters
		u.RawQuery = queryValues.Encode()
		// Reassemble the URL
		s.requestUrl = u.String()
		// make request for next page of response
		s.makeRequest()
	}
}
