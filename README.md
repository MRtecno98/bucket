# Bucket

Bucket is a tool for managing Bukkit Minecraft servers, it allows you to keep track of
plugins, updates and config files and synchronizes your changes across multiple servers,
even on different machines.

Development is currently ongoing

# Feature plan
- [X] Workspace system
	- [X] Remote workspaces
- [X] Platform and plugin detection
- [ ] Retrieve plugins
	- [X] SpigotMC web scraping for plugins
	- [X] Modrinth API integration
	- [ ] Custom repository protocol
- [ ] Download and install plugins
- [X] Resolve local plugins **[WIP]**
	- [X] Differential confidence check
	- [X] Resolution caching
- [ ] Auto update plugins
- [ ] Update/switch server jar
- [ ] Backup worlds and configs
	- [ ] Package servers and configs
