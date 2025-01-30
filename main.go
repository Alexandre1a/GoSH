package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/chzyer/readline"
)

func main() {
	// Chargement de l'historique au démarrage
	homeDir, _ := os.UserHomeDir()
	historyFile := homeDir + "/.gosh_history"

	// Configuration du shell interactif
	rl, err := readline.NewEx(&readline.Config{
		Prompt:       "> ",
		HistoryFile:  historyFile, // Permet de sauvegarder et charger l'historique
		AutoComplete: nil,         // Peut être amélioré avec l'autocomplétion
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "Erreur readline:", err)
		return
	}
	defer rl.Close()

	for {
		// Lecture de l'entrée utilisateur avec édition et historique
		input, err := rl.Readline()
		if err != nil { // EOF ou Ctrl+D
			break
		}

		// Suppression des espaces inutiles
		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		// Exécute la commande
		if err := execInput(input); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}
}

func execInput(input string) error {
	input = strings.TrimSuffix(input, "\n")
	args := strings.Split(input, " ")

	// Gérer les commandes intégrées
	switch args[0] {
	case "cd":
		if len(args) < 2 || args[1] == "" {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("Impossible de trouver le home")
			}
			return os.Chdir(homeDir)
		}
		return os.Chdir(args[1])
	case "exit":
		os.Exit(0)
	case "version":
		fmt.Println("GoShell Version 1.0.0")
		return nil
	}

	// Exécuter la commande système
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return cmd.Run()
}
