package api

import (
	"encoding/json"
	"net/http"
	"os"
)

type healthResponse struct {
	Env    string `json:"env"`
	Status string `json:"status"`
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	env := os.Getenv("CORTADO_ENV")
	if env == "" {
		env = "development"
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(healthResponse{
		Env:    env,
		Status: "ok",
	}); err != nil {
		http.Error(w, "failed to encode health response", http.StatusInternalServerError)
	}
}
