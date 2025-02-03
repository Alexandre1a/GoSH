package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/chzyer/readline"
	"github.com/creack/pty"
	"github.com/google/shlex"
	"github.com/spf13/viper"
)

// Structure pour stocker la configuration
type Config struct {
	Prompt      string `mapstructure:"prompt"`
	Color       string `mapstructure:"color"`
	HistorySize int    `mapstructure:"history_size"`
}

var config Config

func main() {
	// Chargement de la configuration
	loadConfig()

	// Chargement de l'historique au démarrage
	homeDir, _ := os.UserHomeDir()
	historyFile := homeDir + "/.gosh_history"

	// Configuration du shell interactif
	rl, err := readline.NewEx(&readline.Config{
		Prompt:       getPrompt(), // Utilise une fonction pour générer le prompt dynamiquement
		HistoryFile:  historyFile, // Permet de sauvegarder et charger l'historique
		HistoryLimit: config.HistorySize,
		AutoComplete: nil, // Peut être amélioré avec l'autocomplétion
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "Erreur readline:", err)
		return
	}
	defer rl.Close()

	for {
		// Mettre à jour le prompt avec le répertoire courant
		rl.SetPrompt(getPrompt())

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

// Charger la configuration depuis un fichier
func loadConfig() {
	homeDir, _ := os.UserHomeDir()
	configPath := homeDir + "/.config/gosh/gosh_config.toml"
	fmt.Println("Chemin du fichier de configuration:", configPath)
	viper.SetConfigFile(configPath)

	// Valeurs par défaut
	viper.SetDefault("prompt", "[{dir}] > ")
	viper.SetDefault("color", "blue")
	viper.SetDefault("history_size", 1000)

	// Lire le fichier de configuration
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Si le fichier n'existe pas, le créer avec les valeurs par défaut
			fmt.Println("Création du fichier de configuration avec les valeurs par défaut...")
			if err := viper.WriteConfigAs(configPath); err != nil {
				fmt.Fprintln(os.Stderr, "Erreur lors de la création du fichier de configuration:", err)
			} else {
				fmt.Println("Fichier de configuration créé avec succès:", configPath)
			}
		} else {
			// Autre erreur de lecture du fichier
			fmt.Fprintln(os.Stderr, "Erreur de configuration:", err)
			fmt.Println("Utilisation des valeurs par défaut.")
		}
	}

	// Charger la configuration dans la structure Config
	if err := viper.Unmarshal(&config); err != nil {
		fmt.Fprintln(os.Stderr, "Erreur de chargement de la configuration:", err)
		fmt.Println("Utilisation des valeurs par défaut.")
	}

	// Validation des valeurs
	if config.HistorySize <= 0 {
		fmt.Fprintln(os.Stderr, "Taille de l'historique invalide. Utilisation de la valeur par défaut (1000).")
		config.HistorySize = 1000
	}
}

// Fonction pour générer le prompt avec le répertoire courant
func getPrompt() string {
	wd, err := os.Getwd()
	if err != nil {
		wd = "?"
	}

	// Remplacer le chemin du home par "~"
	homeDir, _ := os.UserHomeDir()
	if homeDir != "" && strings.HasPrefix(wd, homeDir) {
		wd = "~" + strings.TrimPrefix(wd, homeDir)
	}

	// Utiliser le prompt défini dans la configuration
	prompt := strings.Replace(config.Prompt, "{dir}", wd, -1)

	// Ajouter de la couleur si configuré
	if config.Color == "blue" {
		blue := "\033[34m"
		reset := "\033[0m"
		prompt = blue + prompt + reset
	}

	return prompt
}

func execInput(input string) error {
	input = strings.TrimSuffix(input, "\n")

	// Diviser la commande en arguments en respectant les guillemets
	args, err := shlex.Split(input)
	if err != nil {
		return fmt.Errorf("Erreur lors de la division des arguments: %v", err)
	}

	if len(args) == 0 {
		return nil
	}

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
		fmt.Println("GoShell Version 2.1.2")
		return nil
	case "set":
		if len(args) < 3 {
			return fmt.Errorf("Usage: set <key> <value>")
		}
		return setConfig(args[1], strings.Join(args[2:], " "))
	}

	// Exécuter la commande système dans un PTY
	cmd := exec.Command(args[0], args[1:]...)
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return fmt.Errorf("Erreur lors du démarrage du PTY: %v", err)
	}
	defer ptmx.Close()

	// Rediriger les entrées/sorties entre le terminal parent et le PTY
	go func() {
		io.Copy(ptmx, os.Stdin) // Rediriger stdin vers le PTY
	}()
	io.Copy(os.Stdout, ptmx) // Rediriger stdout du PTY vers le terminal

	// Attendre la fin de la commande
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("Erreur lors de l'exécution de la commande: %v", err)
	}

	return nil
}

// Fonction pour modifier la configuration à la volée
func setConfig(key, value string) error {
	switch key {
	case "prompt":
		viper.Set("prompt", value)
	case "color":
		viper.Set("color", value)
	case "history_size":
		intValue, err := strconv.Atoi(value)
		if err != nil || intValue <= 0 {
			return fmt.Errorf("history_size doit être un entier positif")
		}
		viper.Set("history_size", intValue)
	default:
		return fmt.Errorf("Clé de configuration inconnue: %s", key)
	}

	// Sauvegarder la configuration dans le fichier
	if err := viper.WriteConfig(); err != nil {
		return fmt.Errorf("Erreur lors de la sauvegarde de la configuration: %v", err)
	}

	// Recharger la configuration
	if err := viper.Unmarshal(&config); err != nil {
		return fmt.Errorf("Erreur lors du rechargement de la configuration: %v", err)
	}

	fmt.Printf("Configuration mise à jour: %s = %s\n", key, value)
	return nil
}
