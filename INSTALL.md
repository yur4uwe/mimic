## Mimic installation guide

1. Download the corresponding to your system acrchive from https://github.com/yur4uwe/mimic/releases and extract to your desired location.
2. Run install script from the extracted folder inside elevated shell 
3. It should walk you through the installation. 

### What does the script do?
On Windows
1. Installs [WinFSP](https://github.com/winfsp/winfsp) on which it depends
2. Adds the binary location to path 
3. Puts config inside %APPDATA%\mimic folder

On Linux
1. Puts binary inside `/usr/local/bin`
2. Puts config inside `/etc/mimic`

Ask questions or report issues at https://github.com/yur4uwe/mimic/issues