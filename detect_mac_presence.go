package main

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"

	"github.com/dchest/safefile"
)

type Person struct {
	Name    string `json:"name"`
	MacAddr string `json:"macaddr"`
	AppId   string `json:"appid"`
	Token   string `json:"token"`
	Changed bool   `json:"changed"`
	Checked bool   `json:"checked"`
	AtHome  bool   `json:"athome"`
}

type SystemState struct {
	BaseURL string   `json:"baseurl"`
	Changed bool     `json:"changed"`
	People  []Person `json:"people"`
}

func BuildSystemState(filename string, baseURL string) SystemState {
	csvFile, _ := os.Open(filename)
	reader := csv.NewReader(bufio.NewReader(csvFile))
	var people []Person
	for {
		line, error := reader.Read()
		if error == io.EOF {
			break
		} else if error != nil {
			log.Fatal(error)
		}
		people = append(people, Person{
			Name:    line[0],
			MacAddr: line[1],
			AppId:   line[2],
			Token:   line[3],
		})
	}
	state := SystemState{
		BaseURL: baseURL,
		Changed: true, // because loading file changes state
		People:  people,
	}
	return state
}

func ResetPeopleState(state SystemState) {
	for i := 0; i < len(state.People); i++ {
		person := &state.People[i]
		person.Checked = false
		person.Changed = false
	}
}

func LookForPeople(state SystemState) SystemState {
	// run arp command
	cmd := exec.Command("arp", "-an")
	cmdReader, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error creating StdoutPipe for Cmd", err)
		os.Exit(1)
	}

	// run and process the command
	err = cmd.Start()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error starting Cmd", err)
		os.Exit(1)
	}

	// find any matched mac addresses in arp output
	scanner := bufio.NewScanner(cmdReader)
	for scanner.Scan() {
		line := scanner.Text()
		for i := 0; i < len(state.People); i++ {
			person := &state.People[i]
			matched, _ := regexp.MatchString(person.MacAddr, line)
			if matched {
				person.Checked = true
				if person.AtHome {
					// nothing to do here...
				} else {
					person.AtHome = true
					person.Changed = true
					state.Changed = true
				}
			}
		}
		// fmt.Fprintln(os.Stderr, ">>> ", line)
	}

	err = cmd.Wait()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error waiting for Cmd", err)
		os.Exit(1)
	}

	// now find any people who were at home but not matched...
	for i := 0; i < len(state.People); i++ {
		person := &state.People[i]
		if person.AtHome {
			if !person.Checked {
				// they were at home but are no longer seen - reset
				person.AtHome = false
				person.Changed = true
				state.Changed = true
			}
		}
	}

	return state
}

func WriteState(filename string, state SystemState) {
	stateJson, _ := json.Marshal(state)
	fmt.Fprintln(os.Stderr, "Writing state file ", filename)
	err := safefile.WriteFile(filename, []byte(string(stateJson)), 0600)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error writing state file", err)
		os.Exit(1)
	}
}

func ReadState(filename string) SystemState {
	// Open our jsonFile
	jsonFile, err := os.Open(filename)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error reading state file", err)
		os.Exit(1)
	}

	// read our opened xmlFile as a byte array.
	byteValue, _ := ioutil.ReadAll(jsonFile)

	// we initialize our Users array
	var state SystemState

	// we unmarshal our byteArray which contains our
	// jsonFile's content into 'users' which we defined above
	json.Unmarshal(byteValue, &state)

	// A state loaded from disk is de facto unchanged
	state.Changed = false

	// return the state
	return state
}

func updateSmartThingsState(state SystemState, force bool) {
	if state.Changed || force {
		for i := 0; i < len(state.People); i++ {
			person := state.People[i]
			personState := "away"
			if person.AtHome {
				personState = "home"
			}
			notificationURL := fmt.Sprintf("%s/api/smartapps/installations/%s/Phone/%s?access_token=%s", state.BaseURL, person.AppId, personState, person.Token)
			res, err := http.Get(notificationURL)
			if err != nil {
				log.Fatal(err)
			}
			response, err := ioutil.ReadAll(res.Body)
			res.Body.Close()
			if err != nil {
				log.Fatal(err)
			}
			fmt.Printf("%s", response)
		}
	}
}

func main() {
	usr, _ := user.Current()
	defaultStateFile := filepath.Join(usr.HomeDir, ".presence.json")
	defaultBaseURL := "https://graph-eu01-euwest1.api.smartthings.com"

	loadFilePtr := flag.String("load", "", "CSV file to load config")
	stateFilePtr := flag.String("state", defaultStateFile, "JSON state file location")
	baseURLPtr := flag.String("baseurl", defaultBaseURL, "SmartThings API Base URL")
	forcePtr := flag.Bool("force", false, "Force SmartThings state update")
	flag.Parse()

	var state SystemState
	if len(*loadFilePtr) > 0 {
		state = BuildSystemState(*loadFilePtr, *baseURLPtr)
	} else {
		state = ReadState(*stateFilePtr)
	}
	ResetPeopleState(state)
	state = LookForPeople(state)
	updateSmartThingsState(state, *forcePtr)
	if state.Changed {
		WriteState(*stateFilePtr, state)
	}
}
