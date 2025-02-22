package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Struktur der Log-Daten
type LogEntry struct {
	FPS               float64
	Players           int
	AI                int
	AvgRTT            float64
	AvgPktLoss        float64
	VehCount          int
	VehExtraCount     int
	ProjShells        int
	ProjMissiles      int
	ProjGrenades      int
	ProjTotal         int
	StreamingDynam    int
	StreamingStatic   int
	StreamingDisabled int
	StreamingNew      int
	StreamingDel      int
	StreamingBump     int
}

// Globale Zähler für Disconnects, Connects, Reservierungen und Timeouts
var (
	totalDisconnects     int
	disconnectErrors     int
	totalConnects        int
	connectionTimeouts   int
	totalReservations    int
	lastProcessedOffsets = make(map[string]int64) // Speicher für den letzten verarbeiteten Offset jeder Logdatei
)

// Funktion zum Abrufen des letzten Offsets einer spezifischen Logdatei
func getLastOffsetForFile(logFile string) (int64, error) {
	offset, exists := lastProcessedOffsets[logFile]
	if !exists {
		return 0, nil
	}
	return offset, nil
}

// Funktion zum Speichern des Offsets einer spezifischen Logdatei
func saveOffsetForFile(logFile string, offset int64) {
	lastProcessedOffsets[logFile] = offset
}

// Funktion zum Abrufen der neuesten Ordner basierend auf dem Zeitstempel im Ordnernamen
func getLastLogFolders(logsDir string, limit int) ([]string, error) {
	var folders []string

	// Durchsuche das Verzeichnis nach Unterordnern
	files, err := os.ReadDir(logsDir)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if file.IsDir() && strings.HasPrefix(file.Name(), "logs_") {
			folders = append(folders, file.Name())
		}
	}

	// Sortiere die Ordner nach Zeitstempel im Namen (neuest zuerst)
	if len(folders) > limit {
		folders = folders[len(folders)-limit:]
	}
	return folders, nil
}

