package goaccessmiddleware

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/nats-io/nats.go"
)

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

func CheckAccessMiddleware(
	next http.Handler,
	nc *nats.Conn,
	service_name string,
	nats_servers string,
	nats_subject string,
	exclude_paths []string,
	test bool,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if nc.Status() != nats.CONNECTED {
			nc, _ = nats.Connect(nats_servers)
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
		accessRequest.ServiceName = service_name
		fmt.Printf("%+v\n", r)
		json_data, err := json.Marshal(accessRequest)
		if err != nil {
			fmt.Println("Error: ", err)
		}

		resp, err := nc.Request(nats_subject, json_data, 1000*time.Millisecond)
		if err != nil {
			fmt.Println("Error: ", err)
		}
		fmt.Println("Received response:", string(resp.Data))
		var accessResponse AccessResponse
		err = json.Unmarshal(resp.Data, &accessResponse)
		if err != nil {
			fmt.Println("Error: ", err)
		}
		if !accessResponse.Access {
			http.Error(w, `404 page not found`, http.StatusNotFound)
			return
		}

		next.ServeHTTP(w, r)
	})
}
