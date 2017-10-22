package main

import "fmt"
import "encoding/json"
import "net/http"
import "os"
import "io/ioutil"
import "log"
import "strings"
import "math/rand"
import "time"
import "reflect"

type Configuration struct {
	Token         string
	RootDir       string
	Port          string
	StaticBaseURL string
}

var Conf Configuration = Configuration{}

func initConf() {
	// Development
	// file, _ := os.Open("conf.json")

	// Production
	file, _ := os.Open("/root/dogbot/prod.conf.json")
	decoder := json.NewDecoder(file)
	err := decoder.Decode(&Conf)
	if err != nil {
		fmt.Println("error:", err)
	}
}

func initBreeds(breeds map[string]string) {
	files, err := ioutil.ReadDir(Conf.RootDir + "static/")
	if err != nil {
		log.Fatal(err)
	}
	for _, file := range files {
		spl := strings.Split(file.Name(), "-")
		breed := strings.ToLower(spl[1])
		breed = strings.Replace(breed, "_", " ", -1)
		breeds[breed] = file.Name()
	}
}

// Builds a url to a random image from the requested directory
func getRandomImageUrl(imgDir string) string {
	files, err := ioutil.ReadDir(Conf.RootDir + "static/" + imgDir)
	if err != nil {
		log.Fatal(err)
	}
	idx := rand.Intn(len(files))
	file := files[idx]
	url := Conf.StaticBaseURL + imgDir + "/" + file.Name()
	return url
}

// Levenshtein distance between two strings
func levenshtein(a string, b string) int {
	return 0
}

// Makes a guess at the requested category
func parseBreedQuery(query string, breeds []string) string {
	// TODO: Calculate levenshtein distance between this string and all other strings
	return query
}

func main() {
	rand.Seed(time.Now().UnixNano())

	initConf()

	staticDir := Conf.RootDir + "static/"

	// Serve the images
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir))))

	go http.ListenAndServe(Conf.Port, nil)

	// Start an RTM session
	ws, id := slackConnect(Conf.Token)

	// build the map of human readable breed names to source directory
	var breeds map[string]string = make(map[string]string)
	initBreeds(breeds)
	keys := reflect.ValueOf(breeds).MapKeys()
	breedsAvailable := make([]string, len(keys))
	for i := 0; i < len(keys); i++ {
		breedsAvailable[i] = keys[i].String()
	}

	for {
		// read each incoming message
		m, err := getMessage(ws)
		if err != nil {
			log.Fatal(err)
		}

		// see if we're mentioned
		if m.Type == "message" && strings.HasPrefix(m.Text, "<@"+id+">") {
			fmt.Println(m)
			go func(m Message) {
				// Strip @ id
				breed := strings.Replace(m.Text, "<@"+id+"> ", "", -1)
				fmt.Println("Attempting to fetch photo for breed: " + breed)
				breed = parseBreedQuery(breed, breedsAvailable)
				if imgDir, ok := breeds[breed]; ok {
					msg := getRandomImageUrl(imgDir)
					m.Text = msg
				} else {
					m.Text = "Sorry, I don't know that dog."
				}
				postMessage(ws, m)
			}(m)
		}
	}
}
