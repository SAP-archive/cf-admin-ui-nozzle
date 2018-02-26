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
var whitelistedRepNames map[string]struct{}
var whitelistedGorouterNames map[string]struct{}

var (
	dopplerEndpoint   = kingpin.Flag("doppler-endpoint", "Doppler endpoint").Default("wss://doppler.cf.bosh-lite.com:443").OverrideDefaultFromEnvar("DOPPLER_ENDPOINT").String()
	skipSSLValidation = kingpin.Flag("skip-ssl-validation", "Please don't").Default("false").OverrideDefaultFromEnvar("SKIP_SSL_VALIDATION").Bool()
	debug             = kingpin.Flag("debug", "Debug logging").Default("false").OverrideDefaultFromEnvar("DEBUG").Bool()
	valueMetricFilter = kingpin.Flag("valuemetric-filter", "Blacklist or Whitelist").Default("whitelist").OverrideDefaultFromEnvar("VALUEMETRIC_FILTER").String()
)

func isLatency(name string) bool {
	return name == "latency" || name == "route_lookup_time" || strings.HasPrefix(name, "latency.")
}

func keepEvent(e *events.Envelope, useWhitelist bool) bool {

	if whitelistedGorouterNames ==nil{
		whitelistedGorouterNames = make(map[string]struct{})
		whitelistedGorouterNames["numCPUS"] = struct{}{}
		whitelistedGorouterNames["memoryStats.numBytesAllocated"] = struct{}{}
		whitelistedGorouterNames["uptime"] = struct{}{}
	}
	if whitelistedRepNames ==nil {
		whitelistedRepNames = make(map[string]struct{})
		whitelistedRepNames["numCPUS"] = struct{}{}
		whitelistedRepNames["memoryStats.numBytesAllocated"] = struct{}{}
		whitelistedRepNames["memoryStats.numBytesAllocatedHeap"] = struct{}{}
		whitelistedRepNames["memoryStats.numBytesAllocatedStack"] = struct{}{}
		whitelistedRepNames["CapacityTotalContainers"] = struct{}{}
		whitelistedRepNames["CapacityRemainingContainers"] = struct{}{}
		whitelistedRepNames["ContainerCount"] = struct{}{}
		whitelistedRepNames["CapacityTotalMemory"] = struct{}{}
		whitelistedRepNames["CapacityRemainingMemory"] = struct{}{}
		whitelistedRepNames["CapacityTotalDisk"] = struct{}{}
		whitelistedRepNames["CapacityRemainingDisk"] = struct{}{}
		whitelistedRepNames["logSenderTotalMessagesRead"] = struct{}{}
		whitelistedRepNames["numGoRoutines"] = struct{}{}
		whitelistedRepNames["memoryStats.numMallocs"] = struct{}{}
		whitelistedRepNames["memoryStats.numFrees"] = struct{}{}
	}

	eventType := e.GetEventType()

	switch eventType {
	case events.Envelope_ValueMetric:
		origin := e.GetOrigin()
		valueName := e.GetValueMetric().GetName()

		if useWhitelist {
			if origin == "rep" {
				if _, ok := whitelistedRepNames[valueName]; ok {
					return true
				}
			} else if origin == "gorouter" {
				if _, ok := whitelistedGorouterNames[valueName]; ok {
					return true
				}
			}
			return false

		} else  {

			if origin == "gorouter" && isLatency(valueName) {
				return false
			} else if origin == "grootfs" {
				return false
			} else {
				return true
			}
		}

	case events.Envelope_ContainerMetric:
		return true

	}
	return false
}

func eventProcessor(eventChan <-chan *events.Envelope, parsedEventChan chan *events.Envelope, stopProcessor chan int) {
	var useWhitelist bool
	if *valueMetricFilter == "whitelist" {
		useWhitelist =true
	}else if *valueMetricFilter == "blacklist"{
		useWhitelist =false
	} else {
		log.Fatal("Configuration error. VALUEMETRIC_FILTER supports only 'whitelist' or 'blacklist'")
	}

	for {
		select {
		case msg := <-eventChan:
			if keepEvent(msg,useWhitelist) {
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
