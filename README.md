# GoSh!
A Shell made in Go, for fun  
## Features
- Display Working Directory
- History file wich you can browse for past commands
- Some colors
- Config file
I'm planning to add more features later (like syntax hilighting, etc...).  
I'm also learning Go by doing this project.  
You can expect breaking changes (or code), the problem has a workaround in the commit message.  
If not, the issue will be fixed soon (in case of a typo for example)  

## Installation
You can grab the binaries for your system and architecture, or build it yourself
To build it, clone the repository, cd into it and run  
'go build -o bin/GoSH'  
To test the shell and see if it suits you.  
If everything works, then move the binary to a place inside your path.  

To directly install it with the Go toolchain, just use 'go intall'  
This will build and place the binary inside your $HOME/go/bin folder.  
Add this folder to your path and you are good to go !  

## Usage
To use the program, just invoke it with 'GoSH'  
If you see a message about a config file, create '~/.config/gosh/gosh_config.toml' and populate it with the defaults written inside this repo.  
To change config parameter on the fly, use the 'set' builtin.  
Currently, 'set' has a limited amount of configuration options.  
To change the color of the prompt use 'set color <color>'  
You can use all "console colors", listed [https://gist.github.com/kamito/704813 | here]  

## Know Issues
Currently there is a number of known or unkwown issues.  
We can list the fact that interactive programs, like SSH or VIM won't work.  
The config has to be manualy created and populated.  
In the handling of flags and strings, the program currently cut all spaces, so no strings in flags.  
For exemple for git commit -a -m "Some light modifications", the shell will think that "git commit -a -m 'Some'" is a command and "light" and "modifications" are other commands.  
Also pipes aren't supported yet, so no ls | grep "thing"  

