package main

import "time"
import "log"
import "fmt"
import "sort"
import "reflect"
import "strings"
import "strconv"
import "math/rand"
import "net/http"
import "github.com/prometheus/client_golang/prometheus"

// Prometheus stuff
// TODO: Consider moving to different file
var levDists = prometheus.NewGauge(prometheus.GaugeOpts{
	Name: "query_levenshtein_distance",
	Help: "Levenshtein distance of the query to the closest match",
})

var imageRequests = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "image_request",
		Help: "Dogbot image requests.",
	},
	[]string{"successful"},
)

type Dogbot struct {
	Conf             *Configuration
	Pg               *PostgresClient
	AvailableClasses map[string]string
}

// Initializes the map of class name -> class id
func (db *Dogbot) InitAvailableClasses() {
	db.AvailableClasses = make(map[string]string)
	available := db.Pg.GetAvailableClasses(db.Conf.MinimumClassConfidence)
	for _, class := range available {
		spl := strings.Split(class.ClassName, ", ")
		for _, s := range spl {
			db.AvailableClasses[strings.ToLower(s)] = class.ClassId
		}
	}
}

// Transforms user input into an available breed of dog
func (db *Dogbot) parseBreedQuery(query string) (string, int) {
	keys := reflect.ValueOf(db.AvailableClasses).MapKeys()
	breeds := make([]string, len(keys))
	for i := 0; i < len(keys); i++ {
		breeds[i] = keys[i].String()
	}
	query = strings.ToLower(query)
	// If it's an exact match, don't bother distance calculations
	if contains(breeds, query) {
		return query, 0
	}

	var bestGuess string
	minDist := 9999999

	// If inexact, find the breed with minimum edit distance
	for _, candidate := range breeds {
		distance := levenshtein(query, candidate)
		if distance < minDist {
			minDist = distance
			bestGuess = candidate
		}
	}
	return bestGuess, minDist
}

// Returns a complete URL for the input breed
// TODO: Should this return a ClassMember instead of (string, float64)?
func (db *Dogbot) GetRandomImageUrl(breed string) (string, float64) {
	class := db.AvailableClasses[breed]
	images := db.Pg.GetClassMembers(class, db.Conf.MinimumClassConfidence)
	idx := rand.Intn(len(images))
	mem := images[idx]
	url := db.Conf.StaticBaseURL + "v2/" + mem.Filename
	prob := mem.Probability
	return url, prob
}

// Starts the dogbot
func (db *Dogbot) Start() {
	rand.Seed(time.Now().UnixNano())

	// Register Prometheus stuff
	prometheus.MustRegister(levDists)
	prometheus.MustRegister(imageRequests)

	// Instrument Prometheus
	http.Handle("/metrics", prometheus.Handler())

	// Serve the images
	if db.Conf.RunImageServer {
		fmt.Println("Running image server")
		staticDir := db.Conf.RootDir + "static/"
		http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir))))
	}

	// We want to listen and serve no matter what, because we sill expose the metrics endpoint
	go http.ListenAndServe(db.Conf.Port, nil)

	// Connect to slack
	ws, id := slackConnect(db.Conf.Token)
	for {
		// read each incoming message
		m, err := getMessage(ws)
		if err != nil {
			log.Fatal(err)
		}

		// see if we're mentioned
		if m.Type == "message" && strings.HasPrefix(m.Text, "<@"+id+">") {
			go func(m Message) {
				// Strip @ id
				query := strings.Replace(m.Text, "<@"+id+"> ", "", -1)
				if query == "classes" {
					keys := reflect.ValueOf(db.AvailableClasses).MapKeys()
					breeds := make([]string, len(keys))
					for i := 0; i < len(keys); i++ {
						breeds[i] = keys[i].String()
					}
					m.Text = strings.Join(breeds, "\n")
				} else if query == "stats" {
					prob := FloatToString(db.Pg.GetAverageConfidence())
					m.Text = strconv.Itoa(db.Pg.GetImageCount()) + " images available" + "\n" + prob + " average confidence"
				} else {
					fmt.Println("Attempting to fetch photo for breed: " + query)
					breed, dist := db.parseBreedQuery(query)

					// Report a distance data point
					levDists.Add(float64(dist))

					if dist < 10 {
						url, probability := db.GetRandomImageUrl(breed)
						pStr := FloatToString(probability)
						msg := "My interpretation: " + breed + "\n" + "My confidence: " + pStr + "\n" + url
						m.Text = msg
						imageRequests.With(prometheus.Labels{"successful": "yes"}).Inc()
					} else {
						imageRequests.With(prometheus.Labels{"successful": "no"}).Inc()
						m.Text = "Sorry, I don't know that dog."
					}
				}
				postMessage(ws, m)
			}(m)
		}
	}
}

func FloatToString(input_num float64) string {
	// to convert a float number to a string
	return strconv.FormatFloat(input_num, 'f', 6, 64)
}

// Levenshtein distance between two strings
func levenshtein(a string, b string) int {

	// Handle empty string cases
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}

	// DP matrix
	mat := make([][]int, len(a))
	for i := range mat {
		mat[i] = make([]int, len(b))
	}

	// Initialize base cases
	for i := 0; i < len(a); i++ {
		mat[i][0] = i
	}
	for i := 0; i < len(b); i++ {
		mat[0][i] = i
	}

	// Fill out optimal edit distance matrix
	for i := 1; i < len(a); i++ {
		for j := 1; j < len(b); j++ {
			cost := 0
			if a[i] != b[j] {
				cost = 1
			}

			// Compute cheapest way of getting to this index
			above := mat[i-1][j] + 1
			left := mat[i][j-1] + 1
			diag := mat[i-1][j-1] + cost

			// Sort and take idx 0 to get minimum
			arr := []int{above, left, diag}
			sort.Ints(arr)
			min := arr[0]
			mat[i][j] = min
		}
	}
	return mat[len(a)-1][len(b)-1]
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func NewDogbot(conf *Configuration) *Dogbot {
	dogbot := new(Dogbot)
	dogbot.Conf = conf
	dogbot.Pg = NewPostgresClient(dogbot.Conf.PGHost, dogbot.Conf.PGPort,
		dogbot.Conf.PGUser, dogbot.Conf.PGPassword, dogbot.Conf.PGDbname)
	dogbot.InitAvailableClasses()
	return dogbot
}
