# go-rest-boilerplate

A great starting point for building RESTful APIs in Go using Gin framework, and sqlx for connecting to a PostgreSQL database. The implementation follows Clean Architecture principles as described by Uncle Bob.

### Features

-   Implements the Clean Architecture pattern for a scalable and maintainable codebase
-   Uses the Gin framework for efficient and fast handling of HTTP requests
-   Integrates with PostgreSQL databases using SQLx.DB for powerful and flexible database operations

##### Authentication

-   Supports JWT authentication with configurable expiration and issuer, allowing for flexible and secure authentication processes.
-   Supports OTP authentication with configurable expiration and Redis caching to store and retrieve the OTP codes, providing fast and efficient authentication processes.

### Getting Started

##### Prerequisites

-   Go version 1.17 or higher
-   PostgreSQL version 9.1 or higher

To get up and running with the Go-REST-Boilerplate, follow these simple steps:

```
$ git clone https://github.com/snykk/go-rest-boilerplate.git
$ cd go-rest-boilerplate
$ cp internal/config/.env.example internal/config/.env # create a copy of the example environment file, and also follow configuration steps on the difference section below
$ go build -o go-rest-boilerplate.out cmd/api/main.go
$ ./go-rest-boilerplate.out
```

#### Configuration

The application can be configured using environment variables to fit your specific needs. A sample environment file is provided as .env.example with the following variables available for customization:

##### App

-   `PORT`: The port on which the server will listen (defaults to 8080)
-   `ENVIRONMENT`: The environment the application is running in (defaults to "development")
-   `DEBUG`: Enable or disable debug mode (defaults to true)

##### Database

PostgreSQL

-   `DB_POSTGRE_DRIVER`: The database driver to use (defaults to "postgres")
-   `DB_POSTGRE_DSN`: The database connection URI in DSN format (defaults to "user=myuser password=mypassword host=myhost port=5432 dbname=mydb sslmode=disable timezone=Asia/Jakarta")
-   `DB_POSTGRE_URL`: The database connection URI in URL format (defaults to "postgres://user:pass@host/db")

##### JWT

-   `JWT_SECRET`: The secret key used to sign and verify JWT tokens (defaults to "dont-tuch-mytralalala-mydangdingdong")
-   `JWT_EXPIRED`: The number of hours until JWT tokens expire (defaults to 5)
-   `JWT_ISSUER`: The issuer of JWT tokens (defaults to "snykk_here")

##### OTP

-   `OTP_EMAIL`: The email address to send OTP codes to (defaults to "patrick@gmail.com")
-   `OTP_PASSWORD`: The password to use for sending OTP codes (defaults to "idonthavepassword")

##### Cache

Redis

-   `REDIS_HOST`: The host and port of the Redis server (defaults to "localhost:6969")
-   `REDIS_PASS`: The password to use for connecting to Redis (defaults to "mydangdingdong")
-   `REDIS_EXPIRED`: The number of minutes until cache items expire in Redis (defaults to 5)

### Folder Structure

