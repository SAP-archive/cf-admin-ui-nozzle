package main

// Inspired by the noaa firehose sample script
// https://github.com/cloudfoundry/noaa/blob/master/firehose_sample/main.go

import (
	"crypto/tls"
	"fmt"
	"github.com/cloudfoundry/noaa/consumer"
	"github.com/cloudfoundry/sonde-go/events"
	"github.com/gogo/protobuf/proto"
	"github.com/gorilla/websocket"
	"gopkg.in/alecthomas/kingpin.v2"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const wss_timeout = 30 * time.Second

var eventKept int64
var eventDiscarded int64

var (
	dopplerEndpoint   = kingpin.Flag("doppler-endpoint", "Doppler endpoint").Default("wss://doppler.cf.bosh-lite.com:443").OverrideDefaultFromEnvar("DOPPLER_ENDPOINT").String()
	skipSSLValidation = kingpin.Flag("skip-ssl-validation", "Please don't").Default("false").OverrideDefaultFromEnvar("SKIP_SSL_VALIDATION").Bool()
	debug             = kingpin.Flag("debug", "Debug logging").Default("false").OverrideDefaultFromEnvar("DEBUG").Bool()
)

func isLatency(name string) bool {
	return name == "latency" || name == "route_lookup_time" || strings.HasPrefix(name, "latency.")
}

func keepEvent(e *events.Envelope) bool {

	eventType := e.GetEventType()

	switch eventType {
	case events.Envelope_ValueMetric:
		if e.GetOrigin() == "gorouter" && isLatency(e.GetValueMetric().GetName()) {
			return false
		} else {
			return true
		}

	case events.Envelope_ContainerMetric:
		return true

	default:
		return false
	}
}

func eventProcessor(eventChan <-chan *events.Envelope, parsedEventChan chan *events.Envelope, stopProcessor chan int) {

	for {
		select {
		case msg := <-eventChan:
			if keepEvent(msg) {
				eventKept++
				parsedEventChan <- msg
			} else {
				eventDiscarded++
			}

		case <-stopProcessor:
			return
		}
	}
}

func wsInit(w http.ResponseWriter, r *http.Request) {

	subID := strings.Split(r.URL.String(), "/firehose/")[1]
	authToken := r.Header.Get("Authorization")

	consumer := consumer.New(*dopplerEndpoint, &tls.Config{InsecureSkipVerify: *skipSSLValidation}, nil)
	consumer.SetIdleTimeout(wss_timeout)
	defer consumer.Close()
	eventChan, errorChan := consumer.Firehose(subID, authToken)

	quitErrorChecker := make(chan int)
	defer func() { quitErrorChecker <- 0 }()
	go func() {
		for {
			select {
			case err := <-errorChan:
				fmt.Fprintf(os.Stderr, "Firehose consumer error: %s", err.Error())
			case <-quitErrorChecker:
				return
			}
		}
	}()

	parsedEventChan := make(chan *events.Envelope, 4096)
	stopProcessor := make(chan int)
	defer func() { stopProcessor <- 0 }()
	go eventProcessor(eventChan, parsedEventChan, stopProcessor)

	conn, err := websocket.Upgrade(w, r, w.Header(), 1024, 1024)
	defer conn.Close()
	if err != nil {
		http.Error(w, "Could not open websocket connection", http.StatusBadRequest)
		return
	} else {
		for {
			event := <-parsedEventChan
			data, _ := proto.Marshal(event)

			if err := conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
				fmt.Fprintf(os.Stderr, "Write on websocket failed: %s. Closing connection.", err.Error())

				return
			}
		}
	}
}

func wsHealth(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "I'm Alive!\n")
}

func main() {
	kingpin.Parse()

	go func() {
		for range time.Tick(30000 * time.Millisecond) {
			if *debug {
				if eventKept != 0 && eventDiscarded != 0 {
					totalEvents := eventKept + eventDiscarded
					percentageKept := (eventKept * 100) / totalEvents

					fmt.Println("Sent " + strconv.FormatInt(percentageKept, 10) + "% of the " + strconv.FormatInt(totalEvents, 10) + " events received in the last 30s")
				}
			}
			eventDiscarded = 0
			eventKept = 0
		}
	}()

	http.HandleFunc("/", wsInit)
	http.HandleFunc("/health", wsHealth)
	log.Fatal(http.ListenAndServe(":"+os.Getenv("PORT"), nil))
}
