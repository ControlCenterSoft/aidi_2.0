package health

import (
	"fmt"
	"net/http"
)

func Handler(version string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, "{\"status\":\"ok\",\"service\":\"aidi-core\",\"version\":%q}\n", version)
	}
}
