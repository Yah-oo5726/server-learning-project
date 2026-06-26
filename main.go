package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync/atomic"

	"github.com/Yah-oo5726/server-learning-project/internal/database"
	"github.com/joho/godotenv"

	_ "github.com/lib/pq"
)

type apiConfig struct {
	fileserverHits atomic.Int64
	db *database.Queries
}

func (c *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func main() {
	godotenv.Load()
	dbURL := os.Getenv("DATABASE_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		fmt.Println("Error connecting to the database:", err)
		return
	}
	defer db.Close()
	dbQueries := database.New(db)
	config := apiConfig{db: dbQueries}
	servemux := http.NewServeMux()
	apphandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.StripPrefix("/app", http.FileServer(http.Dir("."))).ServeHTTP(w, r)
	})
	servemux.Handle("/app/", config.middlewareMetricsInc(apphandler))
	servemux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("OK"))
		if err != nil {
			fmt.Println("Error writing response:", err)
		}
	})
	servemux.HandleFunc("GET /admin/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(fmt.Sprintf("<html>\n<body>\n<h1>Welcome, Chirpy Admin</h1>\n<p>Chirpy has been visited %d times!</p>\n</body>\n</html>", config.fileserverHits.Load())))
		if err != nil {
			fmt.Println("Error writing response:", err)
		}
	})
	servemux.HandleFunc("POST /admin/reset", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		config.fileserverHits.Store(0)
		_, err := w.Write([]byte("Hits reset to 0\n"))
		if err != nil {
			fmt.Println("Error writing response:", err)
		}
	})
	servemux.HandleFunc("POST /api/validate_chirp", func(w http.ResponseWriter, r *http.Request) {
		type parameters struct {
			Body string `json:"body"`
		}
		message := parameters{}
		err := json.NewDecoder(r.Body).Decode(&message)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid JSON payload")
			return
		}
		if len(message.Body) > 140 {
			respondWithError(w, http.StatusBadRequest, "Message exceeds 140 characters")
			return
		}
		respondWithJSON(w, http.StatusOK, successResponse{CleanedBody: cleanMessage(message.Body)})
	})
	server := http.Server{Handler: servemux, Addr: ":8080"}
	server.ListenAndServe()
}
