# Basic Example

The only requirement for this example is [Docker](https://docs.docker.com/install/).

## Docker Compose

The Docker Compose is pre-installed with Docker for Windows. For linux please follow the steps described at: https://docs.docker.com/compose/install/.

Build and run the application for linux arch and expose it on http://localhost:8080.

```sh
$ docker-compose up
```

See [docker-compose file](docker-compose.yml).

## Without Docker Compose

1. Build the image as "myapp" (docker build)
2. Run the image and map exposed ports (-p 8080:8080)
3. Attach the interactive mode so CTRL/CMD+C signals are respected to shutdown the Iris Server (-it)
4. Cleanup the image on finish (--rm)

```sh
$ docker build -t myapp . 
$ docker run --rm -it -p 8080:8080 myapp:latest
```
