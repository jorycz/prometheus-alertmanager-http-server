package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"net/http"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// Alert represents the structure of the "alerts" key in the JSON payload
type Alert struct {
	// Labels      map[string]string     `json:"labels"`
	Labels       Labels      `json:"labels"`
	Annotations  Annotations `json:"annotations"`
	Status       string      `json:"status"`
	GeneratorURL string      `json:"generatorURL"`
}
type Annotations struct {
	Description string `json:"description"`
	Summary     string `json:"summary"`
}
type Labels struct {
	AlertName string `json:"alertname"`
}

// Payload represents the structure of the JSON payload
type Payload struct {
	Receiver          string            `json:"receiver"`
	Status            string            `json:"status"`
	Alerts            []Alert           `json:"alerts"`
	CommonLabels      map[string]string `json:"commonLabels"`
	CommonAnnotations map[string]string `json:"commonAnnotations"`
	Version           string            `json:"version"`
	GroupKey          string            `json:"groupKey"`
	TruncatedAlerts   int               `json:"truncatedAlerts"`
}

// AppState is used for storing last status of "Alert Name" alert
type AppState struct {
	LastPayloads map[string]string
	Received     map[string]bool
	Mutex        sync.Mutex
}

var (
	appState                   AppState
	alertsProcessedWithSuccess bool
	httpResponseText           string
)

func handlePostRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		msg := "Only POST requests are allowed."
		fmt.Println(msg)
		http.Error(w, msg, http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		msg := fmt.Sprintf(" !!! Error reading request body: %s", err)
		fmt.Println(msg)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	// DEBUG: print request body of HTTP POST
	//fmt.Println(string(body))

	var payload Payload
	err = json.Unmarshal(body, &payload)
	if err != nil {
		msg := fmt.Sprintf(" !!! Error parsing JSON payload: %s", err)
		fmt.Println(msg)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	if len(payload.Alerts) <= 0 {
		msg := "No alerts detected in JSON payload."
		fmt.Println(msg)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	numOfAlerts := len(payload.Alerts)
	httpResponseText = fmt.Sprintf("All alerts [ %v ] processed successfuly. Debug files stored in /dev/shm/.\n", numOfAlerts)
	for _, alert := range payload.Alerts {
		alertsProcessedWithSuccess = processAlert(alert, w)
		if !alertsProcessedWithSuccess {
			httpResponseText = fmt.Sprintf("Some or all from [ %v ] alerts processed with errors. Debug files stored in /dev/shm/.\n", numOfAlerts)
			fmt.Println(httpResponseText)
		}
	}
	
	// Send response to client
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(httpResponseText))
}

func processAlert(alert Alert, w http.ResponseWriter) bool {
	// fmt.Printf("DEBUG: %s\n", alert.Annotations.Description)
	// fmt.Printf("DEBUG: %s\n", alert.Annotations.Summary)
	alertName := alert.Labels.AlertName
	if alertName == "" {
		msg := "Missing or invalid Alert Name in JSON payload"
		fmt.Println(msg)
		http.Error(w, msg, http.StatusBadRequest)
		return false
	}
	alertStatus := alert.Status
	if alertStatus == "" {
		msg := "Missing or invalid Alert Status in JSON payload"
		fmt.Println(msg)
		http.Error(w, msg, http.StatusBadRequest)
		return false
	}

	fmt.Printf("Alert [ %s ] with status [ %s ] arrived. ", alert.Labels.AlertName, alert.Status)

	// Check if the new payload is different than the last payload for the current Alert Name
	appState.Mutex.Lock()
	defer appState.Mutex.Unlock()

	if !appState.Received[alert.Labels.AlertName] || appState.LastPayloads[alert.Labels.AlertName] != alert.Status {
		// Save the content to a file in the /dev/shm/ folder with filename based on Alert Name
		debugFileParts := []string{"prometheus-alertmanager-http-server", alert.Labels.AlertName}
		debugFile := strings.Join(debugFileParts, "_")
		debugFile = strings.ReplaceAll(debugFile, " ", "_")
		filePath := filepath.Join("/dev/shm", debugFile)

		jsonData, error := json.MarshalIndent(alert, "", "    ")
		if error != nil {
			msg := fmt.Sprintf(" !!! Error writing to file: %s", error)
			fmt.Println(msg)
			http.Error(w, msg, http.StatusInternalServerError)
			return false
		}
		err := os.WriteFile(filePath, jsonData, 0644)
		if err != nil {
			msg := fmt.Sprintf(" !!! Error writing to file: %s", err)
			fmt.Println(msg)
			http.Error(w, msg, http.StatusInternalServerError)
			return false
		}

		// Check if the payload is not empty and this is not the first time for the current Alert Name
		if len(alert.Labels.AlertName) > 0 && appState.Received[alert.Labels.AlertName] {
			// Run the BASH script with Alert Name (- alert: "Plug Washing Machine") as the first parameter and Status (firing|resolved) as the second parameter
			cmd := exec.Command("./prometheus-alertmanager-http-server-notify.sh", alert.Labels.AlertName, alert.Status)
			cmd.Dir = "." // Set the working directory for the command. Absolute path is set in systemd service by WorkingDirectory= param
			err = cmd.Run()
			if err != nil {
				msg := fmt.Sprintf(" !!! Error running BASH script: %s", err)
				fmt.Println(msg)
				http.Error(w, msg, http.StatusInternalServerError)
				return false
			} else {
				fmt.Printf(" ---> Notification BASH script launched.\n")
			}
		} else {
			fmt.Printf("First JSON payload arrived after app fresh start - storing & ignoring.\n")
		}

		// Update the last payload for the current Alert Name in the application state
		appState.LastPayloads[alert.Labels.AlertName] = alert.Status
		appState.Received[alert.Labels.AlertName] = true
	} else {
		fmt.Printf("Status is the same as last time, ignoring.\n")
	}
	return true
}

func main() {
	// Initialize appState
	appState = AppState{
		LastPayloads: make(map[string]string),
		Received:     make(map[string]bool),
	}

	// Define the HTTP server and its handler
	http.HandleFunc("/", handlePostRequest)

	// Start the server on port 6666
	fmt.Println("------ Starting HTTP server on port 6666 ------")
	err := http.ListenAndServe(":6666", nil)
	if err != nil {
		fmt.Println(" !!! Error starting HTTP server:", err)
	}
}
