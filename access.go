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

		json_data, err := json.Marshal(accessRequest)
		if err != nil {
			fmt.Println("Error:", err)
			http.Error(w, `500 Internal Server Error`, http.StatusInternalServerError)
			return
		}

		resp, err := nc.Request(config.NatsSubject, json_data, 1000*time.Millisecond)
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

		next.ServeHTTP(w, r)
	})
}
