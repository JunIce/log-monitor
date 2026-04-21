package main

import (
	"bufio"
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

//go:embed public
var publicFS embed.FS

type TimeRangeJSON struct {
	GeneratedAt string     `json:"generated_at"`
	Logs        []LogEntry `json:"logs"`
}

type MatchedLog struct {
	Date      string `json:"date"`
	Filename  string `json:"filename"`
	LogType   string `json:"log_type"`
	TimeRange string `json:"time_range"`
	FilePath  string `json:"file_path"`
}

type LogEntry struct {
	Date      string `json:"date"`
	Filename  string `json:"filename"`
	LogType   string `json:"log_type"`
	FirstTime string `json:"first_time"`
	LastTime  string `json:"last_time"`
}

var (
	logIndex TimeRangeJSON
	config   Config
)

type Config struct {
	Port      string
	LogDir    string
	IndexFile string
}

type QueryRequest struct {
	LogType   string `json:"log_type"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
	Date      string `json:"date"`
	Keyword   string `json:"keyword"`
	Limit     int    `json:"limit"`
	LogDir    string `json:"log_dir"`
}

type QueryResponse struct {
	MatchedFiles []MatchedLog `json:"matched_files"`
	Total        int          `json:"total"`
}

type LogLine struct {
	LineNum int    `json:"line_num"`
	Content string `json:"content"`
}

type LogContentResponse struct {
	Filename  string    `json:"filename"`
	Lines     []LogLine `json:"lines"`
	Total     int       `json:"total"`
	TotalLine int       `json:"total_line"`
}

type Project struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	LogDir    string `json:"log_dir"`
	IndexFile string `json:"index_file"`
}

type ProjectsConfig struct {
	Projects []Project `json:"projects"`
}

var projectsConfig ProjectsConfig
var configFile string

func parseTime(timeStr string) time.Time {
	layouts := []string{
		"2006-01-02 15:04:05.000",
		"15:04:05.000",
	}
	for _, layout := range layouts {
		if strings.Contains(timeStr, "-") {
			layout = "2006-01-02 15:04:05.000"
		} else {
			layout = "15:04:05.000"
		}
		t, err := time.Parse(layout, timeStr)
		if err == nil {
			return t
		}
	}
	return time.Time{}
}

func timeToSeconds(t time.Time) int {
	return t.Hour()*3600 + t.Minute()*60 + t.Second()
}

func compareTimeWithDefaultDate(timeStr, defaultDate string) (int, error) {
	var baseDate time.Time
	if strings.Contains(timeStr, "-") {
		baseDate, _ = time.Parse("2006-01-02", defaultDate)
	} else {
		baseDate, _ = time.Parse("2006-01-02", defaultDate)
	}

	var layout string
	if len(timeStr) == 12 {
		layout = "15:04:05.000"
	} else {
		layout = "2006-01-02 15:04:05.000"
	}

	t, err := time.Parse(layout, timeStr)
	if err != nil {
		return 0, err
	}

	if t.Year() == 0 {
		t = time.Date(baseDate.Year(), baseDate.Month(), baseDate.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.UTC)
	}

	return t.Hour()*3600 + t.Minute()*60 + t.Second(), nil
}

func loadIndex() error {
	return loadIndexWithPath(config.IndexFile)
}

func loadIndexWithPath(indexFile string) error {
	data, err := os.ReadFile(indexFile)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &logIndex)
}

func queryLogs(req QueryRequest) []MatchedLog {
	var results []MatchedLog

	for _, entry := range logIndex.Logs {
		if req.LogType != "" && entry.LogType != req.LogType {
			continue
		}
		if req.Date != "" && entry.Date != req.Date {
			continue
		}

		entryFirst, err1 := compareTimeWithDefaultDate(entry.FirstTime, entry.Date)
		entryLast, err2 := compareTimeWithDefaultDate(entry.LastTime, entry.Date)

		if err1 != nil || err2 != nil {
			continue
		}

		queryStart := 0
		queryEnd := 86400
		hasStart := false
		hasEnd := false

		if req.StartTime != "" {
			if s, err := compareTimeWithDefaultDate(req.StartTime, entry.Date); err == nil {
				queryStart = s
				hasStart = true
			}
		}
		if req.EndTime != "" {
			if e, err := compareTimeWithDefaultDate(req.EndTime, entry.Date); err == nil {
				queryEnd = e
				hasEnd = true
			}
		}

		matched := false
		if hasStart && hasEnd {
			matched = (entryFirst <= queryEnd && entryLast >= queryStart)
		} else if hasStart {
			matched = (entryLast > queryStart)
		} else if hasEnd {
			matched = (entryFirst < queryEnd)
		} else {
			matched = true
		}

		if matched {
			logDir := req.LogDir
			if logDir == "" {
				logDir = config.LogDir
			}
			filePath := filepath.Join(logDir, entry.Date, entry.Filename)
			results = append(results, MatchedLog{
				Date:      entry.Date,
				Filename:  entry.Filename,
				LogType:   entry.LogType,
				TimeRange: fmt.Sprintf("%s ~ %s", entry.FirstTime, entry.LastTime),
				FilePath:  filePath,
			})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		a := results[i].Date + results[i].TimeRange
		b := results[j].Date + results[j].TimeRange
		return a < b
	})

	return results
}

func queryLogsHandler(w http.ResponseWriter, r *http.Request) {
	logDir := r.URL.Query().Get("log_dir")
	indexFile := r.URL.Query().Get("index_file")

	if logDir == "" {
		logDir = config.LogDir
	}
	if indexFile == "" {
		indexFile = config.IndexFile
	} else {
		indexFile = filepath.Join(logDir, indexFile)
	}

	if err := loadIndexWithPath(indexFile); err != nil {
		http.Error(w, "Failed to load index: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var req QueryRequest
	if r.Method == http.MethodGet {
		req.LogType = r.URL.Query().Get("log_type")
		req.StartTime = r.URL.Query().Get("start_time")
		req.EndTime = r.URL.Query().Get("end_time")
		req.Date = r.URL.Query().Get("date")
		req.Keyword = r.URL.Query().Get("keyword")
		req.LogDir = logDir
		if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
			fmt.Sscanf(limitStr, "%d", &req.Limit)
		}
	} else {
		json.NewDecoder(r.Body).Decode(&req)
	}

	if req.Limit == 0 {
		req.Limit = 100
	}

	results := queryLogs(req)

	var filtered []MatchedLog
	if req.Keyword != "" {
		for _, m := range results {
			if strings.Contains(m.Filename, req.Keyword) {
				filtered = append(filtered, m)
			}
		}
	} else {
		filtered = results
	}

	if len(filtered) > req.Limit {
		filtered = filtered[:req.Limit]
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(QueryResponse{
		MatchedFiles: filtered,
		Total:        len(results),
	})
}

func logContentHandler(w http.ResponseWriter, r *http.Request) {
	filename := r.URL.Query().Get("filename")
	date := r.URL.Query().Get("date")
	logDir := r.URL.Query().Get("log_dir")
	startLine := 0
	endLine := 100
	startTime := r.URL.Query().Get("start_time")
	endTime := r.URL.Query().Get("end_time")
	keyword := r.URL.Query().Get("keyword")
	fmt.Sscanf(r.URL.Query().Get("start_line"), "%d", &startLine)
	fmt.Sscanf(r.URL.Query().Get("end_line"), "%d", &endLine)
	if endLine == 0 {
		endLine = 100
	}

	if filename == "" || date == "" {
		http.Error(w, "filename and date are required", http.StatusBadRequest)
		return
	}

	if logDir == "" {
		logDir = config.LogDir
	}

	filePath := filepath.Join(logDir, date, filename)
	file, err := os.Open(filePath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	defer file.Close()

	var allLines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		allLines = append(allLines, scanner.Text())
	}

	queryStartSec := 0
	queryEndSec := 86400
	if startTime != "" {
		if s, err := compareTimeWithDefaultDate(startTime, date); err == nil {
			queryStartSec = s
		}
	}
	if endTime != "" {
		if e, err := compareTimeWithDefaultDate(endTime, date); err == nil {
			queryEndSec = e
		}
	}

	var lines []string
	var totalMatched int
	if keyword != "" {
		keywordLower := strings.ToLower(keyword)
		for _, line := range allLines {
			if strings.Contains(strings.ToLower(line), keywordLower) {
				totalMatched++
				if len(lines) < 500 {
					lines = append(lines, line)
				}
			}
		}
	} else {
		for _, line := range allLines {
			if len(line) < 12 {
				continue
			}
			lineTime := line[:12]
			if lineSec, err := compareTimeWithDefaultDate(lineTime, date); err == nil {
				if lineSec >= queryStartSec && lineSec <= queryEndSec {
					totalMatched++
					if len(lines) < 5000 {
						lines = append(lines, line)
					}
				}
			}
		}
	}

	if len(lines) == 0 {
		lines = allLines
		totalMatched = len(allLines)
	}

	start := startLine
	if start < 1 {
		start = 1
	}
	if start > len(lines) {
		start = len(lines)
	}

	end := endLine
	if end > len(lines) {
		end = len(lines)
	}

	var logLines []LogLine
	for i := start - 1; i < end; i++ {
		logLines = append(logLines, LogLine{
			LineNum: i + 1,
			Content: lines[i],
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(LogContentResponse{
		Filename:  filename,
		Lines:     logLines,
		Total:     totalMatched,
		TotalLine: len(allLines),
	})
}

func listLogTypesHandler(w http.ResponseWriter, r *http.Request) {
	logDir := r.URL.Query().Get("log_dir")
	indexFile := r.URL.Query().Get("index_file")

	if logDir == "" {
		logDir = config.LogDir
	}
	if indexFile == "" {
		indexFile = config.IndexFile
	} else {
		indexFile = filepath.Join(logDir, indexFile)
	}

	if err := loadIndexWithPath(indexFile); err != nil {
		http.Error(w, "Failed to load index: "+err.Error(), http.StatusInternalServerError)
		return
	}

	types := make(map[string]bool)
	dates := make(map[string]bool)

	for _, entry := range logIndex.Logs {
		types[entry.LogType] = true
		dates[entry.Date] = true
	}

	typeList := make([]string, 0, len(types))
	for t := range types {
		typeList = append(typeList, t)
	}
	sort.Strings(typeList)

	dateList := make([]string, 0, len(dates))
	for d := range dates {
		dateList = append(dateList, d)
	}
	sort.Strings(dateList)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"log_types": typeList,
		"dates":     dateList,
	})
}

func loadProjects() error {
	data, err := os.ReadFile(configFile)
	if err != nil {
		if os.IsNotExist(err) {
			projectsConfig = ProjectsConfig{Projects: []Project{}}
			return nil
		}
		return err
	}
	return json.Unmarshal(data, &projectsConfig)
}

func saveProjects() error {
	data, err := json.MarshalIndent(projectsConfig, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configFile, data, 0644)
}

func listProjectsHandler(w http.ResponseWriter, r *http.Request) {
	if err := loadProjects(); err != nil {
		http.Error(w, "Failed to load projects: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(projectsConfig)
}

func addProjectHandler(w http.ResponseWriter, r *http.Request) {
	if err := loadProjects(); err != nil {
		http.Error(w, "Failed to load projects: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var newProject Project
	if err := json.NewDecoder(r.Body).Decode(&newProject); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	newProject.ID = fmt.Sprintf("%d", time.Now().UnixNano())
	projectsConfig.Projects = append(projectsConfig.Projects, newProject)

	if err := saveProjects(); err != nil {
		http.Error(w, "Failed to save projects: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(newProject)
}

func updateProjectHandler(w http.ResponseWriter, r *http.Request) {
	if err := loadProjects(); err != nil {
		http.Error(w, "Failed to load projects: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var updateProject Project
	if err := json.NewDecoder(r.Body).Decode(&updateProject); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	found := false
	for i, p := range projectsConfig.Projects {
		if p.ID == updateProject.ID {
			projectsConfig.Projects[i] = updateProject
			found = true
			break
		}
	}

	if !found {
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	if err := saveProjects(); err != nil {
		http.Error(w, "Failed to save projects: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updateProject)
}

func deleteProjectHandler(w http.ResponseWriter, r *http.Request) {
	if err := loadProjects(); err != nil {
		http.Error(w, "Failed to load projects: "+err.Error(), http.StatusInternalServerError)
		return
	}

	projectID := r.URL.Query().Get("id")
	if projectID == "" {
		http.Error(w, "Project ID is required", http.StatusBadRequest)
		return
	}

	found := false
	var newProjects []Project
	for _, p := range projectsConfig.Projects {
		if p.ID != projectID {
			newProjects = append(newProjects, p)
		} else {
			found = true
		}
	}

	if !found {
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	projectsConfig.Projects = newProjects

	if err := saveProjects(); err != nil {
		http.Error(w, "Failed to save projects: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

func generateIndex(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ID        string `json:"id"`
		LogDir    string `json:"log_dir"`
		IndexFile string `json:"index_file"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.LogDir == "" {
		http.Error(w, "log_dir is required", http.StatusBadRequest)
		return
	}

	// Handle Windows backslash paths - replace \ with / for consistent handling
	req.LogDir = strings.ReplaceAll(req.LogDir, "\\", "/")
	req.IndexFile = strings.ReplaceAll(req.IndexFile, "\\", "/")

	indexFile := req.IndexFile
	if indexFile == "" {
		indexFile = "time_ranges.json"
	}

	if !filepath.IsAbs(indexFile) {
		indexFile = filepath.Join(req.LogDir, indexFile)
	}

	// Execute the Go index generation function
	err := generateTimeRangeIndex(req.LogDir, indexFile)
	if err != nil {
		http.Error(w, "Failed to generate index: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": fmt.Sprintf("Index generated successfully for %s", req.LogDir),
	})
}

func generateTimeRangeIndex(logDir, outputFile string) error {
	timePattern := regexp.MustCompile(`^[0-9]{2}:[0-9]{2}:[0-9]{2}\.[0-9]{3}`)
	datePattern := regexp.MustCompile(`\.[0-9]{4}-[0-9]{2}-[0-9]{2}\..*\.log$`)

	type indexedFile struct {
		Date      string    `json:"date"`
		Filename  string    `json:"filename"`
		LogType   string    `json:"log_type"`
		FirstTime string    `json:"first_time"`
		LastTime  string    `json:"last_time"`
		ModTime   time.Time `json:"mod_time"`
	}

	type indexData struct {
		GeneratedAt string        `json:"generated_at"`
		Logs        []indexedFile `json:"logs"`
	}

	extractTimeRange := func(logFile string) (firstTime, lastTime string, err error) {
		file, err := os.Open(logFile)
		if err != nil {
			return "", "", err
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		var times []string

		for scanner.Scan() {
			line := scanner.Text()
			if timeMatch := timePattern.FindString(line); timeMatch != "" {
				times = append(times, timeMatch)
			}
		}

		if len(times) == 0 {
			return "", "", fmt.Errorf("no timestamps found in %s", logFile)
		}

		return times[0], times[len(times)-1], nil
	}

	extractLogType := func(filename string) string {
		return datePattern.ReplaceAllString(filename, "")
	}

	var existingIndex indexData
	if data, err := os.ReadFile(outputFile); err == nil {
		json.Unmarshal(data, &existingIndex)
	}

	existingFiles := make(map[string]indexedFile)
	for _, entry := range existingIndex.Logs {
		key := entry.Date + "/" + entry.Filename
		existingFiles[key] = entry
	}

	var logEntries []indexedFile
	var updatedCount, skippedCount int

	err := filepath.WalkDir(logDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		if !strings.HasSuffix(path, ".log") {
			return nil
		}

		filename := filepath.Base(path)
		dirName := filepath.Base(filepath.Dir(path))
		key := dirName + "/" + filename

		info, err := d.Info()
		if err != nil {
			return nil
		}
		modTime := info.ModTime()

		if existing, ok := existingFiles[key]; ok {
			if existing.ModTime.Equal(modTime) || existing.ModTime.After(modTime) {
				logEntries = append(logEntries, existing)
				skippedCount++
				return nil
			}
		}

		firstTime, lastTime, err := extractTimeRange(path)
		if err != nil {
			fmt.Printf("Warning: %v\n", err)
			return nil
		}

		logType := extractLogType(filename)
		entry := indexedFile{
			Date:      dirName,
			Filename:  filename,
			LogType:   logType,
			FirstTime: firstTime,
			LastTime:  lastTime,
			ModTime:   modTime,
		}

		logEntries = append(logEntries, entry)
		updatedCount++
		return nil
	})

	if err != nil {
		return fmt.Errorf("error walking log directory: %v", err)
	}

	sort.Slice(logEntries, func(i, j int) bool {
		return logEntries[i].Filename < logEntries[j].Filename
	})

	data := indexData{
		GeneratedAt: time.Now().Format(time.RFC3339),
		Logs:        logEntries,
	}

	if err := os.MkdirAll(filepath.Dir(outputFile), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	file, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %v", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("failed to write JSON: %v", err)
	}

	fmt.Printf("Time range info extracted to %s\n", outputFile)
	fmt.Printf("Processed: %d updated, %d skipped, %d total\n", updatedCount, skippedCount, len(logEntries))

	return nil
}

func main() {
	flag.StringVar(&config.Port, "port", "8888", "server port")
	flag.StringVar(&config.LogDir, "log_dir", "logs", "log directory")
	flag.StringVar(&configFile, "config_file", "projects.json", "projects config file")
	flag.Parse()

	log.Printf("Loaded %d log entries", len(logIndex.Logs))
	log.Printf("Server starting on port %s", config.Port)

	cwd, _ := os.Getwd()
	log.Printf("Working dir: %s", cwd)

	defaultMux := http.NewServeMux()

	defaultMux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path[len("/api/"):]
		switch path {
		case "query":
			queryLogsHandler(w, r)
		case "log_content":
			logContentHandler(w, r)
		case "log_types":
			listLogTypesHandler(w, r)
		case "projects":
			if r.Method == http.MethodGet {
				listProjectsHandler(w, r)
			} else if r.Method == http.MethodPost {
				addProjectHandler(w, r)
			} else if r.Method == http.MethodPut {
				updateProjectHandler(w, r)
			} else if r.Method == http.MethodDelete {
				deleteProjectHandler(w, r)
			}
		case "generate_index":
			generateIndex(w, r)
		default:
			http.NotFound(w, r)
		}
	})

	defaultMux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	defaultMux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(publicFS))))

	defaultMux.HandleFunc("/settings", func(w http.ResponseWriter, r *http.Request) {
		data, err := publicFS.ReadFile("settings.html")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Write(data)
	})

	defaultMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Request path: %s", r.URL.Path)
		data, err := publicFS.ReadFile("index.html")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Write(data)
	})

	log.Fatal(http.ListenAndServe(":"+config.Port, defaultMux))
}
