package main

import (
    "encoding/json"
    "io/ioutil"
    "log"
)

func ParseScripts () {
	//scripts = append(scripts, Script{name: "Ping", directory: "/home/sasan/Works/Karina/Mobazi/", format: "*/5 * * * * *", cmd: "php5", args: []string{"pingmobazi"}})
	//scripts = append(scripts, Script{name: "Test", directory: "/home/sasan/Works/Karina/Mobazi/", format: "0 * * * * *", cmd: "ls", args: []string{"-l", "-h"}})
    //var data []Script
    file, err := ioutil.ReadFile("test.json")
    if err != nil {
        log.Fatal(err)
    }
    err = json.Unmarshal(file, &config)
    if err != nil {
        log.Fatal(err)
    }
}
