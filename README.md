# AWS Server

This project provides an HTTP server using Go, which interacts with AWS Services. This project is useful for running a sidecar, to use original images without a hassle.

## Features

- Fetch decrypted parameters from AWS SSM.
- Put parameters to AWS SSM.
- Fetch secrets from AWS Secrets Manager.
- Fetch and serve files from AWS S3.
- Upload files to AWS S3.
- Fetch ECR authorization token.
- Fetch caller identity from AWS STS.
- CI/CD pipeline using GitHub Actions for automatic builds, tests, and container image publishing.

## Requirements

- Go 1.26 or later
- AWS credentials configured (e.g. `~/.aws/credentials`, environment variables, or IAM role)
- Git installed

## Container Image

```sh
docker pull ghcr.io/leneffets/awsserver:v2.0.0
docker pull ghcr.io/leneffets/awsserver:latest
```

The container image uses a `scratch` base (zero OS packages), runs as a non-root user, and is available for `linux/amd64` and `linux/arm64`.

## Setup

1. **Clone the repository:**

    ```sh
    git clone git@github.com:leneffets/awsserver.git
    cd awsserver
    ```

2. **Install dependencies:**

    ```sh
    go mod tidy
    ```

## Running the Server

The server binds to `0.0.0.0` by default. To start it:

```sh
# Port may be changed via environment variable, default 3000
export PORT=3000

# Bind address may be changed, default 0.0.0.0
export BIND_ADDRESS=127.0.0.1

go run cmd/server/main.go
```

The server shuts down gracefully on SIGINT/SIGTERM, finishing in-flight requests before stopping.

## Endpoints

### Health Check

- **URL:** `/healthz`
- **Method:** `GET`
- **Example:**

    ```sh
    curl "http://localhost:3000/healthz"
    ```

### Fetch SSM Parameter

- **URL:** `/ssm`
- **Method:** `GET`
- **Query Parameters:**
  - `name`: Name of the SSM parameter to fetch.
- **Example:**

    ```sh
    curl "http://localhost:3000/ssm?name=example_parameter"
    ```

### Put SSM Parameter

- **URL:** `/ssm`
- **Method:** `POST`
- **Form Parameters:**
  - `name`: Name of the SSM parameter.
  - `value`: Value of the SSM parameter.
  - `type`: Type of the SSM parameter (`String` or `SecureString`).
- **Example:**

    ```sh
    curl -X POST -d "name=/path/to/parameter&value=somevalue&type=String" http://localhost:3000/ssm
    ```

### Fetch Secret from Secrets Manager

- **URL:** `/secrets`
- **Method:** `GET`
- **Query Parameters:**
  - `name`: Name or ARN of the secret to fetch.
- **Example:**

    ```sh
    curl "http://localhost:3000/secrets?name=my-app/db-credentials"
    ```

### Fetch S3 File

- **URL:** `/s3`
- **Method:** `GET`
- **Query Parameters:**
  - `bucket`: Name of the S3 bucket.
  - `key`: Key of the file in the S3 bucket.
- **Example:**

    ```sh
    curl "http://localhost:3000/s3?bucket=example-bucket&key=example-key"
    ```

### Upload S3 File

- **URL:** `/s3`
- **Method:** `POST`
- **Query Parameters:**
  - `bucket`: Name of the S3 bucket.
  - `key`: Key of the file in the S3 bucket.
- **Example:**

    ```sh
    curl -X POST -F 'file=@/path/to/your/file' "http://localhost:3000/s3?bucket=example-bucket&key=example-key"
    ```

### Get ECR Login

- **URL:** `/ecr/login`
- **Method:** `GET`
- **Example:**

    ```sh
    curl "http://localhost:3000/ecr/login" | docker login --username AWS --password-stdin <account-id>.dkr.ecr.<region>.amazonaws.com
    ```

### Get Caller Identity

- **URL:** `/sts`
- **Method:** `GET`
- **Example:**

    ```sh
    curl "http://localhost:3000/sts"
    ```

## Usage as a GitLab CI Sidecar

This server is designed to run as a sidecar service in GitLab CI, giving your jobs easy access to AWS services without installing the AWS CLI.

```yaml
my-job:
  image: alpine:latest
  services:
    - name: ghcr.io/leneffets/awsserver:latest
      alias: awsserver
  variables:
    AWS_REGION: eu-central-1
  script:
    - SECRET=$(curl -s "http://awsserver:3000/ssm?name=/my-app/db-password")
    - echo "Fetched secret successfully"
```

> **Note:** When running as a GitLab CI service, the server is reachable via the `alias` hostname (here `awsserver`) on port 3000. Make sure your AWS credentials are set as [CI/CD variables](https://docs.gitlab.com/ee/ci/variables/) in your project or group settings.

## Running Tests

```sh
go test -v ./...
```

## CI/CD Pipeline

This project uses GitHub Actions with two workflows:

- **CI Pipeline** (`.github/workflows/ci.yml`): Runs on pushes and PRs to `main`. Checks out code, runs tests, builds a static binary, and pushes a Docker image (`:latest`) on pushes to `main`.
- **Release Pipeline** (`.github/workflows/release.yml`): Runs on published releases. Builds static binaries for linux/darwin (amd64/arm64), uploads them as release artifacts (`.tar.gz`), and pushes a multi-arch Docker image tagged with the release version.

## Contribution

Feel free to fork this repository and create pull requests. For major changes, please open an issue first to discuss what you would like to change.

## License

This project is licensed under the MIT License.
