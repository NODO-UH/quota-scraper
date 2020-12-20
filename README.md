# quota-scraper

## Description

Quota Scraper is a tool developed with the aim of parsing the logs left by the Squid proxy in real time. In addition to taking the accumulated consumption during the month for each user, and blocking accounts that exceed the assigned monthly consumption.

## Table of Contents

- [quota-scraper](#quota-scraper)
  - [Description](#description)
  - [Table of Contents](#table-of-contents)
  - [Installation](#installation)
    - [Compiling](#compiling)
    - [Release](#release)
  - [Usage](#usage)
  - [Contributing](#contributing)

## Installation

Quota Scraper is developed in the Go language, so one of the ways to install it is by compiling the source code, or by downloading any of the previously compiled releases.

### Compiling

The only requirement to compile the code is to have Go. Once the repository is cloned and all the dependencies are in the `GOPATH`, it can be compiled from the file in`src/main.go`.

### Release

If you want to download a compilation directly, simply go to the releases in this repository and choose the desired version. If you can't find support for a certain architecture, feel free to open an issue to let us know.

## Usage

Once we have the binary, let's call it `quota-scraper.bin`, let's see how to use it. The configuration can be passed through a JSON file, or through the flags. The configuration consists of the following parameters:

- `config`: path to the configuration file. The path can be absolute or relative to the location of the binary.
- `file`: required parameter, since it is the path to Squid's log file.
- `db-uri`: MongoDB URI to connect with the database and be able to save the records, accumulated by users, the status of Quota Scraper and free sites.
- `cores`: Go provides the option to configure the maximum number of cores that the program will use during execution. Because this tool is designed to run on a server and in real time, it is convenient to be able to decide this parameter, to get the most out of Go's parallelism and concurrency, increasing performance.
- `logs`: is the path to the file where Quota Scraper will leave its logs during execution.
- `id`: is the unique identifier for Quota Scraper. The uniqueness of this identifier depends only on the configuration. It is the administrator's responsibility that multiple Quota Scraper instances do not have the same identifiers. It is used to be able to differentiate the states of each Quota Scraper in the database.
- `cut-file`: file in which the users who exceed the monthly consumption are added. This file must be the same file that Squid reads, this is how Quota Scraper informs Squid users that they can no longer consume more.
- `reload`: path to the `*.sh` file that is run with the intention of reloading the Squid service. It is used every time the user list in `cut-file` is modified by Quota Scraper.

The configuration JSON file must have the following structure:

```json
{
  "squidFile": "path/to/squid/file",  // file
  "dbUri": "MongoDB_URI",             // db-uri
  "cores": "Max number of cores",     // cores
  "scraperId": "Unic identifier",     // id
  "curFile": "path/to/cut/file",      // cut-file
  "reloadSquid": "path/to/reload.sh"  // reload
}
```

> *Note that in the configuration you cannot put the path to the Quota Scraper log file, this is because if there is an error taking the configuration, then you could not put those logs in the desired place.

## Contributing

To participate in this project you can suggest changes through Issues, we will be happy to implement them (as long as we see fit). Or you can create a PR with the changes you proposed, if it doesn't interfere with the current operation, I don't see why we shouldn't accept it :).
