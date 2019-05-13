# Blatt 3
## Ausführen ohne Docker
-   Tree-Service starten
    ```
    treeservice -bind localhost:8090
    ```
-   Baum erstellen mit Blättergröße 3
    ```
    treecli -bind localhost:8091 -remote localhost:8090 create 3
    ```
-   Element (2, "zwei") einfügen
    ```
    treecli -bind localhost:8091 -remote localhost:8090 -id 1 -token 421337 insert 2 zwei
    ```
-   Element mit Schlüssel 2 suchen
    ```
    treecli -bind localhost:8091 -remote localhost:8090 -id 1 -token 421337 search 2
    ```
-   Element mit Schlüssel 2 löschen
    ```
    treecli -bind localhost:8091 -remote localhost:8090 -id 1 -token 421337 deleteitem 2
    ```
-   Sortierte Elemente des Baumes ausgeben
    ```
    treecli -bind localhost:8091 -remote localhost:8090 -id 1 -token 421337 traverse
    ```
-   Baum löschen
    ```
    treecli -bind localhost:8091 -remote localhost:8090 -id 1 -token 421337 deletetree
    ```

## Ausführen mit Docker
### Docker-Images bauen und ausführen

-   Images bauen
    ```
    make docker
    ```
-   ein (Docker)-Netzwerk `actors` erzeugen
    ```
    docker network create actors
    ```
-   Starten des Tree-Services und binden an den Port 8090 des Containers mit dem DNS-Namen
    `treeservice` (entspricht dem Argument von `--name`) im Netzwerk `actors`:

    ```
    docker run --rm --net actors --name treeservice treeservice \
      -bind treeservice.actors:8090
    ```

-   Starten des Tree-CLI, Binden an `treecli.actors:8091` und nutzen des Services unter
    dem Namen und Port `treeservice.actors:8090`:
    ```
    docker run --rm --net actors --name treecli treecli -bind treecli.actors:8091 \
      -remote treeservice.actors:8090 create
    ```
-   Der Tree-Service-Container lässt sich mittels `Ctrl-C` killen
-   Das Netzwerk lässt sich löschen mit
    ```
    docker network rm actors
    ```

### Ausführen mit Docker ohne vorher die Docker-Images zu bauen

-   Herunterladen der Images
    ```
    docker pull terraform.cs.hm.edu:5043/ob-vss-ss19-blatt-3-forever_alone:develop-treeservice
    docker pull terraform.cs.hm.edu:5043/ob-vss-ss19-blatt-3-forever_alone:develop-treecli
    ```
-   Image-IDs herausfinden mit
    ``` 
    docker images
    ```
-   Ausführen analog zu "Docker-Images bauen und ausführen"

## Dokumentation
### tree
-   NodeActors haben zwei unterschiedliche Verhaltensweisen:
    -   Blatt
    -   Innerer Knoten
    
#### Verhaltensweise als Blatt
-   Wird mit der Angabe der Maximalgröße initialisiert
-   Nimmt bei Insert Schlüssel-Wert-Paare entgegen, wenn es den übergebenen Schlüssel noch nicht gibt, ansonsten Fehler
-   Löscht bei Delete Schlüssel-Wert-Paar, wenn es den übergebenen Schlüssel enthält, ansonsten Fehler
-   Gibt bei Search Schlüssel-Wert-Paar zurück, wenn es den übergebenen Schlüssel enthält, ansonsten Fehler
-   Gibt bei Traverse seine Schlüssel-Wert-Paare sortiert nach Schlüssel zurück
-   Wenn die Maximalgröße überschritten wird, initialisiert der Aktor zwei neue Blätter mit seiner Maximalgröße, 
    teilt gleichmäßig die Menge seiner sortierten Schlüssel-Wert-Paare auf und schickt die beiden Hälften an die Kinder 
    und wird zu einem inneren Knoten.

#### Verhaltensweise als innerer Knoten
-   Leitet Inserts, Searches und Deletes anhand ihrer jeweiligen Schlüssel an das passende Kind weiter
-   Sendet bei einem Traverse eigene TraverseRequests an seine Kinder und gibt das verschmolzene Ergebnis der beiden 
    Kinder zurück
-   Beim Beenden werden auch die beiden Kinder beendet

