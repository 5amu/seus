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
	CodeLength = 6
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
	// Write values from the config file
	DBUser, DBPass, DBHost, DBPort := c.DB.User, c.DB.Passwd, c.DB.Domain, c.DB.Port
	// Connect to the database
	DBConnectionURI := fmt.Sprintf("mongodb://%v:%v@%v:%v", DBUser, DBPass, DBHost, DBPort)
	client, err := mongo.NewClient(options.Client().ApplyURI(DBConnectionURI))
	if err != nil {
		return nil, err
	}
	return client, nil
}

func getResult(c Config, filter primitive.D) (Seus, error) {
	var res Seus
	// Connect to client
	client, err := connectDB(c)
	if err != nil {
		return res, err
	}
	// Generate a context
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	if err != nil {
		log.Fatal(err)
	}
	defer cancel()
	// Connect to the database with the context
	err = client.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(ctx)
	// Get collection
	collection := client.Database(c.DB.Name).Collection(c.DB.Collection)
	// Retrieve result with filter from the collection
	err = collection.FindOne(context.Background(), filter).Decode(&res)
	if err != nil {
		return res, err
	}
	return res, nil
}

func insertData(data Seus, c Config) error {
	// Connect to client
	client, err := connectDB(c)
	if err != nil {
		return err
	}
	// Generate a context
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	if err != nil {
		log.Fatal(err)
	}
	defer cancel()
	// Connect to the database with the context
	err = client.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(ctx)
	// Get collection
	collection := client.Database(c.DB.Name).Collection(c.DB.Collection)
	// Insert new value into the collection
	_, err = collection.InsertOne(ctx, data)
	if err != nil {
		log.Fatal(err)
	}
	return nil
}

func generateCode(n int, c Config) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_")
	// Generate the first code
	s := make([]rune, n)
	for i := range s {
		s[i] = letters[rand.Intn(len(letters))]
	}
	// Generate a code since we don't find a matching one in the db
	_, err := getResult(c, bson.D{bson.E{Key: "code", Value: s}})
	for err != mongo.ErrNoDocuments {
		for i := range s {
			s[i] = letters[rand.Intn(len(letters))]
		}
	}
	// Return the code
	return string(s)
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
		reCode := regexp.MustCompile("^[a-zA-Z_0-9]{" + fmt.Sprintf("%d", CodeLength+1) + ",}")
		rePath := regexp.MustCompile("/")
		if reCode.Match([]byte(r.URL.Path[1:])) || rePath.Match([]byte(r.URL.Path[1:])) {
			http.NotFound(rw, r)
			return
		}
		// Get the code
		code := r.URL.Path[1:]
		// Define filters
		filter := bson.D{bson.E{Key: "code", Value: code}}
		// Get result by filter
		res, err := getResult(configs, filter)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				out := *new(SeusResponse)
				out.Status = 404
				out.Code = code
				out.Message = "Code not found"
				mar, _ := json.Marshal(out)
				log.Printf(string(mar))
				fmt.Fprintf(rw, string(mar))
			}
			return
		}
		log.Printf("Redirecting IP %v with code %v to URL %v", r.RemoteAddr, code, res.URL)
		// Redirect to correct URL
		http.Redirect(rw, r, string(res.URL), http.StatusFound)
	})

	// Api handling
	http.HandleFunc("/api/create", func(rw http.ResponseWriter, r *http.Request) {
		out := *new(SeusResponse)
		query := r.URL.Query()
		qurl, present := query["url"]
		if !present || len(qurl) != 1 {
			out.Status = 400
			out.Message = "No URL Specified (or too many)"
			mar, _ := json.Marshal(out)
			log.Printf(string(mar))
			fmt.Fprintf(rw, string(mar))
			return
		}
		url := qurl[0]
		re := regexp.MustCompile("^http(s){0,1}://[a-zA-Z0-9_.-]+$")
		if !re.Match([]byte(url)) {
			out.Status = 400
			out.Message = "Unexpected characters in URL, allowed are: [a-zA-Z0-9_.-]"
			mar, _ := json.Marshal(out)
			log.Printf(string(mar))
			fmt.Fprintf(rw, string(mar))
			return
		}
		filter := bson.D{bson.E{Key: "url", Value: url}}

		res, err := getResult(configs, filter)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				data := *new(Seus)
				data.Code = generateCode(CodeLength, configs)
				data.URL = url
				insertData(data, configs)

				out.Status = 200
				out.Code = data.Code
				out.URL = data.URL
				out.Message = "Code inserted correctly"
				out.Encoded = configs.Host.Domain + "/" + data.Code
				mar, _ := json.Marshal(out)
				log.Printf(string(mar))
				fmt.Fprintf(rw, string(mar))
				return
			}
		}

		out.Status = 400
		out.Code = res.Code
		out.URL = res.URL
		out.Message = "Code already exists"
		out.Encoded = configs.Host.Domain + "/" + res.Code
		mar, _ := json.Marshal(out)
		log.Printf(string(mar))
		fmt.Fprintf(rw, string(mar))
	})

	http.HandleFunc("/api/search", func(rw http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(rw, "Hit api/search")
	})

	// Log the start of the server + start listen on https
	log.Println("Server started at port 8080 and 8443")
	log.Fatal(http.ListenAndServeTLS(":"+configs.Host.HTTPSport, configs.SSL.Cert, configs.SSL.Key, nil))
}
