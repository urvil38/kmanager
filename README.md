# kmanager

Kmanager is a Command Line Tool(CLI) to help manage a kubernetes cluster and other cloud resources on GCP to host webapp using [KubePAAS](https://github.com/urvil38/kubepaas).

#### Requirements:

- [gcloud](https://cloud.google.com/sdk/docs/install)
- [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl)

# Usage

```

     __
    / /______ ___  ____ _____  ____ _____ ____  _____
   / //_/ __ '__ \/ __ '/ __ \/ __ '/ __ '/ _ \/ ___/
  / ,< / / / / / / /_/ / / / / /_/ / /_/ /  __/ /
 /_/|_/_/ /_/ /_/\__,_/_/ /_/\__,_/\__, /\___/_/
                                  /____/

Cluster Manager of KubePAAS platform

Usage:
  kmanager [flags]
  kmanager [command]

Available Commands:
  create      Create a new kubepaas cluster
  delete      delete will delete the cluster of given name
  describe    describe print out configuration of given cluster
  help        Help about any command
  list        List cluster managed by kmanager

Flags:
  -h, --help   help for kmanager

Use "kmanager [command] --help" for more information about a command.
```


# Download

- Download appropriate pre-compiled binary from the [release](https://github.com/urvil38/kmanager/releases) page.

```
# download binary using cURL
$ curl -L https://github.com/urvil38/kmanager/releases/download/0.0.1/kmanager-darwin-amd64 -o kmanager

# make binary executable
$ chmod +x ./kmanager

# move it to bin dir (user need to has root privileges. run following command as root user using sudo.
$ sudo mv ./kmanager /usr/local/bin
```


- Download using `go get`

```
$ go get -u github.com/urvil38/kmanager
```

# Build

- If you want to build kmanager right away, you need a working [Go environment](https://golang.org/doc/install). It requires Go version 1.12 and above.

```
$ git clone https://github.com/urvil38/kmanager.git
$ cd kmanager
$ make build
```
