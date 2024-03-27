package main

import (
	_ "embed"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/gocarina/gocsv"
)

type Date struct {
	time.Time
}

func (d *Date) MarshalCSV() (string, error) {
	return d.Format("01/02/2006"), nil
}

type Time struct {
	time.Time
}

func (t *Time) MarshalCSV() (string, error) {
	return t.Format("15:04"), nil
}

type CSVRow struct {
	EventTitle           string `csv:"event_title"`
	AdministrativeTitle  string `csv:"administrative_title"`
	LocationName         string `csv:"location_name"`
	Address              string `csv:"address"`
	City                 string `csv:"city"`
	State                string `csv:"state"`
	Zip                  string `csv:"zip"`
	Country              string `csv:"country"`
	Time                 Time   `csv:"time"`
	Date                 Date   `csv:"date"`
	Host                 string `csv:"host"`
	Sponsor              string `csv:"sponsor"`
	AttendeePitch        string `csv:"attendee_pitch"`
	AttendeeInstructions string `csv:"attendee_instructions"`
	HostContactInfo      string `csv:"host_contact_info"`
	DisableDiscussions   bool   `csv:"disable_discussions"`
	EndDate              Date   `csv:"end_date"`
	EndTime              Time   `csv:"end_time"`
}

//go:embed index.html
var indexHTML []byte

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		r.Header.Set("Content-Type", "text/html")
		w.Write(indexHTML)
	})
	r.Post("/create", func(w http.ResponseWriter, r *http.Request) {
		dateRangeStart, err := time.Parse("2006-01-02", r.PostFormValue("date_range_start"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		dateRangeEnd, err := time.Parse("2006-01-02", r.PostFormValue("date_range_end"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}

		selectedWeekdays := map[time.Weekday]bool{
			time.Sunday:    false,
			time.Monday:    false,
			time.Tuesday:   false,
			time.Wednesday: false,
			time.Thursday:  false,
			time.Friday:    false,
			time.Saturday:  false,
		}

		for day := range selectedWeekdays {
			selected := r.PostFormValue(day.String())
			if selected != "" {
				selectedWeekdays[day] = true
			}
		}

		var output []CSVRow

		fmt.Printf("start %v\n", dateRangeStart)
		fmt.Printf("end %v\n", dateRangeEnd)

		for d := dateRangeStart; !d.After(dateRangeEnd); d = d.AddDate(0, 0, 1) {
			if selected := selectedWeekdays[d.Weekday()]; !selected {
				continue
			}

			getValueForWeekday := func(key string) string {
				dayKey := key + "__" + d.Weekday().String()
				if val := r.PostFormValue(dayKey); val != "" {
					return val
				}
				return r.PostFormValue(key)
			}

			interpolationReplacer := strings.NewReplacer(
				"{date}", d.Format("January 2"),
				"{shortdate}", d.Format("1/2"),
				"{weekday}", d.Weekday().String(),
			)

			var row CSVRow
			row.EventTitle = interpolationReplacer.Replace(r.PostFormValue("event_title"))
			row.AdministrativeTitle = interpolationReplacer.Replace(r.PostFormValue("administrative_title"))
			row.LocationName = getValueForWeekday("location_name")
			row.Address = getValueForWeekday("address")
			row.City = getValueForWeekday("city")
			row.State = getValueForWeekday("state")
			row.Zip = getValueForWeekday("zip")
			row.Country = "US"
			eventTime, err := time.Parse("15:04", getValueForWeekday("start_time"))
			if err != nil {
				w.Write([]byte(d.Weekday().String()))
				err = fmt.Errorf("error parsing time: %v", err)
				http.Error(w, err.Error(), http.StatusBadRequest)
			}
			row.Time = Time{eventTime}
			row.Date = Date{d}
			row.Host = r.PostFormValue("host")
			row.Sponsor = r.PostFormValue("sponsor")
			row.AttendeePitch = interpolationReplacer.Replace(r.PostFormValue("attendee_pitch"))
			row.AttendeeInstructions = interpolationReplacer.Replace(r.PostFormValue("attendee_instructions"))
			row.HostContactInfo = r.PostFormValue("host_contact_info")
			row.DisableDiscussions = true
			row.EndDate = Date{d}
			endTime, err := time.Parse("15:04", getValueForWeekday("end_time"))
			if err != nil {
				err = fmt.Errorf("error parsing end time: %v", err)
				http.Error(w, err.Error(), http.StatusBadRequest)
			}
			row.EndTime = Time{endTime}
			output = append(output, row)
		}

		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", "attachment; filename=events.csv")
		gocsv.Marshal(&output, w)
	})
	http.ListenAndServe(":"+port, r)
}
