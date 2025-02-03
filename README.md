# GoSh!
A Shell made in Go, for fun  
## Features
- Display Working Directory
- History file wich you can browse for past commands
- Some colors
- Config file
- PTY support for interactive commands
I'm planning to add more features later (like syntax hilighting, etc...).  
I'm also learning Go by doing this project.  
You can expect breaking changes (or code), the problem has a workaround in the commit message.  
If not, the issue will be fixed soon (in case of a typo for example)  

## Installation
You can grab the binaries for your system and architecture, or build it yourself
To build it, clone the repository, cd into it and run  
`go build -o bin/GoSH`  
To test the shell and see if it suits you.  
If everything works, then move the binary to a place inside your path.  

To directly install it with the Go toolchain, just use  
`go intall`  
This will build and place the binary inside your `$HOME/go/bin` folder.  
Add this folder to your path and you are good to go !  

## Usage
To use the program, just invoke it with `GoSH`  
If you see a message about a config file, create `~/.config/gosh/gosh_config.toml` and populate it with the defaults written inside this repo -> [here](/defaults.toml).  
To change config parameter on the fly, use the `set` builtin.  
Currently, `set` has a limited amount of configuration options and need to have a valid config file to write to.  
To change the color of the prompt use `set color <color>`  
You can change the history size with `set history_size <int>`  
You can change the prompt with `set prompt <promp>`  
Here is some exemple of prompts :  
  - `[{dir}] > `
  - `["Hello World"] $ `
  - `"Hello" `  
You can use all "console colors", listed [here](https://gist.github.com/kamito/704813)  

## Know Issues
Currently there is a number of known or unkwown issues.  
We can list the fact that interactive programs, like SSH or VIM work partialy.  
The config has to be manualy created and populated.  
Also pipes aren't supported yet, so no ls | grep "thing"  
PTY currently don't support signals like 'Ctrl+C' so don't use vim, nano nor nvim for exemple.  
