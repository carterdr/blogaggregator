package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/carterdr/blogaggregator/internal/database"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	DB *database.Queries
}

func main() {
	godotenv.Load(".env")
	port := os.Getenv("PORT")
	if port == "" {
		log.Fatal("PORT environment variable is not set")
	}
	dbURL := os.Getenv("CONNECTION")
	if dbURL == "" {
		log.Fatal("CONNECTION environment variable is not set")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal(err)
	}
	dbQueries := database.New(db)
	apiCfg := apiConfig{DB: dbQueries}

	serveMux := http.NewServeMux()
	server := &http.Server{Handler: serveMux, Addr: ":" + port}

	serveMux.HandleFunc("GET /v1/healthz", handlerReady)
	serveMux.HandleFunc("GET /v1/err", handlerError)
	serveMux.HandleFunc("POST /v1/users", apiCfg.handlerUsersCreate)
	serveMux.HandleFunc("GET /v1/users", apiCfg.middlewareAuth(apiCfg.handlerUserGet))

	serveMux.HandleFunc("POST /v1/feeds", apiCfg.middlewareAuth(apiCfg.handlerFeedCreate))
	serveMux.HandleFunc("GET /v1/feeds", apiCfg.handlerGetAllFeeds)

	serveMux.HandleFunc("POST /v1/feed_follows", apiCfg.middlewareAuth(apiCfg.handlerFeedFollowCreate))
	serveMux.HandleFunc("GET /v1/feed_follows", apiCfg.middlewareAuth(apiCfg.handlerFeedFollowGet))
	serveMux.HandleFunc("DELETE /v1/feed_follows/{feedFollowID}", apiCfg.middlewareAuth(apiCfg.handlerFeedFollowDelete))

	serveMux.HandleFunc("GET /v1/posts", apiCfg.middlewareAuth(apiCfg.handlerPostsGet))

	const concurrency = 10
	const collectionInterval = time.Minute

	go startScraping(dbQueries, concurrency, collectionInterval)

	log.Printf("Serving on port: %s\n", port)
	log.Fatal(server.ListenAndServe())
}
