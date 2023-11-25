package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// CalendarEvent represents an event to be stored in the calendar.
type CalendarEvent struct {
	EventName string
	Date      string
}

// Initialize and establish a connection to the database
func InitDB() (*sql.DB, error) {
	dbFileName := "database/calendar.db"
	// Check if the database file exists.
	_, err := os.Stat(dbFileName)
	if os.IsNotExist(err) {
		// Database file does not exist, create it.
		file, err := os.Create(dbFileName)
		if err != nil {
			log.Fatal(err)
		}
		file.Close()
	}

	// Open the SQLite database file.
	db, err := sql.Open("sqlite", dbFileName)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	// Ping the database to check if the connection is valid.
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %v", err)
	}

	// query to create the table if does not exist
	query := `
		CREATE TABLE IF NOT EXISTS CalendarEvents (
			EventName TEXT,
			Date TEXT
		);
	`
	// execute the query
	_, err = db.Exec(query)
	if err != nil {
		fmt.Printf("failed to create CalendarEvents table: %v", err)
		return nil, err
	}

	fmt.Println("Connected to the database")
	return db, nil
}

/*
AddEvents adds a new event to the CalendarEvents table in the sqlite database.
Parameters:
- events: a string of comma seperated event names followed by the date they are on. eg event name, date, event 2, date 2
*/
func AddEvents(db *sql.DB, events string) {
	// split the string into list of names followed by dates
	commaSplit := strings.Split(events, ", ")
	// check an even number of items
	if len(commaSplit)%2 != 0 {
		fmt.Printf("Events not addded to calendar input string does not contain pairs of event name and date. %s\n", events)
		return
	}
	// Process each event name and date pair
	for i := 0; i < len(commaSplit); i += 2 {
		// get event name and event date
		eventName := strings.TrimSpace(commaSplit[i])
		eventDate := strings.TrimSpace(commaSplit[i+1])
		// Parse the event date
		date, err := time.Parse("2006-01-02", eventDate)
		if err != nil {
			fmt.Printf("Event %s not added to calendar invalid date format %s: %s\n", eventName, eventDate, date)
			continue
		}
		// query to insert event into table
		query := "INSERT INTO CalendarEvents (EventName, Date) VALUES (?, ?)"
		// execute the query
		_, err = db.Exec(query, eventName, eventDate)
		if err != nil {
			fmt.Printf("failed to add event to the database: %v", err)
			continue
		}

		fmt.Printf("successfully added %s on %s to calender\n", eventName, eventDate)
	}
}

/*
DeleteEvent deletes an event from the CalendarEvents table using the event name.
*/
func DeleteEvent(db *sql.DB, eventName string) {
	// query to delete a event from the table using the events name
	query := "DELETE FROM CalendarEvents WHERE EventName = ?"
	// execute the query
	_, err := db.Exec(query, eventName)
	if err != nil {
		fmt.Printf("failed to delete event from the database: %v", err)
		return
	}

	fmt.Println("Event deleted successfully")
}

/*
GetEvents retrieves and returns events on or after current date from the CalendarEvents table.
*/
func GetEvents(db *sql.DB) ([]CalendarEvent, error) {
	// query to return all events from the table
	query := "SELECT EventName, Date FROM CalendarEvents"
	// execute the query return the rows from table
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query events from the database: %v", err)
	}
	defer rows.Close()
	// for each returned row put data into a CalendarEvent struct and then append to the events slice
	var events []CalendarEvent
	for rows.Next() {
		var event CalendarEvent
		err := rows.Scan(&event.EventName, &event.Date)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event row: %v", err)
		}
		events = append(events, event)
	}

	// Sort the events by date
	sort.Slice(events, func(i, j int) bool {
		date1, _ := time.Parse(time.DateOnly, events[i].Date)
		date2, _ := time.Parse(time.DateOnly, events[j].Date)
		return date1.Before(date2)
	})
	//return the events slice
	return events, nil
}
