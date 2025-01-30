package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func main() {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("> ")
		// Read the keyboard input.
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}

		// Handle the execution of the input.
		if err = execInput(input); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}
}

func execInput(input string) error {
	// Remove the newline character.
	input = strings.TrimSuffix(input, "\n")

	// Split the input to separate the command and the arguments.
	args := strings.Split(input, " ")

	// Sauvegarde l'historique avant de traiter la commande
	checkHistoryAndWrite(input)

	// Check for built-in commands.
	switch args[0] {
	case "cd":
		if len(args) < 2 || args[1] == "" {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return errors.New("Unable to get home directory")
			}
			return os.Chdir(homeDir)
		}
		return os.Chdir(args[1])
	case "exit":
		os.Exit(0)
	case "version":
		println("GoShell Version 0.2.0")
		return nil
	}

	// Pass the program and the arguments separately.
	cmd := exec.Command(args[0], args[1:]...)

	// Set the correct output device.
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	// Execute the command and return the error.
	return cmd.Run()
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func checkHistoryAndWrite(text string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return errors.New("Unable to get home directory")
	}
	filePath := homeDir + "/.gosh_history"

	// Ouvre le fichier en mode append et avec les bons droits
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("Can't open history file: %v", err)
	}
	defer file.Close()

	_, err = file.WriteString(text + "\n")
	if err != nil {
		return fmt.Errorf("Can't write to history: %v", err)
	}
	return nil
}
