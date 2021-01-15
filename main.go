package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	// DBHost is the address of the Database
	DBHost = os.Getenv("SEUS_DBHOST")
	// DBPort is the port in which the DB is reachable
	DBPort = os.Getenv("SEUS_DBPORT")
	// DBUser is the username of mongodb
	DBUser = os.Getenv("SEUS_DBUSER")
	// DBPass is the password for the user
	DBPass = os.Getenv("SEUS_DBPASS")
	// DBName is the name of the database
	DBName = os.Getenv("SEUS_DBNAME")
	// SSLCrt is the SSL certificate location
	SSLCrt = os.Getenv("SEUS_SSLCRT")
	// SSLKey is the SSL key location
	SSLKey = os.Getenv("SEUS_SSLKEY")
	// DBClient is the object representing the connection
	DBClient interface{}
)

// Redirect redirects the requests from http to https
func Redirect(w http.ResponseWriter, req *http.Request) {
	// target := "https://" + req.Host + req.URL.Path
	target := "https://localhost:8081"
	if len(req.URL.RawQuery) > 0 {
		target += "?" + req.URL.RawQuery
	}
	log.Printf("redirect to: %s", target)
	http.Redirect(w, req, target, http.StatusTemporaryRedirect)
}

// ConnectDB is a function that connects the database and returns the client
func ConnectDB() (interface{}, error) {
	DBConnectionURI := fmt.Sprintf("mongodb://%v:%v@%v:%v/%v", DBUser, DBPass, DBHost, DBPort, DBName)
	client, err := mongo.NewClient(options.Client().ApplyURI(DBConnectionURI))
	if err != nil {
		return nil, err
	}
	return client, nil
}

// URLRedirectWithCode gets a code and redirects to the shortened URL
func URLRedirectWithCode(w http.ResponseWriter, r *http.Request) {
	//code := r.URL.Path[1:]
	target := "https://www.google.com"

	http.Redirect(w, r, target, http.StatusTemporaryRedirect)
}

func main() {

	// Redirect from port 80 to port 443
	go http.ListenAndServe(":8080", http.HandlerFunc(Redirect))

	// Handle base path (even with code)
	http.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
		URLRedirectWithCode(rw, r)
	})

	// Api handling
	http.HandleFunc("/api", func(rw http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(rw, "Hit api")
	})

	http.HandleFunc("/api/create", func(rw http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(rw, "Hit api/create")
	})

	http.HandleFunc("/api/search", func(rw http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(rw, "Hit api/search")
	})

	fmt.Println("Server started at port 8081")
	log.Fatal(http.ListenAndServeTLS(":8081", SSLCrt, SSLKey, nil))
}
