package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/chzyer/readline"
	"github.com/creack/pty"
	"github.com/google/shlex"
	"github.com/spf13/viper"
)

// Structure pour stocker la configuration
type Config struct {
	Prompt      string            `mapstructure:"prompt"`
	Color       string            `mapstructure:"color"`
	HistorySize int               `mapstructure:"history_size"`
	Aliases     map[string]string `mapstructure:"aliases"`
}

// Constantes pour la version et le nom du shell
const (
	VERSION    = "2.2.0"
	SHELL_NAME = "GoShell"
)

var (
	config Config
	// Map des couleurs ANSI
	colors = map[string]string{
		"black":  "\033[30m",
		"red":    "\033[31m",
		"green":  "\033[32m",
		"yellow": "\033[33m",
		"blue":   "\033[34m",
		"purple": "\033[35m",
		"cyan":   "\033[36m",
		"white":  "\033[37m",
		"reset":  "\033[0m",
		// Ajout de couleurs supplémentaires
		"bold":      "\033[1m",
		"underline": "\033[4m",
	}
)

func main() {
	// Gestion des signaux
	setupSignalHandling()

	// Chargement de la configuration
	loadConfig()

	// Chargement de l'historique au démarrage
	homeDir, _ := os.UserHomeDir()
	historyFile := filepath.Join(homeDir, ".gosh_history")

	// Configuration du shell interactif
	rl, err := readline.NewEx(&readline.Config{
		Prompt:              getPrompt(),
		HistoryFile:         historyFile,
		HistoryLimit:        config.HistorySize,
		AutoComplete:        newCompleter(),
		InterruptPrompt:     "^C",
		EOFPrompt:           "exit",
		HistorySearchFold:   true,
		FuncFilterInputRune: nil,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "Erreur readline:", err)
		return
	}
	defer rl.Close()

	fmt.Printf("Bienvenue dans %s version %s\nTapez 'help' pour afficher l'aide.\n\n", SHELL_NAME, VERSION)

	for {
		rl.SetPrompt(getPrompt())

		input, err := rl.Readline()
		if err != nil {
			if err == readline.ErrInterrupt {
				continue // Ignorer Ctrl+C
			} else if err == io.EOF {
				break // Ctrl+D pour quitter
			}
			fmt.Fprintln(os.Stderr, "Erreur de lecture:", err)
			break
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		startTime := time.Now()
		if err := execInput(input); err != nil {
			fmt.Fprintln(os.Stderr, "Erreur:", err)
		}
		duration := time.Since(startTime)

		// Afficher le temps d'exécution pour les commandes qui prennent plus d'une seconde
		if duration > time.Second {
			fmt.Printf("Temps d'exécution: %s\n", duration.Round(time.Millisecond))
		}
	}
}

// Mise en place de la gestion des signaux
func setupSignalHandling() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("\nAu revoir!")
		os.Exit(0)
	}()
}

// Implémentation simple de l'auto-complétion
func newCompleter() *readline.PrefixCompleter {
	// Liste des commandes internes
	internalCommands := []readline.PrefixCompleterInterface{
		readline.PcItem("cd"),
		readline.PcItem("exit"),
		readline.PcItem("help"),
		readline.PcItem("version"),
		readline.PcItem("set",
			readline.PcItem("prompt"),
			readline.PcItem("color",
				readline.PcItem("black"),
				readline.PcItem("red"),
				readline.PcItem("green"),
				readline.PcItem("yellow"),
				readline.PcItem("blue"),
				readline.PcItem("purple"),
				readline.PcItem("cyan"),
				readline.PcItem("white"),
				readline.PcItem("bold"),
				readline.PcItem("underline"),
			),
			readline.PcItem("history_size"),
		),
		readline.PcItem("alias"),
		readline.PcItem("unalias"),
		readline.PcItem("aliases"),
	}

	return readline.NewPrefixCompleter(internalCommands...)
}

