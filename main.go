package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"gopkg.in/yaml.v3"
)

// --- 数据结构 ---
type APIConfig struct {
	URL   string `yaml:"url"`
	Key   string `yaml:"key"`
	Model string `yaml:"model"`
}
type ValidationConfig struct {
	MaxInputChars int `yaml:"max_input_chars"`
}
type Config struct {
	API        APIConfig        `yaml:"api"`
	Validation ValidationConfig `yaml:"validation"`
}
type CompactDef struct {
	POS string `json:"pos"`
	M   string `json:"m"`
	EX  string `json:"ex"`
}
type CompactResponse struct {
	P    string       `json:"p"`
	Defs []CompactDef `json:"defs"`
}

const dbFile = "./dictionary.db"
const promptsDir = "./prompts"

var db *sql.DB
var config Config
var promptCache = make(map[string]string)
var goldenDictTemplate *template.Template

func loadPrompts() {
	files, err := os.ReadDir(promptsDir)
	if err != nil {
		log.Fatalf("Failed to read prompts directory '%s': %v", promptsDir, err)
	}
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".txt") {
			content, err := os.ReadFile(filepath.Join(promptsDir, file.Name()))
			if err != nil {
				log.Printf("Warning: could not read prompt file %s: %v", file.Name(), err)
				continue
			}
			key := strings.TrimSuffix(file.Name(), ".txt")
			promptCache[key] = string(content)
			log.Printf("Loaded prompt template: '%s'", key)
		}
	}
	if len(promptCache) == 0 {
		log.Println("Warning: No prompt templates were loaded. Please check the 'prompts' directory.")
	}
}

