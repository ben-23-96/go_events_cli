package main

import (
	"flag"
	"fmt"

	"github.com/ben-23-96/go_events_cli/database"
)

func main() {
	// Define command-line flags
	var newEvents string
	var deleteEvent string
	var displayUpcomingEvents bool

	flag.StringVar(&newEvents, "add-events", "", "Events and the date they are on to be added to calendar, comma seperated list in quotation marks. Example: \"event name, date, event name 2, date 2\"")

	flag.StringVar(&deleteEvent, "delete-event", "", "Delete a event from the calendar, provided the name of the event as it is stored. Example: \"event name\"")

	flag.BoolVar(&displayUpcomingEvents, "upcoming-events", false, "Display the upcoming events in the calendar.")

	// Parse the command-line arguments
	flag.Parse()

	// Create a new instance of the CalendarDB struct
	calendarDB := database.CalendarDB{}

	// Initialize the AWS session and DynamoDB client
	calendarDB.NewSession()

	// Check if new events were specified to be added
	switch newEvents {
	case "":
		fmt.Println("No events to add")
	default:
		calendarDB.AddEvents(newEvents)
	}

	// Check if an event name was specified to be deleted
	switch deleteEvent {
	case "":
		fmt.Println("No events to delete")
	default:
		calendarDB.DeleteEvent(deleteEvent)
	}

	// Check if the flag to display upcoming events is set
	if displayUpcomingEvents {
		calendarDB.GetEvents()
	}
}
