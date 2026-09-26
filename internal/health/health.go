package health

import (
	"encoding/json"
	"net/http"

	"github.com/ControlCenterSoft/aidi_2.0/internal/buildinfo"
)

type Response struct {
	Status  string         `json:"status"`
	Service string         `json:"service"`
	Build   buildinfo.Info `json:"build"`
}

func Live(build buildinfo.Info) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		write(w, http.StatusOK, Response{
			Status:  "ok",
			Service: "aidi-core",
			Build:   build,
		})
	}
}

func Ready(build buildinfo.Info, isReady func() bool) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		status := http.StatusOK
		state := "ready"
		if !isReady() {
			status = http.StatusServiceUnavailable
			state = "not_ready"
		}
		write(w, status, Response{
			Status:  state,
			Service: "aidi-core",
			Build:   build,
		})
	}
}

func write(w http.ResponseWriter, status int, response Response) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}
