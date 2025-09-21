# Why This Exists

Built for working in large codebases that require multiple terminal coding agents operating simultaneously. Provides oversight and control over agent interactions.

# Inter-Agent Communication

Lots of frameworks are coming out these days to facilitate inter-agent comms - A2A, sub-agents, using claude code/opencode/gemini CLI commands or even SKD's. My issue with all of them is the agents talk in the background and I cant see what they are saying to each other. 

The idea behind the msg command that the agents use is to directly send keys to the other ai coding tools instance, pressing enter, and engagin in raw agent communication. 

# I chose TMUX because;
 
1. Im newer to the terminal and somebody showed me TMUX right away. 
2. Rich Ecosystem for plugins. 
3. Rich commands provided natively see https://tmuxcheatsheet.com 

I'm sure other terminal multiplexers have send-keys functionality and work great - use them - write your own for it - Im happy to take contributions and would be nice if folks could do this on windows too. 

# I have some other tui's worth checking out if you like this:

AGENTDL: Search for Claude.md's on GH and download them to your current config. 

brew install williavs/tap/agentdl

# Ideas for future versions

- Need this to work with itmux for windows
- Need to implement agent groups/projects for agents that are predisposd to collaborate
- Add new TMUX project with agent setup? 
    - Would be simple to just pack this with a bunch of the bash scripts Ive written. 
    - Could create a sort of homebase for folks to add claude code plugins and sync their environments, become more organized. 
- SSH detect to register and communciate with agents in different machines on same network 
    - Setup SSH into new machine, detect TMUX panes over there, register agents, write msg-ssh.go script to handle inter-mchine agent comms


## Author

WillyV3

## Links

- GitHub: https://github.com/williavs
- Website: https://willyv3.com
- Blog: https://breakshit.blog