```
root/
|-- cmd/
|   |-- api
|   |   |-- server/
|   |   |-- main.go
|   |-- cron/
|   |   |-- jobs/
|   |   |-- main.go
|   |-- migration/
|   |   |-- migrations/
|   |   |-- main.go
|   |-- seed
|       |-- seeds/
|       |-- main.go
|-- deploy/
|   |-- Dockerfile
|   |-- docker-compose.yml
|-- docs/
|   |-- swagger.yaml
|-- internal/
|   |-- business/
|   |   |-- domains
|   |   |   |-- v1
|   |   |       |-- domains.users.go
|   |   |-- usecases
|   |       |-- v1
|   |           |-- usecase.users.go
|   |           |-- usecase.users_test.go
|   |-- config/
|   |   |-- .env
|   |   |-- .env.example
|   |   |-- config.go
|   |-- constants/
|   |   |-- constant.users.go
|   |-- datasources/
|   |   |-- caches/
|   |   |   |-- cache.redis.go
|   |   |-- drivers/
|   |   |   |-- driver.postgre.go
|   |   |-- records/
|   |   |   |-- record.user.go
|   |   |   |-- record.user_mapper_v1.go
|   |   |-- repositories
|   |   |   |-- postgres/
|   |   |   |   |-- v1
|   |   |   |       |-- postgre.user.go
|   |   |   |-- mongos/
|   |-- http/
|   |   |-- datatransfers/
|   |   |   |-- requests/
|   |   |   |   |-- request.users.go
|   |   |   |-- responses/
|   |   |       |-- response.users.go
|   |   |-- handlers/
|   |   |   |-- v1/
|   |   |       |-- handler.base_response.go
|   |   |       |-- handler.users.go
|   |   |       |-- handler.users_test.go
|   |   |-- middlewares/
|   |   |   |-- middleware.auth.go
|   |   |   |-- middleware.auth_test.go
|   |   |-- routes/
|   |       |-- route.users.go
|   |-- mocks/
|   |   |-- mock.cache_redis.go
|   |-- utils/
|-- pkg/
|   |-- helpers/
|   |   |-- helper.bcrypt.go
|   |   |-- helper.bcrypt_test.go
|   |-- jwt/
|   |   |-- jwt.go
|   |   |-- jwt_test.go
|   |-- logger/
|   |-- mailer/
|   |-- validators/
|-- vendor/
|-- go.mod
|-- go.sum
|-- makefile
|-- README.md (thisfile)
|-- rest.http
```

##### `cmd` folder

This folder contains all the entry points of the application. There are four sub-folders in the `cmd` folder:

-   `api`: This folder contains the main entry point of the REST API server. The `main.go` file in the `server` sub-folder is responsible for starting the server and setting up all the necessary routes.

-   `cron`: This folder contains the main entry point for any cron jobs that need to be run.

-   `migration`: This folder contains the main entry point for managing database migrations.

-   `seed`: This folder contains the main entry point for seed data into the database.

##### `deploy` folder

This folder contains the necessary configuration files for deploying the application to a production environment.

-   `Dockerfile`: This file is used to build a Docker image of the application.

-   `docker-compose.yml`: This file is used to start the application and its dependencies (such as the database) using Docker Compose.

##### `docs` folder

This folder contains the documentation for the REST API, including the `swagger.yaml` file which defines the API specification.

##### `internal` folder

This folder contains all the business logic and other implementation details of the application. It is structured as follows:

-   `business` folder

    -   domains folder: This folder contains domain-specific logic, such as the business rules for creating, updating, and deleting users.

    -   usecases folder: This folder contains the implementation of the use cases that are defined in the domains folder.

-   `config` folder

    -   `.env`: This file contains the environment variables that are used by the application.
    -   `.env.example`: This file is an example of the .env file, with all the necessary environment variables listed.
    -   `config.go`: This file reads the environment variables and sets up the configuration for the application.

-   `constants` folder

    -   this folder contains constant values used throughout the application.

-   `datasources` folder

    -   `caches` folder: This folder contains the implementation of cache storage, such as Redis.
    -   `drivers` folder: This folder contains the implementation of database drivers, such as PostgreSQL.
    -   `records` folder: This folder contains the implementation of database records, such as User.
    -   `repositories` folder: This folder contains the implementation of database repositories, such as PostgreSQL and MongoDB.

-   `http` folder

    -   `datatransfers` folder: This folder contains the implementation of data transfer objects, such as request and response objects.
    -   `handlers` folder: This folder contains the implementation of HTTP handlers, which handle incoming HTTP requests and send responses back to the client.
    -   `middlewares` folder: This folder contains the implementation of middlewares, which are executed before the request is handled by the handler.
    -   `routes` folder: This folder contains the implementation of routes, which map URLs to handlers.

-   `mocks` folder

    -   this folder contains the implementation of mock objects used in tests.

-   `utils` folder

    -   this folder contains utility functions and classes used throughout the application.

##### `pkg` folder

This folder contains reusable packages that are shared across different parts of the application.

### Contributing

This project is open for contributions and suggestions. If you have an idea for a new feature or a bug fix, don't hesitate to open a pull request

### License

This project is licensed under the MIT License. See the [LICENSE](https://github.com/snykk/go-rest-boilerplate/blob/master/LICENSE) file for details.
