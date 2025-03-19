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

// Map des couleurs ANSI
var colors = map[string]string{
	"black":  "\033[30m",
	"red":    "\033[31m",
	"green":  "\033[32m",
	"yellow": "\033[33m",
	"blue":   "\033[34m",
	"purple": "\033[35m",
	"cyan":   "\033[36m",
	"white":  "\033[37m",
	"reset":  "\033[0m",
}

func main() {
	// Chargement de la configuration
	loadConfig()

	// Chargement de l'historique au démarrage
	homeDir, _ := os.UserHomeDir()
	historyFile := homeDir + "/.gosh_history"

	// Configuration du shell interactif
	rl, err := readline.NewEx(&readline.Config{
		Prompt:       getPrompt(),
		HistoryFile:  historyFile,
		HistoryLimit: config.HistorySize,
		AutoComplete: nil,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "Erreur readline:", err)
		return
	}
	defer rl.Close()

	for {
		rl.SetPrompt(getPrompt())

		input, err := rl.Readline()
		if err != nil {
			break
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		if err := execInput(input); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}
}

// Charger la configuration depuis un fichier
func loadConfig() {
	homeDir, _ := os.UserHomeDir()
	configPath := homeDir + "/.config/gosh/"
	fmt.Println("Chemin du fichier de configuration:", configPath)
	viper.AddConfigPath(configPath)
	viper.SetConfigName("gosh_config")
	viper.SetConfigType("toml")

	// Valeurs par défaut
	viper.SetDefault("prompt", "[{dir}] $ ")
	viper.SetDefault("color", "green")
	viper.SetDefault("history_size", 1000)

	// Lire le fichier de configuration
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			fmt.Println("Création du fichier de configuration avec les valeurs par défaut...")
			if err := viper.WriteConfigAs(configPath + "gosh_config"); err != nil {
				fmt.Fprintln(os.Stderr, "Erreur lors de la création du fichier de configuration:", err)
			} else {
				fmt.Println("Fichier de configuration créé avec succès:", configPath)
			}
		} else {
			fmt.Fprintln(os.Stderr, "Erreur de configuration:", err)
			fmt.Println("Utilisation des valeurs par défaut.")
		}
	}

	if err := viper.Unmarshal(&config); err != nil {
		fmt.Fprintln(os.Stderr, "Erreur de chargement de la configuration:", err)
		fmt.Println("Utilisation des valeurs par défaut.")
	}

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

	homeDir, _ := os.UserHomeDir()
	if homeDir != "" && strings.HasPrefix(wd, homeDir) {
		wd = "~" + strings.TrimPrefix(wd, homeDir)
	}

	prompt := strings.Replace(config.Prompt, "{dir}", wd, -1)

	// Appliquer la couleur si elle existe dans la map
	if colorCode, exists := colors[config.Color]; exists {
		prompt = colorCode + prompt + colors["reset"]
	}

	return prompt
}

// Fonction pour déterminer si une commande est interactive
func isInteractiveCommand(cmd string) bool {
	interactiveCommands := map[string]bool{
		"vim":  true,
		"nano": true,
		"ssh":  true,
		"top":  true,
		"htop": true,
		"less": true,
		"more": true,
	}
	return interactiveCommands[cmd]
}

func execInput(input string) error {
	input = strings.TrimSuffix(input, "\n")

	args, err := shlex.Split(input)
	if err != nil {
		return fmt.Errorf("Erreur lors de la division des arguments: %v", err)
	}

	if len(args) == 0 {
		return nil
	}

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

	cmd := exec.Command(args[0], args[1:]...)

	if isInteractiveCommand(args[0]) {
		// Utiliser un PTY pour les commandes interactives
		ptmx, err := pty.Start(cmd)
		if err != nil {
			return fmt.Errorf("Erreur lors du démarrage du PTY: %v", err)
		}
		defer ptmx.Close()

		// Rediriger stdin et stdout
		go func() {
			io.Copy(ptmx, os.Stdin)
		}()
		io.Copy(os.Stdout, ptmx)
	} else {
		// Exécuter directement les commandes non interactives
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin

		// Démarrer la commande avant d'attendre sa fin
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("Erreur lors du démarrage de la commande: %v", err)
		}
	}

	// Attendre la fin du processus
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
		if _, exists := colors[value]; !exists {
			return fmt.Errorf("Couleur inconnue: %s. Couleurs disponibles: %v", value, getAvailableColors())
		}
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

	if err := viper.WriteConfig(); err != nil {
		return fmt.Errorf("Erreur lors de la sauvegarde de la configuration: %v", err)
	}

	if err := viper.Unmarshal(&config); err != nil {
		return fmt.Errorf("Erreur lors du rechargement de la configuration: %v", err)
	}

	fmt.Printf("Configuration mise à jour: %s = %s\n", key, value)
	return nil
}

// Retourne la liste des couleurs disponibles
func getAvailableColors() []string {
	keys := make([]string, 0, len(colors))
	for k := range colors {
		keys = append(keys, k)
	}
	return keys
}