// Charger la configuration depuis un fichier
func loadConfig() {
	homeDir, _ := os.UserHomeDir()
	configPath := filepath.Join(homeDir, ".config", "gosh")

	if err := os.MkdirAll(configPath, 0755); err != nil {
		fmt.Fprintln(os.Stderr, "Erreur lors de la création du dossier de configuration:", err)
	}

	viper.AddConfigPath(configPath)
	viper.SetConfigName("gosh_config")
	viper.SetConfigType("toml")

	// Valeurs par défaut
	viper.SetDefault("prompt", "[{dir}] $ ")
	viper.SetDefault("color", "green")
	viper.SetDefault("history_size", 1000)
	viper.SetDefault("aliases", map[string]string{
		"ll": "ls -la",
		"la": "ls -a",
	})

	// Lire le fichier de configuration
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			fmt.Println("Création du fichier de configuration avec les valeurs par défaut...")
			configFilePath := filepath.Join(configPath, "gosh_config.toml")
			if err := viper.WriteConfigAs(configFilePath); err != nil {
				fmt.Fprintln(os.Stderr, "Erreur lors de la création du fichier de configuration:", err)
			} else {
				fmt.Println("Fichier de configuration créé avec succès:", configFilePath)
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

	// Initialiser la map des alias si elle est nil
	if config.Aliases == nil {
		config.Aliases = make(map[string]string)
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

	// Remplacer des variables dans le prompt
	prompt := strings.Replace(config.Prompt, "{dir}", wd, -1)
	prompt = strings.Replace(prompt, "{time}", time.Now().Format("15:04:05"), -1)
	prompt = strings.Replace(prompt, "{date}", time.Now().Format("2006-01-02"), -1)
	prompt = strings.Replace(prompt, "{shell}", SHELL_NAME, -1)
	prompt = strings.Replace(prompt, "{version}", VERSION, -1)

	// Appliquer la couleur si elle existe dans la map
	if colorCode, exists := colors[config.Color]; exists {
		prompt = colorCode + prompt + colors["reset"]
	}

	return prompt
}

// Fonction pour déterminer si une commande est interactive
func isInteractiveCommand(cmd string) bool {
	interactiveCommands := map[string]bool{
		"vim":   true,
		"nano":  true,
		"emacs": true,
		"ssh":   true,
		"top":   true,
		"htop":  true,
		"less":  true,
		"more":  true,
		"man":   true,
		"vi":    true,
		"pico":  true,
	}
	return interactiveCommands[cmd]
}

// Remplacer les alias par leurs commandes correspondantes
func replaceAliases(input string) string {
	args, err := shlex.Split(input)
	if err != nil || len(args) == 0 {
		return input
	}

	// Si le premier mot est un alias, le remplacer
	if aliasCmd, exists := config.Aliases[args[0]]; exists {
		// Si l'alias contient des arguments, les combiner avec ceux de la commande
		aliasArgs, err := shlex.Split(aliasCmd)
		if err != nil {
			return input
		}

		if len(args) > 1 {
			// Joindre les arguments de l'alias avec ceux de la commande
			return strings.Join(append(aliasArgs, args[1:]...), " ")
		}
		return aliasCmd
	}

	return input
}

func execInput(input string) error {
	// Remplacer les alias
	expandedInput := replaceAliases(input)
	if expandedInput != input {
		fmt.Printf("Alias expanded: %s\n", expandedInput)
		input = expandedInput
	}

	args, err := shlex.Split(input)
	if err != nil {
		return fmt.Errorf("erreur lors de la division des arguments: %v", err)
	}

	if len(args) == 0 {
		return nil
	}

	switch args[0] {
	case "cd":
		// Gestion du changement de répertoire
		if len(args) < 2 || args[1] == "" {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("impossible de trouver le home")
			}
			return os.Chdir(homeDir)
		}

		// Expansion du tilde en chemin complet du répertoire utilisateur
		if args[1] == "~" || strings.HasPrefix(args[1], "~/") {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("impossible de trouver le home: %v", err)
			}
			args[1] = strings.Replace(args[1], "~", homeDir, 1)
		}

		if err := os.Chdir(args[1]); err != nil {
			return fmt.Errorf("cd: %v", err)
		}
		return nil

	case "exit":
		fmt.Println("Au revoir!")
		os.Exit(0)

	case "version":
		fmt.Printf("%s Version %s\n", SHELL_NAME, VERSION)
		return nil

	case "help":
		printHelp()
		return nil

	case "set":
		if len(args) < 2 {
			// Afficher la configuration actuelle
			fmt.Printf("Configuration actuelle:\n")
			fmt.Printf("  prompt = %s\n", config.Prompt)
			fmt.Printf("  color = %s\n", config.Color)
			fmt.Printf("  history_size = %d\n", config.HistorySize)
			return nil
		}
		if len(args) < 3 {
			return fmt.Errorf("usage: set <key> <value>")
		}
		return setConfig(args[1], strings.Join(args[2:], " "))

	case "alias":
		if len(args) == 1 {
			// Afficher tous les alias
			return listAliases()
		}
		if len(args) < 3 {
			return fmt.Errorf("usage: alias <name> <command>")
		}
		return addAlias(args[1], strings.Join(args[2:], " "))

	case "unalias":
		if len(args) != 2 {
			return fmt.Errorf("usage: unalias <name>")
		}
		return removeAlias(args[1])

	case "aliases":
		return listAliases()
	}

	// Exécution des commandes externes
	cmd := exec.Command(args[0], args[1:]...)

	if isInteractiveCommand(args[0]) {
		// Utiliser un PTY pour les commandes interactives
		ptmx, err := pty.Start(cmd)
		if err != nil {
			return fmt.Errorf("erreur lors du démarrage du PTY: %v", err)
		}
		defer ptmx.Close()

		// Gérer le redimensionnement du terminal
		go func() {
			// TODO: Implémenter la gestion du redimensionnement
		}()

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

		// Démarrer la commande
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("erreur lors du démarrage de la commande: %v", err)
		}
	}

	// Attendre la fin du processus
	if err := cmd.Wait(); err != nil {
		// Vérifier si l'erreur est due à un code de sortie non nul
		if exitErr, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("la commande a échoué avec le code: %d", exitErr.ExitCode())
		}
		return fmt.Errorf("erreur lors de l'exécution de la commande: %v", err)
	}

	return nil
}