### treeservice
#### Funktionsweise des Services
-   Nimmt Nachrichten von treecli entgegen
-   Verwaltet Bäume(PIDs der Wurzelaktoren) mitsamt ihrer IDs und Tokens
-   Prüft ID und Token von eingehenden Nachrichten und leitet diese an jeweiligen Baum weiter, bei passendem Token
-   Wartet nicht auf Antwort von Bäumen, sondern kann direkt neue Anfragen entgegen nehmen 

#### Benutzung des Services
-   Treeservice starten über `treeservice -bind [addr]`
-   Ausgabe von `treeservice help`:
    ```
    NAME:
       treeservice - proto.actor service for managing search trees
    
    USAGE:
       treeservice [global options] command [arguments...]
    
    VERSION:
       1.0.0
    
    AUTHOR:
       Dimitri Krivoj <krivoj@hm.edu>
    
    COMMANDS:
         help, h  Shows a list of commands or help for one command
    
    GLOBAL OPTIONS:
       --bind value   the treeservice will listen on this address (default: "localhost:8090")
       --help, -h     show help
       --version, -v  print the version
    ```

### treecli
#### Benutzung des CLI
-   Ausgabe von `treecli help`
    ```
    NAME:
       treecli - proto.actor client for treeservice
    
    USAGE:
       treecli [global options] command [arguments...]
    
    VERSION:
       1.0.0
    
    AUTHOR:
       Dimitri Krivoj <krivoj@hm.edu>
    
    COMMANDS:
         create      create a new search tree
         insert      insert key-value pair into tree
         search      search value specified by key in tree
         deleteitem  delete key-value pair in tree
         traverse    get all key-value pairs sorted by key
         deletetree  remove tree from treeservice
         help, h     Shows a list of commands or help for one command
    
    GLOBAL OPTIONS:
       --bind value    address treecli should use (default: "localhost:8091")
       --remote value  address of the treeservice (default: "localhost:8090")
       --id value      id of the tree you want to alter (default: 0)
       --token value   token to authorize your access for the specified tree
       --help, -h      show help
       --version, -v   print the version
    ```
-   Ausgabe von `treecli help create`:
    ```
    NAME:
       create - create a new search tree
    
    USAGE:
       create [maxSize=2]
    
    DESCRIPTION:
       Create a new search tree with the specified maximum size for its leafs (default 2). Outputs id and token of the created tree.
    ```
-   Ausgabe von `treecli help insert`:
    ```
    NAME:
       insert - insert key-value pair into tree
    
    USAGE:
       insert key value
    
    DESCRIPTION:
       Inserts new key-value pair into specified tree. Outputs key-value pair on success.
       Fails if the specified tree doesn't exist or if an invalid token is provided.
       Also fails if the specified key already exists. In this case the existing key-value pair will be printed.
    ```
-   Ausgabe von `treecli help search`:
    ```
    NAME:
       search - search value specified by key in tree
    
    USAGE:
       search key
    
    DESCRIPTION:
       Searches value specified by key in specified tree. Outputs key-value pair if found.
       Fails if the specified tree doesn't exist or if an invalid token is provided.
       Also fails if the specified key doesn't exist.
    ```
-   Ausgabe von `treecli help deleteitem`:
    ```
    NAME:
       deleteitem - delete key-value pair in tree
    
    USAGE:
       deleteitem key
    
    DESCRIPTION:
       Deletes key-value pair specified by key in specified tree. Outputs deleted key-value pair on success.
       Fails if the specified tree doesn't exist or if an invalid token is provided.
       Also fails if the specified key doesn't exist.
    ```
-   Ausgabe von `treecli help traverse`:
    ``` 
    NAME:
       traverse - get all key-value pairs sorted by key
    
    USAGE:
       traverse [arguments...]
    
    DESCRIPTION:
       Gets all key-value pairs in specified tree sorted by keys.
       Fails if the specified tree doesn't exist or if an invalid token is provided.
    ```
-   Ausgabe von `treecli help deletetree`:
    ``` 
    NAME:
       deletetree - remove tree from treeservice
    
    USAGE:
       deletetree [arguments...]
    
    DESCRIPTION:
       Removes specified tree. Asks for confirmation by repeating the token.
       Fails if the specified tree doesn't exist or if an invalid token is provided.
    ```
