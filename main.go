package main

import "fmt"
import "encoding/json"
import "net/http"
import "os"
import "io/ioutil"
import "log"
import "strings"
import "math/rand"

type Configuration struct {
	Token    string
	ImageDir string
}

var Conf Configuration = Configuration{}

func initConf() {
	file, _ := os.Open("conf.json")
	decoder := json.NewDecoder(file)
	err := decoder.Decode(&Conf)
	if err != nil {
		fmt.Println("error:", err)
	}
}

func initBreeds(breeds map[string]string) {
	files, err := ioutil.ReadDir("static/")
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

// Returns a url to a random image of the input breed
func getImageUrl(breeds map[string]string, breed string) string {
	// Development
	// base := "http://localhost:8080/static/"
	// Production
	base := "http://dogbot.freddysplant.com/static/"
	imgDir := breeds[breed]
	files, err := ioutil.ReadDir("static/" + imgDir)
	if err != nil {
		log.Fatal(err)
	}
	idx := rand.Intn(len(files))
	file := files[idx]
	url := base + imgDir + "/" + file.Name()
	return url
}

func main() {
	initConf()

	// Serve the images
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	// Development
	// go http.ListenAndServe(":5555", nil)

	// Production
	go http.ListenAndServe(":8080", nil)

	// Start an RTM session
	ws, id := slackConnect(Conf.Token)

	// build the map of human readable breed names to source directory
	var breeds map[string]string = make(map[string]string)
	initBreeds(breeds)

	fmt.Println(getImageUrl(breeds, "pug"))

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
				m.Text = getImageUrl(breeds, breed)
				postMessage(ws, m)
			}(m)
		}
	}
}
