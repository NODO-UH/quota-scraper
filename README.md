# quota-scraper

## Usage

- `-cores int`: max number of cores (default: number of cores of the host)
- `-cut-file string`: file to insert over quota users (default "cut.list")
- `-db-uri string`: MongoDB Connection URI
- `-file string`: Path to squid file with logs (default "squid.logs")
- `-id string`: unique id between all quota-scraper instances (default: hostname)
- `-logs string`: path to file for logs (default "squid-parser.logs")
- `-reload string`: script for reload Squid service (default "reload.sh")
