# Autoscout
A Recon Server with several other functionalities

currently the web server can perform search for subdomains and send it to your desired app using notify
modify and copy the [provider config](https://github.com/projectdiscovery/notify#provider-config) to `$HOME/.config/notify/provider-config.yaml`

### Installation
you can install this from command line using go install -v github.com/HaythmKenway/autoscout@latest

Setup the following webclient [Autoscout-web-client](https://github.com/HaythmKenway/autoscout-client)
The project consists of Automating all the popular bug bounty hunting tool into a single packed framework to make it easy for pentesters

Currently I have planned 3 modes of operation for the application

Deamon mode                  ->  performs the programmed scans on specified intervals
Server mode(depricated)       ->  Opens a Graphql Server which can be accessed by using [Autoscout-web-client](https://github.com/HaythmKenway/autoscout-client) 
Command line operation mode  -> Use `autoscout -h` to view all commands  
SSh Connection               -> By default the ssh server runs at  @ 0.0.0.0:2222
### Project Milestone
- [x] Subdomain Enumeration
- [x] Notifying New targets to discord server
- [ ] Validating and Processing all the urls
- [ ] Validating Ports on Identified servers
.... much more comming soon