// Afficher l'aide du shell
func printHelp() {
	fmt.Printf("%s v%s - Un shell léger écrit en Go\n\n", SHELL_NAME, VERSION)
	fmt.Println("Commandes internes:")
	fmt.Println("  cd [dir]           - Changer de répertoire")
	fmt.Println("  exit               - Quitter le shell")
	fmt.Println("  version            - Afficher la version du shell")
	fmt.Println("  help               - Afficher cette aide")
	fmt.Println("  set [key] [value]  - Afficher ou modifier la configuration")
	fmt.Println("  alias <nom> <cmd>  - Créer un alias pour une commande")
	fmt.Println("  unalias <nom>      - Supprimer un alias")
	fmt.Println("  aliases            - Lister tous les alias")
	fmt.Println()
	fmt.Println("Variables de prompt:")
	fmt.Println("  {dir}     - Répertoire courant")
	fmt.Println("  {time}    - Heure actuelle")
	fmt.Println("  {date}    - Date actuelle")
	fmt.Println("  {shell}   - Nom du shell")
	fmt.Println("  {version} - Version du shell")
	fmt.Println()
	fmt.Println("Couleurs disponibles:")
	fmt.Print("  ")
	for color := range colors {
		if color != "reset" {
			fmt.Printf("%s ", color)
		}
	}
	fmt.Println()
}

// Fonction pour modifier la configuration à la volée
func setConfig(key, value string) error {
	switch key {
	case "prompt":
		viper.Set("prompt", value)
	case "color":
		if _, exists := colors[value]; !exists {
			return fmt.Errorf("couleur inconnue: %s. Couleurs disponibles: %v", value, getAvailableColors())
		}
		viper.Set("color", value)
	case "history_size":
		intValue, err := strconv.Atoi(value)
		if err != nil || intValue <= 0 {
			return fmt.Errorf("history_size doit être un entier positif")
		}
		viper.Set("history_size", intValue)
	default:
		return fmt.Errorf("clé de configuration inconnue: %s", key)
	}

	if err := viper.WriteConfig(); err != nil {
		return fmt.Errorf("erreur lors de la sauvegarde de la configuration: %v", err)
	}

	if err := viper.Unmarshal(&config); err != nil {
		return fmt.Errorf("erreur lors du rechargement de la configuration: %v", err)
	}

	fmt.Printf("Configuration mise à jour: %s = %s\n", key, value)
	return nil
}

// Ajouter un alias
func addAlias(name, command string) error {
	if name == "" || command == "" {
		return fmt.Errorf("le nom et la commande ne peuvent pas être vides")
	}

	// Vérifier que le nom n'est pas un mot clé réservé
	reservedCommands := map[string]bool{
		"cd":      true,
		"exit":    true,
		"version": true,
		"help":    true,
		"set":     true,
		"alias":   true,
		"unalias": true,
		"aliases": true,
	}

	if reservedCommands[name] {
		return fmt.Errorf("impossible de créer un alias avec un nom réservé: %s", name)
	}

	// Ajouter ou mettre à jour l'alias
	if config.Aliases == nil {
		config.Aliases = make(map[string]string)
	}
	config.Aliases[name] = command
	viper.Set("aliases", config.Aliases)

	if err := viper.WriteConfig(); err != nil {
		return fmt.Errorf("erreur lors de la sauvegarde des alias: %v", err)
	}

	fmt.Printf("Alias ajouté: %s = %s\n", name, command)
	return nil
}

// Supprimer un alias
func removeAlias(name string) error {
	if config.Aliases == nil || config.Aliases[name] == "" {
		return fmt.Errorf("alias non trouvé: %s", name)
	}

	delete(config.Aliases, name)
	viper.Set("aliases", config.Aliases)

	if err := viper.WriteConfig(); err != nil {
		return fmt.Errorf("erreur lors de la sauvegarde des alias: %v", err)
	}

	fmt.Printf("Alias supprimé: %s\n", name)
	return nil
}

// Lister tous les alias
func listAliases() error {
	if config.Aliases == nil || len(config.Aliases) == 0 {
		fmt.Println("Aucun alias défini.")
		return nil
	}

	fmt.Println("Aliases définis:")
	for name, command := range config.Aliases {
		fmt.Printf("  %s = %s\n", name, command)
	}
	return nil
}

// Retourne la liste des couleurs disponibles
func getAvailableColors() []string {
	keys := make([]string, 0, len(colors))
	for k := range colors {
		if k != "reset" {
			keys = append(keys, k)
		}
	}
	return keys
}
