package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
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

// SeusResponse is a wrapper for json response
type SeusResponse struct {
	Status  int    `json:"status,omitempty"`
	Message string `json:"message,omitempty"`
	URL     string `json:"url,omitempty"`
	Code    string `json:"code,omitempty"`
	Encoded string `json:"encoded,omitempty"`
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

func connectDB(c Config) (*mongo.Client, error) {
	DBUser, DBPass, DBHost, DBPort := c.DB.User, c.DB.Passwd, c.DB.Domain, c.DB.Port
	DBConnectionURI := fmt.Sprintf("mongodb://%v:%v@%v:%v", DBUser, DBPass, DBHost, DBPort)
	client, err := mongo.NewClient(options.Client().ApplyURI(DBConnectionURI))
	if err != nil {
		return nil, err
	}
	return client, nil
}

func getResult(c Config, filter primitive.D) (Seus, error) {
	var res Seus
	client, err := connectDB(c)
	if err != nil {
		return res, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	if err != nil {
		log.Fatal(err)
	}
	defer cancel()
	err = client.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(ctx)
	collection := client.Database(c.DB.Name).Collection(c.DB.Collection)
	err = collection.FindOne(context.Background(), filter).Decode(&res)
	if err != nil {
		return res, err
	}
	return res, nil
}

func generateCode(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

	s := make([]rune, n)
	for i := range s {
		s[i] = letters[rand.Intn(len(letters))]
	}
	return string(s)
}

// URLRedirectWithCode gets a code and redirects to the shortened URL
func URLRedirectWithCode(w http.ResponseWriter, r *http.Request, c Config) error {
	// Get the code
	code := r.URL.Path[1:]
	// Define filters
	filter := bson.D{bson.E{Key: "code", Value: code}}
	// Get result by filter
	res, err := getResult(c, filter)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			out := new(SeusResponse)
			out.Status = 404
			out.Code = code
			out.Message = "Code not found"
			mar, _ := json.Marshal(out)
			log.Printf(string(mar))
			fmt.Fprintf(w, string(mar))
		}
		return err
	}
	log.Printf("Redirecting IP %v with code %v to URL %v", r.RemoteAddr, code, res.URL)
	// Redirect to correct URL
	http.Redirect(w, r, res.URL, http.StatusTemporaryRedirect)
	return nil
}

func main() {
	var configs Config
	// Open LOG file in append mode (or create mode)
	f, err := os.OpenFile(LogFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
	defer f.Close()
	// Set log target to the file just opened
	log.SetOutput(f)

	// Open config file
	reader, err := ioutil.ReadFile(ConfigFile)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	// Parse config file
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
		reCode := regexp.MustCompile("^[a-zA-Z_0-9]{7,}")
		rePath := regexp.MustCompile("/")
		if reCode.Match([]byte(r.URL.Path[1:])) || rePath.Match([]byte(r.URL.Path[1:])) {
			http.NotFound(rw, r)
			return
		}
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

	// Log the start of the server + start listen on https
	log.Println("Server started at port 8080 and 8443")
	log.Fatal(http.ListenAndServeTLS(":"+configs.Host.HTTPSport, configs.SSL.Cert, configs.SSL.Key, nil))
}
