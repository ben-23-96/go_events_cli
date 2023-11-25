package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"

	"github.com/ben-23-96/go_events_cli/database"
	"github.com/ben-23-96/go_events_cli/eventsearch"
)

func main() {
	// load .env file
	err := godotenv.Load(".env")

	if err != nil {
		fmt.Printf("error loading envirment variables, ticketmaster api keys.")
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
	var cities string
	var genres string
	var dateFrom string
	var dateTo string
	// Set default values for dateFrom and dateTo
	defaultDateFrom := time.Now().Format(time.DateOnly)
	defaultDateTo := time.Now().AddDate(0, 1, 0).Format(time.DateOnly)
	// search subcommand flags
	eventSearchCmd.StringVar(&cities, "cities", "", "Indivual city or comma seperated list of cities. Example: \"Manchester,Brisol\"")
	eventSearchCmd.StringVar(&genres, "genres", "", "Indivual genre or subgenre comma seperated list. Example: \"Music,Sport\" Example2: \"Techno,Football\"")
	eventSearchCmd.StringVar(&dateFrom, "date-from", defaultDateFrom, "Date to start searching from in format YYYY-MM-DD. Default current date.")
	eventSearchCmd.StringVar(&dateTo, "date-to", defaultDateTo, "Date to start searching to in format YYYY-MM-DD. Default 1 month from current date.")
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
		handleSearchCmd(cities, genres, dateFrom, dateTo)
	default:
		fmt.Println("expected 'calendar' or 'search' subcommands")
		os.Exit(1)
	}
}

/*
Handles the calendar subcommand. Adds, deletes and displays events from the calendar. Calendar is a dynamodb table.
*/
func handleCalendarCmd(newEvents string, deleteEvent string, displayUpcomingEvents bool) {
	db, err := database.InitDB()

	if err != nil {
		fmt.Printf("error initializing database: %s", err)
		return
	}
	defer db.Close()

	// Check if new events were specified to be  add events
	if newEvents != "" {
		database.AddEvents(db, newEvents)
	}

	// Check if an event name was specified to be deleted delete event
	if deleteEvent != "" {
		database.DeleteEvent(db, deleteEvent)
	}

	// Check if the flag to display upcoming events is set then display the events
	if displayUpcomingEvents {
		events, err := database.GetEvents(db)
		if err != nil {
			fmt.Printf("Error retrieving events from database. Err: %s\n", err)
		}
		fmt.Print("Upcoming Events:\n\n")
		for _, event := range events {
			fmt.Printf("%s    %s\n", event.EventName, event.Date)
		}
	}
}

/*
Handles the search subcommand. Makes requests to the ticketmaster and skiddle API's searching for events using the paramters provided by the user in the CLI flags. Prints the found events in terminal checking if they do not clash with events in the calendar.
*/
func handleSearchCmd(cities string, genres string, dateFromString string, dateToString string) {
	db, err := database.InitDB()

	if err != nil {
		fmt.Printf("error initializing database: %s", err)
	}
	// get calendarEvents from calendar
	var calendarEvents []database.CalendarEvent
	calendarEvents, err = database.GetEvents(db)
	if err != nil {
		fmt.Printf("Error retrieving events from database. Err: %s\n", err)
	}

	// create new instance of api search struct with arguments
	eventSearch := eventsearch.ApiSearch{
		Cities:       cities,
		Genres:       genres,
		DateFrom:     dateFromString,
		DateTo:       dateToString,
		Ticketmaster: true,
		Skiddle:      true,
	}
	// search for events
	foundEvents := eventSearch.Search()
	// Create a map for calendar events
	calendarMap := make(map[time.Time]string)

	// Iterate through calendar events and populate the map
	for _, calendarEvent := range calendarEvents {
		date, _ := time.Parse(time.DateOnly, calendarEvent.Date)
		calendarMap[date] = calendarEvent.EventName
	}
	// Iterate through found events and check if they clash with a calendar event date with a map lookup
	for _, foundEvent := range foundEvents {
		// format date to string for print
		foundEventDate := foundEvent.Date.Format(time.DateOnly)
		if eventName, ok := calendarMap[foundEvent.Date]; !ok {
			// The date doesn't clash with a date in the calendar, print the event details
			fmt.Println("Event: ", foundEvent.Name)
			fmt.Println("city", foundEvent.City)
			fmt.Println("date", foundEventDate)
			fmt.Println("tickets", foundEvent.Tickets)
			fmt.Printf("genre: %s, subgenre: %s\n\n", foundEvent.Genre, foundEvent.Subgenre)
		} else {
			// The event date clashes with event in the calendar
			fmt.Printf("CALENDAR CLASH: %s (Event: %s)\n\n", foundEventDate, eventName)
		}
	}
}
