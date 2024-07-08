package main

import (
	"log"
	"net/http"

	"github.com/tmslabs/go-access-middleware"
)

func healthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write([]byte(`{"status": "OK", "code": 200}`))
}

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", healthz)

	wrappedMux := access.CheckAccessMiddleware(
		mux,
		access.Config{
			ServiceName: "test_service",
			NatsServers: "nats://localhost:4222",
			NatsSubject: "access",
		},
	)

	log.Fatal(http.ListenAndServe(":8080", wrappedMux))
}
