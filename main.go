package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"github.com/joho/godotenv"

	"github.com/ben-23-96/go_events_cli/database"
	"github.com/ben-23-96/go_events_cli/responsestructs"
)

func main() {
	// load .env file
	err := godotenv.Load(".env")

	if err != nil {
		fmt.Printf("error loading envirment variables, need skiddle and ticketmaster api keys.")
		return
	}
	// define calendar subcommand
	calendarCmd := flag.NewFlagSet("calendar", flag.ExitOnError)
	// calendar subcommand vars
	var newEvents string
	var deleteEvent string
	var displayUpcomingEvents bool
	// calendar subcommand flags
	calendarCmd.StringVar(&newEvents, "add-events", "", "Events and the date they are on to be added to calendar, comma seperated list in quotation marks. Example: \"event name, date, event name 2, date 2\"")

	calendarCmd.StringVar(&deleteEvent, "delete-event", "", "Delete a event from the calendar, provided the name of the event as it is stored. Example: \"event name\"")

	calendarCmd.BoolVar(&displayUpcomingEvents, "upcoming-events", false, "Display the upcoming events in the calendar.")

	// define search subcommand
	eventSearchCmd := flag.NewFlagSet("search", flag.ExitOnError)
	// search subcommand vars
	var searchEvents bool
	// search subcommand flags
	eventSearchCmd.BoolVar(&searchEvents, "search-events", false, "Search for upcoming events.")
	// exit if neither subcommand provided
	if len(os.Args) < 2 {
		fmt.Println("expected 'calendar' or 'search' subcommands")
		os.Exit(1)
	}
	// call relevant function to handle the arguments of relevant subcommands
	switch os.Args[1] {
	case "calendar":
		calendarCmd.Parse(os.Args[2:])
		handleCalendarCmd(newEvents, deleteEvent, displayUpcomingEvents)
	case "search":
		eventSearchCmd.Parse(os.Args[2:])
		handleSearchCmd(searchEvents)
	default:
		fmt.Println("expected 'foo' or 'bar' subcommands")
		os.Exit(1)
	}
}

func handleCalendarCmd(newEvents string, deleteEvent string, displayUpcomingEvents bool) {
	// Create a new instance of the CalendarDB struct
	calendarDB := database.CalendarDB{}

	// Initialize the AWS session and DynamoDB client
	calendarDB.NewSession()

	// Check if new events were specified to be added
	if newEvents != "" {
		calendarDB.AddEvents(newEvents)
	}

	// Check if an event name was specified to be deleted
	if deleteEvent != "" {
		calendarDB.DeleteEvent(deleteEvent)
	}

	// Check if the flag to display upcoming events is set
	if displayUpcomingEvents {
		calendarDB.GetEvents()
	}
}

func handleSearchCmd(searchEvents bool) {
	if searchEvents {
		eventSearch()
	}
}

func eventSearch() {
	apiKey := os.Getenv("ticketmasterAPIKey")
	// URL of the API or web service you want to request
	requestUrl := fmt.Sprintf("https://app.ticketmaster.com/discovery/v2/events.json?apikey=%s", apiKey)

	city := "manchester"
	classificationName := "Music"
	startDateTime := "2023-10-20T00:00:00Z"
	endDateTime := "2023-10-27T00:00:00Z"
	pageSize := "20"
	requestUrl += fmt.Sprintf("&city=%s", url.QueryEscape(city))
	requestUrl += fmt.Sprintf("&classificationName=%s", url.QueryEscape(classificationName))
	requestUrl += fmt.Sprintf("&startDateTime=%s", url.QueryEscape(startDateTime))
	requestUrl += fmt.Sprintf("&endDateTime=%s", url.QueryEscape(endDateTime))
	requestUrl += fmt.Sprintf("&size=%s", url.QueryEscape(pageSize))

	makeRequest(requestUrl)

}

func makeRequest(requestUrl string) {
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

	body, err := io.ReadAll(response.Body)
	if err != nil {
		fmt.Println("error reading response body:", err)
		return
	}

	resStruct := responsestructs.Events{}
	if err := json.Unmarshal(body, &resStruct); err != nil {
		fmt.Println("Error unmarshaling JSON:", err)
		return
	}

	// Access and use the data
	fmt.Println("Page info:", resStruct.Page)
	for _, event := range resStruct.Embedded.Events {
		fmt.Println("Event: ", event.Name)
	}

	fmt.Println("next page:", resStruct.Links.Next)

	if resStruct.Links.Next.Href != "" {
		fmt.Println("NEXT PAGGE")
		u, err := url.Parse(requestUrl)
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
		nexPageURL := u.String()
		makeRequest(nexPageURL)
	}
}
