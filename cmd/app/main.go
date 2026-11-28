// Command app serves the appointment HTTP endpoints.
package main

import (
	"encoding/json"
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
	log.Fatal(http.ListenAndServe(":8080", newMux(&scheduling.Service{DB: d})))
}

func newMux(s *scheduling.Service) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /appointments/cancel", func(w http.ResponseWriter, r *http.Request) {
		c := scheduling.Caller{UserID: r.Header.Get("X-User"), FacilityID: r.Header.Get("X-Facility")}
		if err := s.Cancel(r.Context(), c, r.FormValue("id"), r.FormValue("reason")); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("POST /appointments/cancel-coded", func(w http.ResponseWriter, r *http.Request) {
		c := scheduling.Caller{UserID: r.Header.Get("X-User"), FacilityID: r.Header.Get("X-Facility")}
		if err := s.CancelWithCode(r.Context(), c, r.FormValue("id"), r.FormValue("code"), r.FormValue("note")); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("GET /appointments/cancellation-counts", func(w http.ResponseWriter, r *http.Request) {
		c := scheduling.Caller{UserID: r.Header.Get("X-User"), FacilityID: r.Header.Get("X-Facility")}
		counts, err := s.CancellationCounts(r.Context(), c)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(counts)
	})
	return mux
}