func main() {
	// 关键修复：确保在main函数的最开始调用 loadPrompts()
	loadPrompts()

	// 加载YAML配置
	configFile, err := os.ReadFile("config.yaml")
	if err != nil {
		log.Fatalf("Failed to read config.yaml: %v.", err)
	}
	err = yaml.Unmarshal(configFile, &config)
	if err != nil {
		log.Fatalf("Failed to parse config.yaml: %v", err)
	}
	if config.API.Key == "sk-or-your-key-here" || config.API.Key == "" {
		log.Fatal("API key is not set in config.yaml.")
	}
	if config.Validation.MaxInputChars <= 0 {
		config.Validation.MaxInputChars = 50
	}
	log.Println("Configuration loaded successfully.")

	// 加载HTML模板
	goldenDictTemplate = template.Must(template.ParseFiles("templates/goldendict.html"))
	log.Println("HTML template loaded successfully.")

	// 初始化数据库
	db, err = sql.Open("sqlite3", dbFile)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	createTableSQL := `CREATE TABLE IF NOT EXISTS cache ( "word_key" TEXT NOT NULL PRIMARY KEY, "definition" TEXT, "created_at" DATETIME DEFAULT CURRENT_TIMESTAMP );`
	_, err = db.Exec(createTableSQL)
	if err != nil {
		log.Fatalf("Failed to create table: %v", err)
	}
	log.Println("Database initialized successfully.")

	// 注册路由
	http.HandleFunc("/api/config", configHandler())
	http.HandleFunc("/api/lookup", lookupHandler())
	http.HandleFunc("/golden-dict", goldenDictHandler())
	http.Handle("/", http.FileServer(http.Dir("./static")))

	fmt.Println("Server starting on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func getDefinition(word, sourceLang, targetLang string) ([]byte, error) {
	// --- 新增：将接收到的单词统一转换为小写进行处理 ---
	normalizedWord := strings.ToLower(word)
	// --- 标准化结束 ---

	// 后续所有操作都使用这个标准化后的单词
	wordKey := fmt.Sprintf("%s-%s:%s", sourceLang, targetLang, normalizedWord)

	var cachedDefinition string
	err := db.QueryRow("SELECT definition FROM cache WHERE word_key = ?", wordKey).Scan(&cachedDefinition)
	if err == nil {
		log.Printf("Cache hit for key: %s (Original: '%s')", wordKey, word)
		return []byte(cachedDefinition), nil
	}

	log.Printf("Cache miss for key: %s. Calling AI API...", wordKey)

	promptKey := fmt.Sprintf("%s-%s", sourceLang, targetLang)
	promptTemplate, ok := promptCache[promptKey]
	if !ok {
		return nil, fmt.Errorf("unsupported language pair: %s", promptKey)
	}
	// 使用标准化后的单词去填充Prompt
	prompt := strings.Replace(promptTemplate, "${word}", normalizedWord, -1)

	requestBody, _ := json.Marshal(map[string]interface{}{"model": config.API.Model, "messages": []map[string]string{{"role": "user", "content": prompt}}})
	req, _ := http.NewRequest("POST", config.API.URL, bytes.NewBuffer(requestBody))
	req.Header.Set("Authorization", "Bearer "+config.API.Key)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("HTTP-Referer", "http://localhost")
	req.Header.Set("X-Title", "AI Dictionary")

	client := &http.Client{}
	resp, apiErr := client.Do(req)
	if apiErr != nil {
		return nil, fmt.Errorf("failed to call AI API: %v", apiErr)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var aiApiResponse struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &aiApiResponse); err != nil || len(aiApiResponse.Choices) == 0 {
		return nil, fmt.Errorf("failed to parse AI response: %s", string(body))
	}
	contentJSON := aiApiResponse.Choices[0].Message.Content
	if strings.Contains(contentJSON, "```") {
		startIndex := strings.Index(contentJSON, "{")
		endIndex := strings.LastIndex(contentJSON, "}")
		if startIndex != -1 && endIndex != -1 && startIndex < endIndex {
			contentJSON = contentJSON[startIndex : endIndex+1]
		}
	}

	_, err = db.Exec("INSERT INTO cache (word_key, definition) VALUES (?, ?)", wordKey, contentJSON)
	if err != nil {
		log.Printf("Failed to insert into cache: %v", err)
	}

	return []byte(contentJSON), nil
}

func goldenDictHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var word, sourceLang, targetLang string

		if r.URL.Query().Has("word") {
			word = r.URL.Query().Get("word")
			sourceLang = r.URL.Query().Get("source")
			targetLang = r.URL.Query().Get("target")
		} else {
			queryParts := strings.SplitN(r.URL.RawQuery, "-", 3)
			if len(queryParts) == 3 {
				sourceLang = queryParts[0]
				targetLang = queryParts[1]
				word = queryParts[2]
			}
		}

		if word == "" || sourceLang == "" || targetLang == "" {
			w.WriteHeader(http.StatusBadRequest)
			goldenDictTemplate.Execute(w, map[string]interface{}{"Error": "Invalid request format."})
			return
		}

		definitionJSON, err := getDefinition(word, sourceLang, targetLang)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			goldenDictTemplate.Execute(w, map[string]interface{}{"Error": err.Error(), "Word": word})
			return
		}

		var responseData CompactResponse
		err = json.Unmarshal(definitionJSON, &responseData)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			goldenDictTemplate.Execute(w, map[string]interface{}{"Error": "Failed to parse AI JSON response.", "Word": word})
			return
		}

		templateData := map[string]interface{}{"Word": word, "Result": responseData, "Error": nil}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		goldenDictTemplate.Execute(w, templateData)
	}
}

func lookupHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		word := r.URL.Query().Get("word")
		sourceLang := r.URL.Query().Get("source")
		targetLang := r.URL.Query().Get("target")
		if word == "" || sourceLang == "" || targetLang == "" {
			http.Error(w, "Missing required parameters (word, source, target)", http.StatusBadRequest)
			return
		}
		maxChars := config.Validation.MaxInputChars
		if len([]rune(word)) > maxChars {
			http.Error(w, fmt.Sprintf("Input too long. Max %d characters allowed.", maxChars), http.StatusBadRequest)
			return
		}
		definitionJSON, err := getDefinition(word, sourceLang, targetLang)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(definitionJSON)
	}
}

func configHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		availablePairs := make(map[string][]string)
		for key := range promptCache {
			parts := strings.Split(key, "-")
			if len(parts) == 2 {
				source, target := parts[0], parts[1]
				availablePairs[source] = append(availablePairs[source], target)
			}
		}
		frontendConfig := map[string]interface{}{"max_input_chars": config.Validation.MaxInputChars, "available_pairs": availablePairs}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(frontendConfig)
	}
}
