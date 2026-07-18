package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync/atomic"
	"time"

	"github.com/Yah-oo5726/server-learning-project/internal/auth"
	"github.com/Yah-oo5726/server-learning-project/internal/database"
	"github.com/google/uuid"
	"github.com/joho/godotenv"

	_ "github.com/lib/pq"
)

type apiConfig struct {
	fileserverHits atomic.Int64
	jwtSecret      string
	db             *database.Queries
}

type User struct {
	ID           uuid.UUID `json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Email        string    `json:"email"`
	JWT          string    `json:"token,omitempty"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	IsChirpyRed  bool      `json:"is_chirpy_red"`
}

type Chirp struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserID    uuid.UUID `json:"user_id"`
}

type JWTtoken struct {
	Token string `json:"token"`
}

func (c *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func main() {
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		fmt.Println("Error connecting to the database:", err)
		return
	}
	defer db.Close()
	dbQueries := database.New(db)
	config := apiConfig{db: dbQueries, jwtSecret: os.Getenv("JWT_SECRET")}
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
		_, err := fmt.Fprintf(w, "<html>\n<body>\n<h1>Welcome, Chirpy Admin</h1>\n<p>Chirpy has been visited %d times!</p>\n</body>\n</html>", config.fileserverHits.Load())
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
		err = config.db.ResetUsers(r.Context())
		if err != nil {
			log.Println(err)
			respondWithError(w, http.StatusInternalServerError, "Error resetting users")
			return
		}
		err = config.db.ResetChirps(r.Context())
		if err != nil {
			log.Println(err)
			respondWithError(w, http.StatusInternalServerError, "Error resetting chirps")
			return
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
	servemux.HandleFunc("POST /api/users", func(w http.ResponseWriter, r *http.Request) {
		type parameters struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		message := parameters{}
		err := json.NewDecoder(r.Body).Decode(&message)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid JSON payload")
			return
		}
		password, err := auth.HashPassword(message.Password)
		if err != nil {
			log.Println(err)
			respondWithError(w, http.StatusInternalServerError, "error hashing password")
			return
		}
		user, err := config.db.CreateUser(r.Context(), database.CreateUserParams{Email: message.Email, HashedPassword: password})
		if err != nil {
			log.Println(err)
			respondWithError(w, http.StatusInternalServerError, "Error creating user")
			return
		}
		userStruct := User{
			ID:          user.ID,
			CreatedAt:   user.CreatedAt,
			UpdatedAt:   user.UpdatedAt,
			Email:       user.Email,
			IsChirpyRed: user.IsChirpyRed.Bool,
		}
		respondWithJSON(w, 201, userStruct)
	})
	servemux.HandleFunc("POST /api/login", func(w http.ResponseWriter, r *http.Request) {
		type parameters struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		message := parameters{}
		err := json.NewDecoder(r.Body).Decode(&message)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid JSON payload")
			return
		}
		user, err := config.db.GetUserByEmail(r.Context(), message.Email)
		if err != nil {
			log.Println(err)
			respondWithError(w, http.StatusInternalServerError, "Error getting user from database")
			return
		}
		matches, err := auth.CheckPasswordHash(message.Password, user.HashedPassword)
		if err != nil {
			log.Println(err)
			respondWithError(w, http.StatusInternalServerError, "Error checking password")
			return
		}
		jwt, err := auth.MakeJWT(user.ID, config.jwtSecret, 3600*time.Second)
		if err != nil {
			log.Println(err)
			respondWithError(w, http.StatusInternalServerError, "Error creating JWT")
			return
		}
		refreshToken := auth.MakeRefreshToken()
		_, err = config.db.CreateRefreshToken(r.Context(), database.CreateRefreshTokenParams{Token: refreshToken, UserID: user.ID, ExpiresAt: time.Now().Add(144 * time.Hour)})
		if err != nil {
			log.Println(err)
			respondWithError(w, http.StatusInternalServerError, "Error creating refresh token")
			return
		}
		userStruct := User{
			ID:           user.ID,
			CreatedAt:    user.CreatedAt,
			UpdatedAt:    user.UpdatedAt,
			Email:        user.Email,
			JWT:          jwt,
			RefreshToken: refreshToken,
			IsChirpyRed:  user.IsChirpyRed.Bool,
		}
		if matches {
			respondWithJSON(w, 200, userStruct)
		} else {
			respondWithError(w, http.StatusUnauthorized, "Incorrect email or password")
		}
	})
	servemux.HandleFunc("POST /api/chirps", func(w http.ResponseWriter, r *http.Request) {
		type parameters struct {
			Body string `json:"body"`
		}
		message := parameters{}
		err := json.NewDecoder(r.Body).Decode(&message)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid JSON payload")
			return
		}
		token, err := auth.GetBearerToken(r.Header)
		if err != nil {
			respondWithError(w, http.StatusUnauthorized, "Missing or invalid authorization header")
			return
		}
		tokenID, err := auth.ValidateJWT(token, config.jwtSecret)
		if err != nil {
			respondWithError(w, http.StatusUnauthorized, "Error validating JWT")
			return
		}
		chirp, err := config.db.CreateChirp(r.Context(), database.CreateChirpParams{
			Body:   message.Body,
			UserID: tokenID,
		})
		if err != nil {
			log.Println(err)
			respondWithError(w, http.StatusInternalServerError, "Error creating chirp")
			return
		}
		chirpStruct := Chirp{
			ID:        chirp.ID,
			CreatedAt: chirp.CreatedAt,
			UpdatedAt: chirp.UpdatedAt,
			Body:      chirp.Body,
			UserID:    chirp.UserID,
		}
		respondWithJSON(w, 201, chirpStruct)
	})
	servemux.HandleFunc("GET /api/chirps", func(w http.ResponseWriter, r *http.Request) {
		chirps, err := config.db.GetChirps(r.Context())
		if err != nil {
			log.Println(err)
			respondWithError(w, http.StatusInternalServerError, "Error fetching chirps")
			return
		}
		chirpStructs := make([]Chirp, len(chirps))
		for i, chirp := range chirps {
			chirpStructs[i] = Chirp{
				ID:        chirp.ID,
				CreatedAt: chirp.CreatedAt,
				UpdatedAt: chirp.UpdatedAt,
				Body:      chirp.Body,
				UserID:    chirp.UserID,
			}
		}
		respondWithJSON(w, http.StatusOK, chirpStructs)
	})
	servemux.HandleFunc("GET /api/chirps/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := uuid.Parse(r.PathValue("id"))
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid chirp ID")
			return
		}
		chirp, err := config.db.GetChirpByID(r.Context(), id)
		if err != nil {
			log.Println(err)
			respondWithError(w, http.StatusNotFound, "Chirp not found")
			return
		}
		chirpStruct := Chirp{
			ID:        chirp.ID,
			CreatedAt: chirp.CreatedAt,
			UpdatedAt: chirp.UpdatedAt,
			Body:      chirp.Body,
			UserID:    chirp.UserID,
		}
		respondWithJSON(w, http.StatusOK, chirpStruct)
	})
	servemux.HandleFunc("POST /api/refresh", func(w http.ResponseWriter, r *http.Request) {
		token, err := auth.GetBearerToken(r.Header)
		if err != nil {
			respondWithError(w, http.StatusUnauthorized, "Missing or invalid authorization header")
			return
		}
		tokenInfo, err := config.db.GetRefreshToken(r.Context(), token)
		if err != nil || tokenInfo.ExpiresAt.Before(time.Now()) || tokenInfo.RevokedAt.Valid {
			respondWithError(w, http.StatusUnauthorized, "Invalid refresh token")
			return
		}
		jwtToken, err := auth.MakeJWT(tokenInfo.UserID, config.jwtSecret, 3600*time.Second)
		if err != nil {
			log.Println(err)
			respondWithError(w, http.StatusInternalServerError, "Error creating JWT")
			return
		}
		respondWithJSON(w, http.StatusOK, JWTtoken{Token: jwtToken})
	})
	servemux.HandleFunc("POST /api/revoke", func(w http.ResponseWriter, r *http.Request) {
		token, err := auth.GetBearerToken(r.Header)
		if err != nil {
			respondWithError(w, http.StatusUnauthorized, "Missing or invalid authorization header")
			return
		}
		config.db.RevokeRefreshToken(r.Context(), token)
		respondWithJSON(w, 204, "genuinely nothing dawg i have nothing to return")
	})
	servemux.HandleFunc("PUT /api/users", func(w http.ResponseWriter, r *http.Request) {
		type parameters struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		message := parameters{}
		err := json.NewDecoder(r.Body).Decode(&message)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid JSON payload")
			return
		}
		token, err := auth.GetBearerToken(r.Header)
		if err != nil {
			respondWithError(w, http.StatusUnauthorized, "Missing or invalid authorization header")
			return
		}
		userID, err := auth.ValidateJWT(token, config.jwtSecret)
		if err != nil {
			respondWithError(w, http.StatusUnauthorized, "Invalid JWT")
			return
		}
		password, err := auth.HashPassword(message.Password)
		if err != nil {
			log.Println(err)
			respondWithError(w, http.StatusInternalServerError, "error hashing password")
			return
		}
		user, err := config.db.ChangeCredentials(r.Context(), database.ChangeCredentialsParams{ID: userID, Email: message.Email, HashedPassword: password})
		if err != nil {
			log.Println(err)
			respondWithError(w, http.StatusInternalServerError, "Error updating user")
			return
		}
		respondWithJSON(w, http.StatusOK, User{ID: user.ID, Email: user.Email, CreatedAt: user.CreatedAt, UpdatedAt: user.UpdatedAt, IsChirpyRed: user.IsChirpyRed.Bool})
	})
	servemux.HandleFunc("DELETE /api/chirps/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := uuid.Parse(r.PathValue("id"))
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid chirp ID")
			return
		}
		token, err := auth.GetBearerToken(r.Header)
		if err != nil {
			respondWithError(w, http.StatusUnauthorized, "Missing or invalid authorization header")
			return
		}
		userID, err := auth.ValidateJWT(token, config.jwtSecret)
		if err != nil {
			respondWithError(w, http.StatusUnauthorized, "Invalid JWT")
			return
		}
		chirp, err := config.db.GetChirpByID(r.Context(), id)
		if err != nil {
			log.Println(err)
			respondWithError(w, http.StatusNotFound, "Chirp not found")
			return
		}
		if chirp.UserID != userID {
			respondWithError(w, http.StatusForbidden, "You are not allowed to delete this chirp")
			return
		}
		err = config.db.DeleteChirpByID(r.Context(), id)
		if err != nil {
			log.Println(err)
			respondWithError(w, http.StatusInternalServerError, "Error deleting chirp")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	servemux.HandleFunc("POST /api/polka/webhooks", func(w http.ResponseWriter, r *http.Request) {
		type parameters struct {
			Event string `json:"event"`
			Data  struct {
				UserID string `json:"user_id"`
			} `json:"data"`
		}
		message := parameters{}
		err := json.NewDecoder(r.Body).Decode(&message)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid JSON payload")
			return
		}
		if message.Event != "user.upgraded" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		userID, err := uuid.Parse(message.Data.UserID)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid user ID")
			return
		}
		err = config.db.SetUserChirpyRed(r.Context(), userID)
		if err != nil {
			log.Println(err)
			respondWithError(w, http.StatusNotFound, "User Not Found")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	server := http.Server{Handler: servemux, Addr: ":8080"}
	server.ListenAndServe()
}
