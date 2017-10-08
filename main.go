package main

import "fmt"
import "encoding/json"
import "os"
import "log"
import "strings"

type Configuration struct {
	Token string
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

func main() {
	initConf()

	// start a websocket-based Real Time API session
	ws, id := slackConnect(Conf.Token)

	for {
		// read each incoming message
		m, err := getMessage(ws)
		if err != nil {
			log.Fatal(err)
		}

		// see if we're mentioned
		if m.Type == "message" && strings.HasPrefix(m.Text, "<@"+id+">") {
			fmt.Println(m)
			// parts := strings.Fields(m.Text)
			go func(m Message) {
				m.Text = "hello"
				postMessage(ws, m)
			}(m)
		}
	}
}