// Funktion zum Abrufen der letzten Log-Daten aus den letzten 5 Ordnern
func getLastLogData() (LogEntry, error) {
	logsDir := "logs"
	lastLog := LogEntry{}

	// Hole die letzten 5 Ordner
	folders, err := getLastLogFolders(logsDir, 5)
	if err != nil {
		return LogEntry{}, err
	}

	// Durchsuche die Ordner nach der "console.log"-Datei
	for _, folder := range folders {
		logFile := filepath.Join(logsDir, folder, "console.log")

		// Prüfen, ob die Datei existiert
		if _, err := os.Stat(logFile); os.IsNotExist(err) {
			continue
		}

		// Prüfen, ob die Logdatei seit dem letzten Mal neue Zeilen enthält
		offset, err := getLastOffsetForFile(logFile)
		if err != nil {
			return LogEntry{}, err
		}

		file, err := os.Open(logFile)
		if err != nil {
			return LogEntry{}, err
		}
		defer file.Close()

		file.Seek(offset, 0) // Setze den Lese-Offset
		scanner := bufio.NewScanner(file)

		// Regex für FPS, Spieler, AI, RTT/Paketverlust, Veh, Projektil, Streaming und Reservierung
		fpsRegex := regexp.MustCompile(`FPS:\s([0-9.]+)`)
		playerRegex := regexp.MustCompile(`Player:\s([0-9]+)`)
		aiRegex := regexp.MustCompile(`AI:\s([0-9]+)`)
		rttPktRegex := regexp.MustCompile(`PktLoss:\s([0-9]+)/100,\sRtt:\s([0-9]+)`)
		vehRegex := regexp.MustCompile(`Veh:\s([0-9]+)\s\(([0-9]+)\)`)
		projRegex := regexp.MustCompile(`Proj\s\(S:\s([0-9]+),\sM:\s([0-9]+),\sG:\s([0-9]+)\s\|\s([0-9]+)\)`)
		streamingRegex := regexp.MustCompile(`Streaming\(Dynam:\s([0-9]+),\sStatic:\s([0-9]+),\sDisabled:\s([0-9]+)\s\|\sNew:\s([0-9]+),\sDel:\s([0-9]+),\sBump:\s([0-9]+)\)`)

		disconnectRegex := regexp.MustCompile(`disconnected.*reason=([0-9]+)`)
		timeoutRegex := regexp.MustCompile(`connection timeout.*identity=0x[0-9A-F]+`)
		connectRegex := regexp.MustCompile(`Player connected:`)
		reserveSlotRegex := regexp.MustCompile(`Reserving slot for player`)

		var rttValues []float64
		var pktLossValues []float64

		// Durchsuche nur die neuen Zeilen seit dem letzten Offset
		for scanner.Scan() {
			line := scanner.Text()

			// FPS extrahieren
			if fpsMatch := fpsRegex.FindStringSubmatch(line); len(fpsMatch) > 1 {
				lastLog.FPS, _ = strconv.ParseFloat(fpsMatch[1], 64)
			}

			// Spieleranzahl extrahieren
			if playerMatch := playerRegex.FindStringSubmatch(line); len(playerMatch) > 1 {
				lastLog.Players, _ = strconv.Atoi(playerMatch[1])
			}

			// AI-Anzahl extrahieren
			if aiMatch := aiRegex.FindStringSubmatch(line); len(aiMatch) > 1 {
				lastLog.AI, _ = strconv.Atoi(aiMatch[1])
			}

			// RTT und Paketverlust extrahieren
			if rttPktMatch := rttPktRegex.FindStringSubmatch(line); len(rttPktMatch) > 2 {
				pktLoss, _ := strconv.ParseFloat(rttPktMatch[1], 64)
				rtt, _ := strconv.ParseFloat(rttPktMatch[2], 64)

				pktLossValues = append(pktLossValues, pktLoss)
				rttValues = append(rttValues, rtt)
			}

			// Disconnects erfassen
			if disconnectMatch := disconnectRegex.FindStringSubmatch(line); len(disconnectMatch) > 1 {
				totalDisconnects++
				reason, _ := strconv.Atoi(disconnectMatch[1])
				if reason == 6 {
					disconnectErrors++
				}
			}

			// "connection timeout"-Fehler erfassen
			if timeoutRegex.MatchString(line) {
				connectionTimeouts++
			}

			// Connects erfassen
			if connectRegex.MatchString(line) {
				totalConnects++
			}

			// Reservierungen erfassen
			if reserveSlotRegex.MatchString(line) {
				totalReservations++
			}

			// Fahrzeuge extrahieren
			if vehMatch := vehRegex.FindStringSubmatch(line); len(vehMatch) > 2 {
				lastLog.VehCount, _ = strconv.Atoi(vehMatch[1])
				lastLog.VehExtraCount, _ = strconv.Atoi(vehMatch[2])
			}

			// Projektil-Daten extrahieren
			if projMatch := projRegex.FindStringSubmatch(line); len(projMatch) > 4 {
				lastLog.ProjShells, _ = strconv.Atoi(projMatch[1])
				lastLog.ProjMissiles, _ = strconv.Atoi(projMatch[2])
				lastLog.ProjGrenades, _ = strconv.Atoi(projMatch[3])
				lastLog.ProjTotal, _ = strconv.Atoi(projMatch[4])
			}

			// Streaming-Daten extrahieren
			if streamingMatch := streamingRegex.FindStringSubmatch(line); len(streamingMatch) > 6 {
				lastLog.StreamingDynam, _ = strconv.Atoi(streamingMatch[1])
				lastLog.StreamingStatic, _ = strconv.Atoi(streamingMatch[2])
				lastLog.StreamingDisabled, _ = strconv.Atoi(streamingMatch[3])
				lastLog.StreamingNew, _ = strconv.Atoi(streamingMatch[4])
				lastLog.StreamingDel, _ = strconv.Atoi(streamingMatch[5])
				lastLog.StreamingBump, _ = strconv.Atoi(streamingMatch[6])
			}
		}

		if err := scanner.Err(); err != nil {
			return LogEntry{}, err
		}

		// Berechne den Durchschnitt der RTT-Werte und Paketverluste
		if len(rttValues) > 0 {
			totalRTT := 0.0
			totalPktLoss := 0.0

			for i := range rttValues {
				totalRTT += rttValues[i]
				totalPktLoss += pktLossValues[i]
			}

			lastLog.AvgRTT = totalRTT / float64(len(rttValues))
			lastLog.AvgPktLoss = totalPktLoss / float64(len(pktLossValues))
		}

		// Speichere den neuen Offset
		currentOffset, _ := file.Seek(0, 1)
		saveOffsetForFile(logFile, currentOffset)
	}

	return lastLog, nil
}

