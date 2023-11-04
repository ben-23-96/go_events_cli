package eventsearch

import (
	"encoding/json"
	"fmt"
)

type Response struct {
	Events []FoundEvent
}

type FoundEvent struct {
	Name     string
	Date     string
	City     string
	Tickets  string
	Genre    string
	Subgenre string
}

type UnmarshalFunction func([]byte) error

func (res *Response) UnmarshalTicketmasterJSON(b []byte) error {

	ticketmasterRes := TicketmasterResponse{}
	if err := json.Unmarshal(b, &ticketmasterRes); err != nil {
		fmt.Println("Error unmarshaling JSON:", err)
		return err
	}

	for _, event := range ticketmasterRes.Embedded.Events {
		res.Events = append(res.Events, FoundEvent{
			Name:     event.Name,
			Date:     event.Dates.Start.LocalDate,
			City:     event.Embedded.Venues[0].City.Name,
			Tickets:  event.URL,
			Genre:    event.Classifications[0].Segment.Name,
			Subgenre: event.Classifications[0].Genre.Name,
		})
	}

	return nil
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
