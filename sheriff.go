package main

import (
	"fmt"
	"github.com/fzzy/radix/redis"
	"github.com/robfig/cron"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"time"
)

type Script struct {
	name      string
	directory string
	format    string
	cmd       string
	args      []string
}

var (
	scripts []Script
)

func ErrorHandler(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(0)
	}
}

func main() {
	fmt.Println("Connecting to redis server...")
	RedisClient, err := redis.Dial("tcp", "127.0.0.1:6379")
	ErrorHandler(err)

	defer RedisClient.Close()

	scripts = append(scripts, Script{name: "Ping", directory: "/home/sasan/Works/Karina/Mobazi/", format: "*/5 * * * * *", cmd: "php5", args: []string{"pingmobazi"}})
	scripts = append(scripts, Script{name: "Test", directory: "/home/sasan/Works/Karina/Mobazi/", format: "0 * * * * *", cmd: "ls", args: []string{"-l", "-h"}})

	fmt.Println("Adding scripts routines...")

	var wg sync.WaitGroup

	cr := cron.New()
	for _, script := range scripts {
		wg.Add(1)
		script := script
		cr.AddFunc(script.format, func() {

			cmd := exec.Command(script.cmd, script.args...)

			cmd.Dir = script.directory

			output, err := cmd.Output()

			if err != nil {
				fmt.Printf("Error calling %s: %v\n", script.cmd, err)
			} else {
				output := string(output[:])
				var value float64
				value, err := strconv.ParseFloat(output, 64)
				if err != nil {
					fmt.Printf("Error parsing value %s: %v\n", script.cmd, err)
				} else {
					t := time.Now().Unix()
					RedisClient.Cmd("zAdd", "sheriff:"+script.name, float64(t), strconv.FormatInt(t, 10)+":"+strconv.FormatFloat(value, 'f', -1, 64))
				}
			}
		})

		fmt.Printf("Script %v added\n", script.name)
	}

	cr.Start()

	wg.Wait()
}
