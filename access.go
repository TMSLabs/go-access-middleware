package access

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/nats-io/nats.go"
)

type Config struct {
	ServiceName  string
	NatsServers  string
	NatsSubject  string
	ExcludePaths []string
	Test         bool
}

type AccessResponse struct {
	Access bool `json:"access"`
	Id     int  `json:"id"`
}

type AccessRequest struct {
	Headers     map[string]string `json:"headers"`
	Method      string            `json:"method"`
	Path        string            `json:"path"`
	QueryString string            `json:"query_string"`
	RawPath     string            `json:"raw_path"`
	Scheme      string            `json:"scheme"`
	ServiceName string            `json:"service_name"`
	Type        string            `json:"type"`
	URL         string            `json:"url"`
	Version     string            `json:"version"`
}

type Result struct {
	Id          int    `json:"id"`
	Status      string `json:"status"`
	ResponeTime string `json:"response_time"`
}

var nc *nats.Conn

func CheckAccessMiddleware(
	next http.Handler,
	config Config,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		for _, path := range config.ExcludePaths {
			if r.URL.Path == path {
				next.ServeHTTP(w, r)
				return
			}
		}

		if nc == nil {
			nc_tmp, err := nats.Connect(config.NatsServers)
			if err != nil {
				http.Error(w, `500 Internal Server Error`, http.StatusInternalServerError)
				return
			}
			nc = nc_tmp
		}

		if nc.Status() != nats.CONNECTED {
			nc, _ = nats.Connect(config.NatsServers)
		}

		accessRequest := AccessRequest{}
		accessRequest.Headers = map[string]string{}
		for name, values := range r.Header {
			accessRequest.Headers[name] = values[0]
		}
		accessRequest.Method = r.Method
		accessRequest.Path = r.URL.Path
		accessRequest.QueryString = r.URL.RawQuery
		accessRequest.RawPath = r.URL.Path
		accessRequest.Scheme = "http"
		accessRequest.Type = "http"
		accessRequest.URL = accessRequest.Type + "://" + r.Host + r.RequestURI
		accessRequest.ServiceName = config.ServiceName
		accessRequest.Version = "v2"

		json_data, err := json.Marshal(accessRequest)
		if err != nil {
			fmt.Println("Error:", err)
			http.Error(w, `500 Internal Server Error`, http.StatusInternalServerError)
			return
		}

		resp, err := nc.Request(config.NatsSubject, json_data, 5*time.Second)
		if err != nil {
			fmt.Println("Error:", err)
			http.Error(w, `500 Internal Server Error`, http.StatusInternalServerError)
			return
		}
		fmt.Println("Received response:", string(resp.Data))
		var accessResponse AccessResponse
		err = json.Unmarshal(resp.Data, &accessResponse)
		if err != nil {
			fmt.Println("Error:", err)
			http.Error(w, `500 Internal Server Error`, http.StatusInternalServerError)
			return
		}
		if !accessResponse.Access {
			http.Error(w, `404 Page Not Found`, http.StatusNotFound)
			return
		}

		start := time.Now()

		next.ServeHTTP(w, r)

		end := time.Now()

		fmt.Printf("Request took %v\n", end.Sub(start))

		result := Result{
			Id:          accessResponse.Id,
			Status:      "OK",
			ResponeTime: end.Sub(start).String(),
		}
		result_json, _ := json.Marshal(result)
		nc.Publish(config.NatsSubject+".result", result_json)
	})
}
