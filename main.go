package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ConfigFile is the file that loads the user configurations
const ConfigFile = "./config.json"

// ConfigHost is a wrapper for configs about the host
type ConfigHost struct {
	Domain    string `json:"domain"`
	HTTPPort  string `json:"httpport"`
	HTTPSport string `json:"httpsport"`
}

// ConfigDB is a wrapper for configs about the db
type ConfigDB struct {
	Domain string `json:"domain"`
	Port   string `json:"port"`
	User   string `json:"user"`
	Passwd string `json:"passwd"`
	Name   string `json:"name"`
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
func ConnectDB(c Config) (interface{}, error) {
	DBUser, DBPass, DBHost, DBPort, DBName := c.DB.User, c.DB.Passwd, c.DB.Domain, c.DB.Port, c.DB.Name
	DBConnectionURI := fmt.Sprintf("mongodb://%v:%v@%v:%v/%v", DBUser, DBPass, DBHost, DBPort, DBName)
	client, err := mongo.NewClient(options.Client().ApplyURI(DBConnectionURI))
	if err != nil {
		return nil, err
	}
	return client, nil
}

// URLRedirectWithCode gets a code and redirects to the shortened URL
func URLRedirectWithCode(w http.ResponseWriter, r *http.Request, c Config) {
	//code := r.URL.Path[1:]
	target := "https://www.google.com"

	http.Redirect(w, r, target, http.StatusTemporaryRedirect)
}

func main() {
	var configs Config

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
	go http.ListenAndServe(":8080", http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		Redirect(rw, r, configs)
	}))

	// Handle base path (even with code)
	http.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
		URLRedirectWithCode(rw, r, configs)
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
	log.Fatal(http.ListenAndServeTLS(":8081", configs.SSL.Cert, configs.SSL.Key, nil))
}
