package main

import "encoding/json"
import "fmt"
import "os"

type Configuration struct {
	Token         string
	RootDir       string
	Port          string
	StaticBaseURL string
	PGHost        string
	PGPort        int
	PGUser        string
	PGPassword    string
	PGDbname      string
}

func main() {
	args := os.Args[1:]
	if len(args) != 1 {
		fmt.Println("Usage: ./main <absolute path to configuration file>")
		return
	}
	file, _ := os.Open(args[0])
	decoder := json.NewDecoder(file)
	var conf = Configuration{}
	err := decoder.Decode(&conf)
	if err != nil {
		fmt.Println("error:", err)
	}
	dogbot := NewDogbot(&conf)
	dogbot.Start()
}
