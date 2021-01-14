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
	fmt.Print("LOL")
}

func main() {

	http.HandleFunc("/", URLRedirectWithCode)
	fmt.Println("Server started at port 8080")
	log.Fatal(http.ListenAndServeTLS(":8080", SSLCrt, SSLKey, nil))
}
