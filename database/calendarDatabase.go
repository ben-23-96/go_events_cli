package database

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

// CalendarEvent represents an event to be stored in the calendar.
type CalendarEvent struct {
	EventName string
	Date      string
}

// CalendarDB is a data structure that holds a reference to a DynamoDB client.
type CalendarDB struct {
	db *dynamodb.DynamoDB
}

/*
initializes a new AWS session and creates a DynamoDB client.
*/
func (c *CalendarDB) NewSession() {
	// Create a new AWS session
	sess, err := session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	})
	// if error exit as cannot connect to dynamo db database
	if err != nil {
		log.Fatalf("Error creating session to connect to calendar db. Error: %s", err)
	}
	// Create DynamoDB client
	c.db = dynamodb.New(sess)
}

/*
AddEvents adds a new event to the Calendar table in DynamoDB.
Parameters:
- events: a string of comma seperated event names followed by the date they are on. eg event name, date, event 2, date 2
*/
func (c *CalendarDB) AddEvents(events string) {
	// split the string into list of names followed by dates
	commaSplit := strings.Split(events, ", ")
	// check an even number of items
	if len(commaSplit)%2 != 0 {
		fmt.Printf("Events not addded to calendar input string does not contain pairs of event name and date. %s\n", events)
		return
	}
	// Process each event name and date pair
	for i := 0; i < len(commaSplit); i += 2 {
		// Create a CalendarEvent struct with the provided event name and date
		event := CalendarEvent{
			EventName: strings.TrimSpace(commaSplit[i]),
			Date:      strings.TrimSpace(commaSplit[i+1]),
		}
		// Parse the event date
		date, err := time.Parse("2006-01-02", event.Date)
		if err != nil {
			fmt.Printf("Event %s not added to calendar invalid date format %s: %s\n", event.EventName, event.Date, date)
			continue
		}
		// Marshal the event into a DynamoDB attribute value
		av, err := dynamodbattribute.MarshalMap(event)
		if err != nil {
			fmt.Printf("Event %s not added to calendar. Got error marshalling event item: %s\n", event.EventName, err)
			continue
		}
		// Create a PutItemInput with the marshaled item and the table name
		input := &dynamodb.PutItemInput{
			Item:      av,
			TableName: aws.String("Calendar"),
		}
		// Put the item into the DynamoDB table
		_, err = c.db.PutItem(input)
		if err != nil {
			fmt.Printf("Event %s not added to calendar. Got error calling PutItem: %s\n", event.EventName, err)
			continue
		}

		fmt.Printf("successfully added %s on %s to calender\n", event.EventName, event.Date)
	}
}

/*
DeleteEvent deletes an event from the Calendar table in DynamoDB using the event name as the partiton key.
*/
func (c *CalendarDB) DeleteEvent(eventName string) {
	// Create a DeleteItemInput to specify the key (EventName) of the item to delete
	input := &dynamodb.DeleteItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"EventName": {
				S: aws.String(eventName),
			},
		},
		TableName: aws.String("Calendar"),
	}
	// Attempt to delete the item from the DynamoDB table
	_, err := c.db.DeleteItem(input)
	if err != nil {
		fmt.Printf("Got error calling DeleteItem: %s\n", err)
		return
	}

	fmt.Printf("Deleted %s from calendar\n", eventName)
}

/*
GetEvents retrieves and prints all events from the Calendar table in DynamoDB.
*/
func (c *CalendarDB) GetEvents() {
	// Perform a scan operation to retrieve all items from the Calendar table
	result, err := c.db.Scan(&dynamodb.ScanInput{
		TableName: aws.String("Calendar"),
	})

	if err != nil {
		fmt.Printf("Failed to fetch upcoming events from calendar: %s\n", err)
		return
	}
	fmt.Println("Upcoming Events:")
	// Iterate through the retrieved items and unmarshal them into CalendarEvent structs
	for _, i := range result.Items {
		event := CalendarEvent{}

		err = dynamodbattribute.UnmarshalMap(i, &event)

		if err != nil {
			fmt.Printf("Got error unmarshalling upcoming events: %s\n", err)
			return
		}
		// Print the name and date of each event
		fmt.Printf("%s    %s\n", event.Date, event.EventName)
	}
}
