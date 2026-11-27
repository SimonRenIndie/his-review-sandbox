// Command app serves the appointment endpoints.
package main

import (
	"log"
	"net/http"

	"github.com/SimonRenIndie/his-review-sandbox/internal/db"
	"github.com/SimonRenIndie/his-review-sandbox/internal/scheduling"
)

func main() {
	d, err := db.Open("app.db")
	if err != nil {
		log.Fatal(err)
	}
	if err := db.Up(d); err != nil {
		log.Fatal(err)
	}
	s := &scheduling.Service{DB: d}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /appointments/cancel", func(w http.ResponseWriter, r *http.Request) {
		c := scheduling.Caller{UserID: r.Header.Get("X-User"), FacilityID: r.Header.Get("X-Facility")}
		if err := s.Cancel(r.Context(), c, r.FormValue("id"), r.FormValue("reason")); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	log.Fatal(http.ListenAndServe(":8080", mux))
}
