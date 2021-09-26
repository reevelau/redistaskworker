package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/go-redis/redis/v8"
)

var (
	Ctx context.Context = context.TODO()
)

func main() {
	redisAddrPtr := flag.String("redisAddr", "localhost:6379", "Redis address e.g. localhost:6379")
	controllerChannelNamePtr := flag.String("cname", "controller", "Controller Channel Name")
	flag.Parse()
	log.Printf("Going to connect: %s\n", *redisAddrPtr)
	log.Printf("Controller channel name %s\n", *controllerChannelNamePtr)

	hostname, err := os.Hostname()
	if err != nil {
		panic(err)
	}

	log.Printf("hostname: %s\n", hostname)

	rdb := redis.NewClient(&redis.Options{
		Addr:     *redisAddrPtr,
		Password: "",
		DB:       0,
	})

	pubsub := rdb.Subscribe(Ctx, hostname)

	_, err = pubsub.Receive(Ctx)

	if err != nil {
		panic(err)
	}

	rch := pubsub.Channel()

	exitCh := make(chan int, 1)

	go func() {
		var runningShellCmd *exec.Cmd = nil

		for msg := range rch {

			//chname := msg.Channel
			chpayload := msg.Payload

			//log.Printf("channel: %s\n", chname)
			//log.Printf("payload: %s\n", chpayload)
			payloadSlice := strings.Fields(chpayload)

			userCmdStr := payloadSlice[0]
			log.Printf("Receive user command %s\n", userCmdStr)
			switch userCmdStr {
			case "exec":

				var startNew = false
				if runningShellCmd == nil {
					startNew = true
				} else {
					if runningShellCmd.ProcessState == nil {
						log.Printf("Last user command is still running\n")
					} else {
						log.Printf("Last user command finished with state: %v\n", runningShellCmd.ProcessState)
						startNew = true
					}
				}

				if startNew {
					log.Printf("Starting new command\n")
					runningShellCmd = execCommand(payloadSlice[1:])
					runningShellCmd.Start()

					go func() {
						runningShellCmd.Wait()
					}()
				}

			case "status":
				var resp string = "statusresp " + hostname + " "
				if runningShellCmd == nil {
					log.Println("no process is running")
					resp += "initiated"
				} else {
					if runningShellCmd.ProcessState == nil {
						resp += "running"
						log.Println("Last user command is still running")
					} else {
						resp += fmt.Sprintf("%v", runningShellCmd.ProcessState)
						log.Printf("Last user command completed with %v\n", runningShellCmd.ProcessState)
					}
				}
				if err := rdb.Publish(Ctx, *controllerChannelNamePtr, resp).Err(); err != nil {
					log.Fatal(err)
				}
			case "kill":
				if runningShellCmd == nil {
					log.Println("nothing to kill")
				} else {
					if runningShellCmd.ProcessState == nil {
						//log.Println("Last user command is still running")
						err := runningShellCmd.Process.Kill()
						if err != nil {
							log.Printf("cannot kill process: %v\n", err)
						}
					} else {
						log.Printf("Last user command completed with %v\n", runningShellCmd.ProcessState)
					}
				}
			case "exit":
				log.Println("receive exit command")
				exitCh <- 0
			default:
				log.Printf("unknown command: %s\n", userCmdStr)
			}
		}
	}()

	ret := <-exitCh
	os.Exit(ret)
}

func execCommand(cmdStr []string) *exec.Cmd {
	cmd := exec.Command("/bin/sh", "-c", strings.Join(cmdStr, " "))

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		sout := bufio.NewScanner(stdout)
		serr := bufio.NewScanner(stderr)

		for sout.Scan() {
			log.Printf(sout.Text())
		}
		for serr.Scan() {
			log.Printf(serr.Text())
		}
	}()

	return cmd
}
