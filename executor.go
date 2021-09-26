package main

import (
	"bufio"
	"log"
	"os/exec"
)

func main() {
	log.Println("Hello")

	cmd := exec.Command("sleep", "5")

	//cmd.Stdout = os.Stdout
	//cmd.Stderr = os.Stderr

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("cmd.ProcessState %v\n", cmd.ProcessState)

	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}
	log.Printf("cmd.ProcessState %v\n", cmd.ProcessState)

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

	log.Printf("Waiting for command to finished...")
	if err := cmd.Wait(); err != nil {
		log.Fatal(err)
	}

	log.Printf("cmd.ProcessState %v\n", cmd.ProcessState)

	log.Println("Command finished")

}
