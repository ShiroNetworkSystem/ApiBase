package ApiBase

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type System struct {
	webHost  string
	webPort  int
	apiKey   string
	handlers map[string]interface{}
	err      chan error
}

type Handler func() error
type Errorhandler func(err error)
type M map[string]interface{}

func NewSystem(host string, port int, key string) System {
	return System{
		webHost:  host,
		webPort:  port,
		apiKey:   key,
		handlers: map[string]interface{}{},
		err:      make(chan error),
	}
}

func (s System) SetApiHandler(h Handler) {
	s.handlers["handler"] = h
}

func (s System) SetErrorHandler(h Errorhandler) {
	s.handlers["error"] = h
}

func (s System) StartServer(then func()) {
	host := fmt.Sprintf("%s:%d", s.webHost, s.webPort)

	go errorProcess(s.err, s.handlers["error"].(Errorhandler))
	go server(host, s.apiKey, s.handlers["handler"].(Handler), s.err)
	then()
}

func errorProcess(ch chan error, errorhandler Errorhandler) {
	e := errorhandler
	for err := range ch {
		if err != nil {
			e(err)
		}
	}
}

func server(host string, apiKey string, handler Handler, errCh chan<- error) {
	mux := http.NewServeMux()
	h := handler

	mux.HandleFunc("/api/reload", func(res http.ResponseWriter, req *http.Request) {
		if req.Method == "POST" {
			auth := req.Header.Get("Authorization")
			if auth != "" && strings.Contains(auth, "Bearer") {
				key := strings.Replace(auth, "Bearer ", "", 1)
				if key != "" && key == apiKey {
					err := h()
					if err != nil {
						res.WriteHeader(http.StatusInternalServerError)
						msg, err := createJsonMessage(M{
							"error": "aa",
						})
						if err != nil {
							sendError(errCh, err)
						}

						_, err = res.Write(msg)
						if err != nil {
							sendError(errCh, err)
						}
					} else {
						res.WriteHeader(http.StatusOK)
					}
				} else {
					res.WriteHeader(http.StatusUnauthorized)
				}
			}
		} else {
			res.WriteHeader(http.StatusMethodNotAllowed)
		}

	})

	httpServer := &http.Server{
		Addr:    host,
		Handler: mux,
	}

	err := httpServer.ListenAndServe()
	if err != nil {
		sendError(errCh, err)
	}
}

func sendError(ch chan<- error, err error) {
	select {
	case ch <- err:
	default:
		break
	}
}

func createJsonMessage(m M) ([]byte, error) {
	b, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}

	return b, nil
}