// Prometheus-kompatiblen Metriken-Endpunkt bereitstellen
func prometheusMetricsHandler(w http.ResponseWriter, r *http.Request) {
	lastLog, err := getLastLogData()
	if err != nil {
		http.Error(w, "Fehler beim Lesen der Logs", http.StatusInternalServerError)
		return
	}

	// Prometheus-kompatible Metriken ausgeben
	fmt.Fprintf(w, "# HELP server_fps Die FPS des Servers\n")
	fmt.Fprintf(w, "# TYPE server_fps gauge\n")
	fmt.Fprintf(w, "server_fps %f\n", lastLog.FPS)

	fmt.Fprintf(w, "# HELP server_players Anzahl der Spieler\n")
	fmt.Fprintf(w, "# TYPE server_players gauge\n")
	fmt.Fprintf(w, "server_players %d\n", lastLog.Players)

	fmt.Fprintf(w, "# HELP server_ai Anzahl der AI\n")
	fmt.Fprintf(w, "# TYPE server_ai gauge\n")
	fmt.Fprintf(w, "server_ai %d\n", lastLog.AI)

	fmt.Fprintf(w, "# HELP server_avg_rtt Durchschnittliche RTT in Millisekunden\n")
	fmt.Fprintf(w, "# TYPE server_avg_rtt gauge\n")
	fmt.Fprintf(w, "server_avg_rtt %f\n", lastLog.AvgRTT)

	fmt.Fprintf(w, "# HELP server_avg_pktloss Durchschnittlicher Paketverlust (in %% pro 100)\n")
	fmt.Fprintf(w, "# TYPE server_avg_pktloss gauge\n")
	fmt.Fprintf(w, "server_avg_pktloss %f\n", lastLog.AvgPktLoss)

	fmt.Fprintf(w, "# HELP server_veh_count Anzahl der Fahrzeuge\n")
	fmt.Fprintf(w, "# TYPE server_veh_count gauge\n")
	fmt.Fprintf(w, "server_veh_count %d\n", lastLog.VehCount)

	fmt.Fprintf(w, "# HELP server_veh_extra_count Anzahl der zusätzlichen Fahrzeuge\n")
	fmt.Fprintf(w, "# TYPE server_veh_extra_count gauge\n")
	fmt.Fprintf(w, "server_veh_extra_count %d\n", lastLog.VehExtraCount)

	// Proj-Metriken
	fmt.Fprintf(w, "# HELP server_proj_shells Anzahl der aktiven Granaten\n")
	fmt.Fprintf(w, "# TYPE server_proj_shells gauge\n")
	fmt.Fprintf(w, "server_proj_shells %d\n", lastLog.ProjShells)

	fmt.Fprintf(w, "# HELP server_proj_missiles Anzahl der aktiven Raketen\n")
	fmt.Fprintf(w, "# TYPE server_proj_missiles gauge\n")
	fmt.Fprintf(w, "server_proj_missiles %d\n", lastLog.ProjMissiles)

	fmt.Fprintf(w, "# HELP server_proj_grenades Anzahl der aktiven Granaten\n")
	fmt.Fprintf(w, "# TYPE server_proj_grenades gauge\n")
	fmt.Fprintf(w, "server_proj_grenades %d\n", lastLog.ProjGrenades)

	fmt.Fprintf(w, "# HELP server_proj_total Gesamtanzahl der Projektile\n")
	fmt.Fprintf(w, "# TYPE server_proj_total gauge\n")
	fmt.Fprintf(w, "server_proj_total %d\n", lastLog.ProjTotal)

	// Streaming-Metriken
	fmt.Fprintf(w, "# HELP server_streaming_dynam Anzahl der dynamischen Streaming-Objekte\n")
	fmt.Fprintf(w, "# TYPE server_streaming_dynam gauge\n")
	fmt.Fprintf(w, "server_streaming_dynam %d\n", lastLog.StreamingDynam)

	fmt.Fprintf(w, "# HELP server_streaming_static Anzahl der statischen Streaming-Objekte\n")
	fmt.Fprintf(w, "# TYPE server_streaming_static gauge\n")
	fmt.Fprintf(w, "server_streaming_static %d\n", lastLog.StreamingStatic)

	fmt.Fprintf(w, "# HELP server_streaming_disabled Anzahl der deaktivierten Streaming-Objekte\n")
	fmt.Fprintf(w, "# TYPE server_streaming_disabled gauge\n")
	fmt.Fprintf(w, "server_streaming_disabled %d\n", lastLog.StreamingDisabled)

	fmt.Fprintf(w, "# HELP server_streaming_new Anzahl der neuen Streaming-Objekte\n")
	fmt.Fprintf(w, "# TYPE server_streaming_new gauge\n")
	fmt.Fprintf(w, "server_streaming_new %d\n", lastLog.StreamingNew)

	fmt.Fprintf(w, "# HELP server_streaming_del Anzahl der gelöschten Streaming-Objekte\n")
	fmt.Fprintf(w, "# TYPE server_streaming_del gauge\n")
	fmt.Fprintf(w, "server_streaming_del %d\n", lastLog.StreamingDel)

	fmt.Fprintf(w, "# HELP server_streaming_bump Anzahl der Streaming-Bumps\n")
	fmt.Fprintf(w, "# TYPE server_streaming_bump gauge\n")
	fmt.Fprintf(w, "server_streaming_bump %d\n", lastLog.StreamingBump)

	// Disconnect, Connect und Reservierungen
	fmt.Fprintf(w, "# HELP server_disconnects Anzahl der Disconnect-Ereignisse\n")
	fmt.Fprintf(w, "# TYPE server_disconnects counter\n")
	fmt.Fprintf(w, "server_disconnects %d\n", totalDisconnects)

	fmt.Fprintf(w, "# HELP server_disconnect_errors Anzahl der fehlerhaften Disconnects (reason=6)\n")
	fmt.Fprintf(w, "# TYPE server_disconnect_errors counter\n")
	fmt.Fprintf(w, "server_disconnect_errors %d\n", disconnectErrors)

	fmt.Fprintf(w, "# HELP server_connection_timeouts Anzahl der 'connection timeout'-Fehler\n")
	fmt.Fprintf(w, "# TYPE server_connection_timeouts counter\n")
	fmt.Fprintf(w, "server_connection_timeouts %d\n", connectionTimeouts)

	fmt.Fprintf(w, "# HELP server_connects Anzahl der Connect-Ereignisse\n")
	fmt.Fprintf(w, "# TYPE server_connects counter\n")
	fmt.Fprintf(w, "server_connects %d\n", totalConnects)

	fmt.Fprintf(w, "# HELP server_reservations Anzahl der Reservierungsslots\n")
	fmt.Fprintf(w, "# TYPE server_reservations counter\n")
	fmt.Fprintf(w, "server_reservations %d\n", totalReservations)
}

func main() {
	// Prometheus-Endpunkt unter /metrics verfügbar machen
	http.HandleFunc("/metrics", prometheusMetricsHandler)

	log.Println("Server läuft auf Port 8880")
	log.Fatal(http.ListenAndServe(":8880", nil))
}
