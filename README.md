# Calendar and Event Search CLI

This command-line tool allows you to manage calendar events and search for events using the Ticketmaster and Skiddle APIs while ensuring they don't clash with existing calendar events.

## Usage
### Calendar
The calendar command allows you to add and display events from your calendar. It also provides an option to delete events. Here are the available options:
- **Add events to calendar:**
```
calendar -add-events "event name, date, event name 2, date 2"
```

- **Delete Event from Calendar:**
```
calendar -delete-event "event name"
```

- **Display Upcoming Events:**
```
calendar -upcoming-events
```

### Event Search

The `search` command lets you search for events from Ticketmaster and Skiddle APIs, ensuring they don't clash with your calendar. Here are the available options:

- **Cities:**

Specify individual cities or a comma-separated list of cities. For example:

```
search -cities "Manchester, Bristol"
```

- **Date Range:**

Date to start searching from (in format YYYY-MM-DD). Default is the current date.

```
search -date-from "2023-11-05"
```

Date to start searching to (in format YYYY-MM-DD). Default is 1 month from the current date.

```
search -date-to "2023-12-05"
```

- **Genres:**

Specify individual genres or subgenres as a comma-separated list. For example:

```
search -genres "Techno, Football"
```

## Example

Search for music events in Manchester from November 5, 2023, to December 5, 2023:
```
search -cities "Manchester" -genres "Music" -date-from "2023-11-05" -date-to "2023-12-05"
```
