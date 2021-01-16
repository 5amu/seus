package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ConfigFile is the file that loads the user configurations
const (
	ConfigFile = "./config.json"
	LogFile    = "./seus.log"
)

// ConfigHost is a wrapper for configs about the host
type ConfigHost struct {
	Domain    string `json:"domain"`
	HTTPPort  string `json:"httpport"`
	HTTPSport string `json:"httpsport"`
}

// ConfigDB is a wrapper for configs about the db
type ConfigDB struct {
	Domain     string `json:"domain"`
	Port       string `json:"port"`
	User       string `json:"user"`
	Passwd     string `json:"passwd"`
	Name       string `json:"name"`
	Collection string `json:"collection"`
}

// ConfigSSL is a wrapper for configs about the certs
type ConfigSSL struct {
	Cert string `json:"cert"`
	Key  string `json:"key"`
}

// Config wraps the configurations in
type Config struct {
	Host ConfigHost `json:"host"`
	DB   ConfigDB   `json:"db"`
	SSL  ConfigSSL  `json:"ssl"`
}

// Seus is a wrapper for all the info in the database
type Seus struct {
	Code string `json:"code,omitempty"`
	URL  string `json:"url,omitempty"`
	// Counter int    `json:"counter,omitempty"`
}

// Redirect redirects the requests from http to https
func Redirect(w http.ResponseWriter, req *http.Request, c Config) {
	target := "https://" + c.Host.Domain + ":" + c.Host.HTTPSport
	if len(req.URL.RawQuery) > 0 {
		target += "?" + req.URL.RawQuery
	}
	log.Printf("redirect to: %s", target)
	http.Redirect(w, req, target, http.StatusTemporaryRedirect)
}

// ConnectDB is a function that connects the database and returns the client
func ConnectDB(c Config) (*mongo.Client, error) {
	DBUser, DBPass, DBHost, DBPort := c.DB.User, c.DB.Passwd, c.DB.Domain, c.DB.Port
	DBConnectionURI := fmt.Sprintf("mongodb://%v:%v@%v:%v", DBUser, DBPass, DBHost, DBPort)
	client, err := mongo.NewClient(options.Client().ApplyURI(DBConnectionURI))
	if err != nil {
		return nil, err
	}
	return client, nil
}

// URLRedirectWithCode gets a code and redirects to the shortened URL
func URLRedirectWithCode(w http.ResponseWriter, r *http.Request, c Config) error {
	var res Seus

	// Get the code
	code := r.URL.Path[1:]

	// Connect to the DB
	dbclient, err := ConnectDB(c)
	if err != nil {
		return err
	}
	defer dbclient.Disconnect(context.Background())

	// Define filters
	filter := bson.D{bson.E{Key: "code", Value: code}}
	// Access the collection
	collection := dbclient.Database(c.DB.Name).Collection(c.DB.Collection)
	// Run the query
	err = collection.FindOne(context.Background(), filter).Decode(&res)
	if err != mongo.ErrNoDocuments {
		if err != nil {
			return err
		}
	}

	target := res.URL

	http.Redirect(w, r, target, http.StatusTemporaryRedirect)
	return nil
}

func main() {
	var configs Config

	f, err := os.OpenFile(LogFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
	defer f.Close()
	log.SetOutput(f)

	reader, err := ioutil.ReadFile(ConfigFile)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	if err := json.Unmarshal([]byte(reader), &configs); err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	// Redirect from port 80 to port 443
	go http.ListenAndServe(":"+configs.Host.HTTPPort, http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		Redirect(rw, r, configs)
	}))

	// Handle base path (even with code)
	http.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
		if err := URLRedirectWithCode(rw, r, configs); err != nil {
			log.Println(err)
		}
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

	log.Println("Server started at port 8080 and 8443")
	log.Fatal(http.ListenAndServeTLS(":"+configs.Host.HTTPSport, configs.SSL.Cert, configs.SSL.Key, nil))
}
