# hasher 

hasher is an app that listens on a default port of 8080. It has the following command-line parameters:
```
-h  This help message
-d	Enable debug output (shorthand)
-debug
  	Enable debug output
-hash_wait int
  	Time in seconds to wait for hash to be computed (default 5)
-hw int
  	Time in seconds to wait for hash to be computed (shorthand) (default 5)
-p int
  	Port number for HTTP listener (shorthand) (default 8080)
-port int
  	Port number for HTTP listener (default 8080)
```
### Shutdown
Once running, hasher can be gracefully shutdown from the shell in which was launched via ctrl-c. It can also be shutdown by sending it the SIGINT (-2) or SIGTERM (-15) signals. For example:
```
kill -2 <pid>
```
or
```
kill -15 <pid>
```
### Clone
Clone the repo:
```
git clone https://github.com/mattdefoor/jc.git && cd jc/hasher
```
### Build
```
go build
```
### Run
```
.\hasher
```
### Run Tests
```
go test -v
```
