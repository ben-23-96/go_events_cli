package eventsearch

import (
	"encoding/json"
	"fmt"
)

type FoundEvent struct {
	Name     string
	Date     string
	City     string
	Tickets  string
	Genre    string
	Subgenre string
}

type UnmarshalFunction func([]byte) ([]FoundEvent, error)

func UnmarshalTicketmasterJSON(b []byte) ([]FoundEvent, error) {

	ticketmasterRes := TicketmasterResponse{}
	if err := json.Unmarshal(b, &ticketmasterRes); err != nil {
		fmt.Println("Error unmarshaling JSON:", err)
		return nil, err
	}

	var foundEvents []FoundEvent

	for _, event := range ticketmasterRes.Embedded.Events {
		foundEvents = append(foundEvents, FoundEvent{
			Name:     event.Name,
			Date:     event.Dates.Start.LocalDate,
			City:     event.Embedded.Venues[0].City.Name,
			Tickets:  event.URL,
			Genre:    event.Classifications[0].Segment.Name,
			Subgenre: event.Classifications[0].Genre.Name,
		})
	}

	return foundEvents, nil
}

func UnmarshalSkiddleJSON(b []byte) ([]FoundEvent, error) {

	skiddleRes := SkiddleResponse{}
	if err := json.Unmarshal(b, &skiddleRes); err != nil {
		fmt.Println("Error unmarshaling JSON:", err)
		return nil, err
	}

	var foundEvents []FoundEvent

	for _, event := range skiddleRes.Results {
		foundEvents = append(foundEvents, FoundEvent{
			Name:    event.EventName,
			Date:    event.Date,
			City:    event.Venue.Town,
			Tickets: event.Link,
			Genre:   event.EventCode,
			//Subgenre: event.Genres[0].Name,
		})
	}

	return foundEvents, nil
}

type TicketmasterResponse struct {
	Embedded struct {
		Events []TicketmasterEvent `json:"events"`
	} `json:"_embedded"`
}

type TicketmasterEvent struct {
	Name  string `json:"name"`
	URL   string `json:"url"`
	Dates struct {
		Start struct {
			LocalDate string `json:"localDate"`
		} `json:"start"`
	} `json:"dates"`
	Embedded struct {
		Venues []struct {
			City struct {
				Name string `json:"name"`
			} `json:"city"`
		} `json:"venues"`
	} `json:"_embedded"`
	Classifications []struct {
		Segment struct {
			Name string `json:"name"`
		} `json:"segment"`
		Genre struct {
			Name string `json:"name"`
		} `json:"genre"`
	} `json:"classifications"`
}

type SkiddleResponse struct {
	Results []struct {
		EventCode string `json:"EventCode"`
		EventName string `json:"eventname"`
		Venue     struct {
			Town string `json:"town"`
		} `json:"venue"`
		Link   string `json:"link"`
		Date   string `json:"date"`
		Genres []struct {
			Name string `json:"name"`
		} `json:"genres"`
	} `json:"results"`
}
